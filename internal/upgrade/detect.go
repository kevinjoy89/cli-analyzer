// Package upgrade 提供被扫描 CLI 工具的版本检测与官方升级（change
// add-tool-upgrade）。与 internal/updater（应用自身更新）严格分离：
// 本包检测走包管理器自身的查询命令（镜像友好），无缓存、无状态。
package upgrade

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cli-analyzer/internal/cmdexec"
	"cli-analyzer/internal/scanner"
)

// commandFunc 抽象检测查询的子进程执行：返回合并输出（stdout+stderr）与错误。
// 包级变量便于测试注入假实现（镜像 updater 的 dpkgCommand 模式）。
type commandFunc func(ctx context.Context, bin string, args ...string) (string, error)

var runQuery commandFunc = defaultRunQuery

// defaultRunQuery 经（增强的）PATH 解析命令绝对路径并注入完整 PATH 执行：
// GUI 从 Finder 启动时进程 PATH 是系统最小集，裸 exec "brew" 会失败；增强
// 目录补齐 shell-only 安装位置（与 uninstall 代跑行为一致）。
//
// 返回 stdout 作为可解析负载、stderr 仅随错误附带——brew/npm 等会把警告
// 写在 stderr（如 brew 的 CLT 更新提醒），若合并进 JSON 负载会破坏解析
// （brew outdated / npm outdated 检出更新时退出码为 1，此时 stdout 仍有效）。
func defaultRunQuery(ctx context.Context, bin string, args ...string) (string, error) {
	resolved := bin
	if r, rerr := cmdexec.ResolveCommand(bin); rerr == nil {
		resolved = r
	}
	cmd := exec.CommandContext(ctx, resolved, args...)
	cmd.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil && outBuf.Len() == 0 && strings.TrimSpace(errBuf.String()) != "" {
		// 无有效输出时附带 stderr 便于诊断（%w 保留 *exec.ExitError 供 execError）
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), err
}

// execError 报告 err 是否为命令执行失败（退出码非零）——这类错误输出仍可能
// 有效（brew outdated / npm outdated 检出更新时退出码为 1），只有"命令不存在/
// 无法启动"类错误才应降级为检测失败。
func execError(err error) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee)
}

// Detectable 报告安装来源是否具备版本检测能力（design D2）。
// go/versioned/pyenv/other/local-bin 无可靠 latest 来源 → 仅命令/提示。
func Detectable(i scanner.Installer) bool {
	switch i {
	case scanner.InstBrew, scanner.InstNpm, scanner.InstPipx, scanner.InstCargo:
		return true
	}
	return false
}

// detect 按安装来源分发检测，返回包管理器口径的 (current, latest)。
func detect(ctx context.Context, i scanner.Installer, name string) (string, string, bool, error) {
	switch i {
	case scanner.InstBrew:
		return detectBrew(ctx, name)
	case scanner.InstNpm:
		return detectNpm(ctx, name)
	case scanner.InstPipx:
		return detectPipx(ctx, name)
	case scanner.InstCargo:
		return detectCargo(ctx, name)
	}
	return "", "", false, fmt.Errorf("installer %q has no detection capability", i)
}

// brewOutdated 是 `brew outdated --json=v2 <公式>` 的解析目标。
type brewOutdated struct {
	Formulae []struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	} `json:"formulae"`
}

// detectBrew 单命令检测：`brew outdated --json=v2 <公式>`。
// 空 formulae = 已最新；非空 = current 与 latest（current_version）一次给全。
// brew 检出更新时退出码为 1，但输出有效。
// installed_versions 是「过时 keg」列表，brew 源码按 scheme_and_version 升序
// 排序（旧→新）。取最后一个（最新）作为 current，避免多 keg 时展示成最旧版本
// （如 fontconfig 装了 2.17.1+2.18.1、最新 2.18.3 时，应展示 2.18.1 而非 2.17.1）。
func detectBrew(ctx context.Context, formula string) (string, string, bool, error) {
	out, err := runQuery(ctx, "brew", "outdated", "--json=v2", formula)
	if err != nil && !execError(err) {
		return "", "", false, fmt.Errorf("brew outdated: %w", err)
	}
	var v brewOutdated
	if jerr := json.Unmarshal([]byte(out), &v); jerr != nil {
		return "", "", false, fmt.Errorf("brew outdated: parse: %w", jerr)
	}
	if len(v.Formulae) == 0 {
		return "", "", true, nil // 已最新
	}
	f := v.Formulae[0]
	cur := ""
	if n := len(f.InstalledVersions); n > 0 {
		cur = f.InstalledVersions[n-1] // 最新过时 keg（brew 升序排列）
	}
	return cur, f.CurrentVersion, true, nil
}

// npmOutdated 是 `npm outdated -g --json <包>` 的解析目标：按包名键控。
type npmOutdated map[string]struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// detectNpm 单命令检测：`npm outdated -g --json <包>`。
// 空输出或空对象 `{}` = 已最新（npm outdated 仅列出过时包；实测最新态
// 输出 `{}`，退出码 0）；非空 JSON 按包名取 current/latest。
// npm 检出更新时退出码为 1，但输出有效。
func detectNpm(ctx context.Context, pkg string) (string, string, bool, error) {
	// pkg 由调用方解析（CheckTool 经 t.Package / NpmPackageFor 得到真实包名）；
	// 本函数只查询，不再启发式改写——假名会静默误报「已最新」。
	out, err := runQuery(ctx, "npm", "outdated", "-g", "--json", pkg)
	if err != nil && !execError(err) {
		return "", "", false, fmt.Errorf("npm outdated: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "{}" {
		return "", "", true, nil // 已最新
	}
	var v npmOutdated
	if jerr := json.Unmarshal([]byte(out), &v); jerr != nil {
		return "", "", false, fmt.Errorf("npm outdated: parse: %w", jerr)
	}
	info, ok := v[pkg]
	if !ok {
		return "", "", false, fmt.Errorf("npm outdated: %q not in output", pkg)
	}
	return info.Current, info.Latest, true, nil
}

// pipxList 是 `pipx list --json` 的解析目标。
type pipxList struct {
	Venvs map[string]struct {
		Metadata struct {
			MainPackage struct {
				Package        string `json:"package"`
				PackageVersion string `json:"package_version"`
			} `json:"main_package"`
		} `json:"metadata"`
	} `json:"venvs"`
}

// detectPipx 双命令检测：`pipx list --json`（installed）+ `pip index versions
// <包>`（latest，走 pip 配置，镜像友好）。
func detectPipx(ctx context.Context, pkg string) (string, string, bool, error) {
	out, err := runQuery(ctx, "pipx", "list", "--json")
	if err != nil {
		return "", "", false, fmt.Errorf("pipx list: %w", err)
	}
	var v pipxList
	if jerr := json.Unmarshal([]byte(out), &v); jerr != nil {
		return "", "", false, fmt.Errorf("pipx list: parse: %w", jerr)
	}
	venv, ok := v.Venvs[pkg]
	if !ok {
		return "", "", false, fmt.Errorf("pipx list: %q not found", pkg)
	}
	cur := venv.Metadata.MainPackage.PackageVersion
	// 查询 latest 用真实 PyPI 发行名（main_package.package），而非 venv 名：
	// pipx 的 venv 名 = 发行名 + 可选 --suffix（如 `pipx install --suffix foo
	// uv` 建 venv `uv-foo`），用 venv 名查 `pip index versions` 会查错包。
	// 与 npm 的「用真实包名」原则一致（code review #6）。
	dist := venv.Metadata.MainPackage.Package
	if dist == "" {
		dist = pkg
	}
	lout, lerr := runQuery(ctx, "pip", "index", "versions", dist)
	if lerr != nil {
		return cur, "", false, fmt.Errorf("pip index versions: %w", lerr)
	}
	latest, ok2 := parsePipIndexVersions(lout)
	if !ok2 {
		return cur, "", false, fmt.Errorf("pip index versions: no parseable output")
	}
	return cur, latest, true, nil
}

// parsePipIndexVersions 解析 `pip index versions <包>` 输出中的
// "Available versions: 1.2.3, 1.2.2, ..." 行，取第一个（最新）版本。
// 前缀解析失败（格式漂移）→ ok=false，调用方按检测失败降级。
func parsePipIndexVersions(out string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		rest, found := strings.CutPrefix(line, "Available versions:")
		if !found {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(rest))
		if len(fields) == 0 {
			return "", false
		}
		v := strings.TrimSuffix(strings.TrimSuffix(fields[0], ","), " ")
		if v == "" {
			return "", false
		}
		return v, true
	}
	return "", false
}

// detectCargo 双命令检测：`cargo install --list`（installed）+ `cargo search
// <crate> --limit 1`（max_version，走 registry 配置，镜像友好）。
// 注意：扫描器按二进制名归类 cargo 工具（如 ripgrep 的二进制 rg），而 cargo
// 按 crate 名寻址——先经二进制列表把工具名映射回 crate 名（code review #6）。
func detectCargo(ctx context.Context, name string) (string, string, bool, error) {
	out, err := runQuery(ctx, "cargo", "install", "--list")
	// 与 brew/npm 一致：退出码非零但输出有效的（execError）不降级，继续解析；
	// 只有命令缺失/无法启动类错误才降级（code review #8）。
	if err != nil && !execError(err) {
		return "", "", false, fmt.Errorf("cargo install --list: %w", err)
	}
	crate, cur := parseCargoInstallList(out, name)
	if cur == "" {
		return "", "", false, fmt.Errorf("cargo install --list: %q not found", name)
	}

	sout, serr := runQuery(ctx, "cargo", "search", crate, "--limit", "1")
	if serr != nil && !execError(serr) {
		return cur, "", false, fmt.Errorf("cargo search: %w", serr)
	}
	latest, ok := parseCargoSearch(sout, crate)
	if !ok {
		return cur, "", false, fmt.Errorf("cargo search: no parseable output")
	}
	return cur, latest, true, nil
}

// parseCargoInstallList 解析 `cargo install --list`，返回包含二进制 binName 的
// crate 的 (crate 名, 已安装版本)。输出形如：
//
//	ripgrep v14.1.1:
//	    /usr/local/bin/rg
//
// 顶层 crate 条目行（无缩进）带版本；其后的缩进行是二进制完整路径。扫描器按
// 二进制名归类工具（rg），故用二进制 basename 反查所属 crate（ripgrep），再以
// crate 名做 cargo search。git 安装的条目带修订后缀（"v0.1.0 (git+…):"），
// 只取版本号本身（code review #6）。
func parseCargoInstallList(out, binName string) (string, string) {
	var crate, ver string
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			// 顶层 crate 条目行："<crate> v<ver>:"
			if idx := strings.Index(trimmed, " v"); idx > 0 && strings.HasSuffix(trimmed, ":") {
				crate = trimmed[:idx]
				rest := trimmed[idx+2 : len(trimmed)-1]
				ver = strings.TrimSpace(rest)
				if i := strings.IndexAny(ver, " ("); i >= 0 {
					ver = ver[:i] // 去掉 git 修订等后缀
				}
			}
			continue
		}
		// 缩进行 = 二进制路径；basename 匹配则命中
		if crate != "" && ver != "" && filepath.Base(trimmed) == binName {
			return crate, ver
		}
	}
	return "", ""
}

// parseCargoInstallList 解析 `cargo install --list` 中形如
// "name v1.2.3:" 的条目行，返回该 crate 的已安装版本。
// parseCargoSearch 解析 `cargo search <名> --limit 1` 中形如
// "name = \"1.2.3\"    # description" 的行，返回 max_version。
func parseCargoSearch(out, name string) (string, bool) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		rest, found := strings.CutPrefix(line, name+" = ")
		if !found {
			continue
		}
		v := strings.Trim(strings.SplitN(rest, " ", 2)[0], "\"")
		if v != "" && v != "\"" {
			return v, true
		}
	}
	return "", false
}
