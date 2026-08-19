package upgrade

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/updater"
)

// 可区分错误：调用方（CLI）据此输出不同文案（spec：cli.noScanResult /
// up.toolNotFound 语义可区分）。
var (
	ErrNoScan       = errors.New("no scan result")
	ErrToolNotFound = errors.New("tool not found")
)

// CheckResult 是单次工具更新检测的完整结果（CLI --json 与 GUI 横幅共用契约）。
type CheckResult struct {
	Name      string `json:"name"`
	Installer string `json:"installer,omitempty"`
	// Current / Latest 均为包管理器口径（design D3），不与 probe 的
	// --version 值交叉比较；"已最新"时可为空（brew/npm 空输出无版本）。
	Current string `json:"current,omitempty"`
	Latest  string `json:"latest,omitempty"`
	// Detected 表示版本检测是否成功（false = 来源无检测能力或查询失败，
	// 横幅显示"无法检测"但仍给升级命令）。
	Detected bool `json:"detected"`
	// HasUpdate 在 Detected 且 latest > current 时为 true。
	HasUpdate bool `json:"hasUpdate"`
	// Command 是官方升级命令或提示；Runnable 表示可代跑。
	Command  string `json:"command,omitempty"`
	Runnable bool   `json:"runnable"`
	// Error 携带检测失败的可读原因（供 CLI 非 JSON 输出与调试；GUI 横幅
	// 只取 detected=false 而不展示细节）。
	Error string `json:"error,omitempty"`
}

// detectTimeout 是单次检测的总时间预算：brew outdated / npm outdated /
// cargo search 均可能触发网络或索引刷新，数秒到数十秒；超预算降级为
// "无法检测"（spec：检测失败降级，不阻塞页面）。
const detectTimeout = 3 * time.Minute

// npmPkgOf 返回 npm 工具的真实全局包名：优先用扫描记录的真实包名
// （t.Package，npmToolID 映射时与短名不同），兼容旧缓存回退逆映射。
// 用短名查询会静默误报「已最新」（见 NpmPackageFor 的碰撞注释）。
func npmPkgOf(t scanner.Tool) string {
	if t.Package != "" {
		return t.Package
	}
	return scanner.NpmPackageFor(strings.TrimSpace(t.Name))
}

// cargoCrateOf 返回 cargo 工具的真实 crate 名：扫描器按二进制名归类工具
// （如 ripgrep 的二进制 rg），而 cargo 按 crate 名寻址——用二进制名跑
// `cargo install <名> --force` 会装错/装不到。经本地 `cargo install --list`
// 的二进制列表反查 crate 名（code review #6）。本地命令、无网络；解析失败
// 回退工具名（同名的 crate 直接可用）。
func cargoCrateOf(t scanner.Tool) string {
	binName := ""
	if len(t.Binaries) > 0 {
		binName = t.Binaries[0].Name
	}
	name := strings.TrimSpace(t.Name)
	if binName == "" {
		return name
	}
	ctx, cancel := context.WithTimeout(context.Background(), detectTimeout)
	defer cancel()
	out, err := runQuery(ctx, "cargo", "install", "--list")
	if err != nil {
		return name
	}
	crate, _ := parseCargoInstallList(out, binName)
	if crate == "" {
		return name
	}
	return crate
}

// CheckTool 对已解析的工具执行检测并组装 CheckResult（无状态，每次全新查询）。
// 调用方（CLI/GUI）负责从扫描结果解析工具与安装来源。
func CheckTool(t scanner.Tool) CheckResult {
	name := strings.TrimSpace(t.Name)
	res := CheckResult{Name: name, Installer: t.Installer}
	binName := ""
	if len(t.Binaries) > 0 {
		binName = t.Binaries[0].Name
	}
	// cmdPkg 供 officialCommand 生成命令；detectName 供 detect 查询。
	// npm：都用真实包名；cargo：命令用 crate 名（ripgrep），检测用二进制名
	// （rg，detectCargo 据此反查 crate）——两者不同（code review #6）。
	cmdPkg := name
	detectName := name
	if t.Installer == string(scanner.InstNpm) {
		cmdPkg = npmPkgOf(t)
		detectName = cmdPkg
	} else if t.Installer == string(scanner.InstCargo) {
		cmdPkg = cargoCrateOf(t)
		detectName = binName
	}
	cmd := officialCommand(scanner.Installer(t.Installer), name, binName, cmdPkg)
	res.Command = cmd.Command
	res.Runnable = cmd.Runnable

	if !Detectable(scanner.Installer(t.Installer)) {
		return res // detected=false，仅命令/提示
	}
	ctx, cancel := context.WithTimeout(context.Background(), detectTimeout)
	defer cancel()
	cur, latest, detected, err := detect(ctx, scanner.Installer(t.Installer), detectName)
	res.Current, res.Latest, res.Detected = cur, latest, detected
	if err != nil {
		res.Error = err.Error()
	}
	if detected && latest != "" {
		res.HasUpdate = newerThan(latest, cur)
	}
	return res
}

// CheckToolByName 从最近的扫描缓存解析工具并检测（CLI 入口）。
// 返回 error 表示缓存缺失或工具不存在（spec：cli.noScanResult /
// un.toolNotFound 语义可区分）。
func CheckToolByName(name string) (CheckResult, error) {
	res, err := scanner.LoadCache()
	if err != nil {
		return CheckResult{Name: name}, fmt.Errorf("%w: %v", ErrNoScan, err)
	}
	for i := range res.Tools {
		t := &res.Tools[i]
		if t.Name == name {
			return CheckTool(*t), nil
		}
		for _, a := range t.Aliases {
			if a == name {
				return CheckTool(*t), nil
			}
		}
	}
	return CheckResult{Name: name}, fmt.Errorf("%w: %q", ErrToolNotFound, name)
}

// OfficialFor 返回工具的标准升级建议（不做版本检测；`update run` 用，避免
// 为已决定升级的用户强制发起网络查询）。
func OfficialFor(t scanner.Tool) Command {
	binName := ""
	if len(t.Binaries) > 0 {
		binName = t.Binaries[0].Name
	}
	pkg := strings.TrimSpace(t.Name)
	if t.Installer == string(scanner.InstNpm) {
		pkg = npmPkgOf(t)
	} else if t.Installer == string(scanner.InstCargo) {
		pkg = cargoCrateOf(t)
	}
	return officialCommand(scanner.Installer(t.Installer), strings.TrimSpace(t.Name), binName, pkg)
}

// newerThan 判定 latest 是否新于 current：两者均可按语义化版本解析时做
// 数值比较；任一解析失败（如 pre-release 后缀）回退字符串不等判定——
// 诚实降级，不猜测数字（design D3：不伪造版本）。
func newerThan(latest, current string) bool {
	lv, le := updater.ParseVersion(latest)
	cv, ce := updater.ParseVersion(current)
	if le == nil && ce == nil {
		return lv.Compare(cv) > 0
	}
	return strings.TrimSpace(latest) != strings.TrimSpace(current)
}
