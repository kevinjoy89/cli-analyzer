package upgrade

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cli-analyzer/internal/cmdexec"
	"cli-analyzer/internal/scanner"
)

// Command 是某个工具的官方升级建议（镜像 uninstall.Official 的结构，
// 但代跑面更窄：local-bin 脚本仅展示，design D4）。
type Command struct {
	// Command 是展示给用户的完整命令（go/local-bin/other 为提示文案）。
	Command string `json:"command"`
	// Runnable 表示能否代为执行（brew/npm/pipx/cargo）。
	Runnable bool `json:"runnable"`
	// Bin / Args 是代跑用的可执行与参数（Runnable 时有效）。
	Bin  string   `json:"-"`
	Args []string `json:"-"`
}

// localBinScripts 是已知官方安装脚本（local-bin 来源）的升级命令：
// 重跑官方脚本即升级。小表、URL 会腐烂（astral 迁移史），未知 local-bin
// 回退通用提示（design D4）。仅展示，不代跑（curl|sh 形态风险等级不同）。
// 注：若工具自带自升级子命令（如 uv self update）会优先走 selfUpgradeSub。
var localBinScripts = map[string]string{
	"poetry": "curl -sSL https://install.python-poetry.org | python3 -",
	"rye":    "curl -sSf https://rye.astral.sh/get | bash",
}

// selfUpgradeSub 是已知「自带自升级子命令」的工具表（二进制名 → 子命令）。
// 这些工具无包管理器来源（versioned/other/local-bin），但 -h 里提供官方
// 自升级方式（claude update / kimi upgrade / codegraph upgrade / uv self update）。
// 静态表优先于 -h 探测：命中即免子进程，且命令形态可人工核对。
var selfUpgradeSub = map[string]string{
	"claude":    "update",
	"kimi":      "upgrade",
	"codegraph": "upgrade",
	"uv":        "self update",
}

// probeTimeout 是 -h 探测的单条命令超时（help 输出应即时；挂起说明该工具
// 不适合探测，不再尝试 --help）。
const probeTimeout = 3 * time.Second

// probeCache 缓存 -h 探测结果，键 = real path + size + mtime（镜像 probe 包
// 的缓存键）。GUI 详情页每次渲染都会问「有无升级命令」，探测是子进程 IO，
// 必须缓存避免每次渲染重跑。进程内缓存即可（扫描/会话生命周期内二进制
// 不变；升级后 mtime 变化自然失效）。
type probeEntry struct {
	Size    int64
	MtimeNs int64
	Sub     string // 探测到的自升级子命令；空 = 无
	Ok      bool   // 探测是否成功（区分「无子命令」与「探测失败」）
}

var (
	probeMu    sync.Mutex
	probeCache = map[string]probeEntry{}
)

// ProbeSelfUpgrade 运行 <bin> -h（失败回退 --help）解析自升级子命令。
// 返回子命令串（如 "update"、"self update"）；无子命令或探测失败返回空。
// 带缓存（real path + size + mtime），GUI 详情页渲染与 CLI 检测共用。
// 解析策略：在 Commands 区找以 update/upgrade 开头的子命令行。多数 CLI 的
// help 形如 "  update|upgrade  Check for updates"，取第一个词（update 或
// upgrade）。uv 是 "  self  Manage the uv executable" → self update。
func ProbeSelfUpgrade(ctx context.Context, bin string) string {
	st, err := stat(bin)
	if err != nil {
		return ""
	}
	key := bin
	probeMu.Lock()
	e, hit := probeCache[key]
	probeMu.Unlock()
	if hit && e.Size == st.Size() && e.MtimeNs == st.ModTime().UnixNano() {
		return e.Sub
	}
	sub := probeHelpSub(ctx, bin)
	probeMu.Lock()
	probeCache[key] = probeEntry{Size: st.Size(), MtimeNs: st.ModTime().UnixNano(), Sub: sub, Ok: true}
	probeMu.Unlock()
	return sub
}

// probeHelpSub 实际执行 -h / --help 并解析子命令。
func probeHelpSub(ctx context.Context, bin string) string {
	binBase := stripExeExt(filepath.Base(bin))
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		out := helpOutput(ctx, bin, args...)
		if out == "" {
			continue // 无输出/启动失败：尝试下一个参数
		}
		if sub := parseSelfUpgradeSub(out, binBase); sub != "" {
			return sub
		}
	}
	return ""
}

// helpOutput 执行 <bin> <args> 并返回合并的 stdout+stderr。
// 与 defaultRunQuery（检测用）不同：help 输出有的 CLI 写 stdout（claude/
// kimi），有的写 stderr（opencode/yargs 风格），合并两侧保证都能解析。
// 仅探测用，不参与 detect 的 JSON 契约。带 PATH 增强与超时（镜像
// defaultRunQuery 的进程环境）。包级变量便于测试注入假实现。
var helpOutput = defaultHelpOutput

func defaultHelpOutput(ctx context.Context, bin string, args ...string) string {
	ctx2, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	resolved := bin
	if r, rerr := cmdexec.ResolveCommand(bin); rerr == nil {
		resolved = r
	}
	var cmd *exec.Cmd
	if isCmdShim(resolved) {
		// Windows .cmd/.bat shim（npm 全局等）：CreateProcess 不解析脚本，
		// 经 cmd.exe /c 执行（镜像 RunOfficial 的处理）。
		cmd = exec.CommandContext(ctx2, "cmd.exe", append([]string{"/c", resolved}, args...)...)
	} else {
		cmd = exec.CommandContext(ctx2, resolved, args...)
	}
	cmd.Env = cmdexec.WithPath(os.Environ(), cmdexec.AugmentedPathEnv())
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil && outBuf.Len() == 0 && errBuf.Len() == 0 {
		return ""
	}
	return outBuf.String() + errBuf.String()
}

// parseSelfUpgradeSub 从 help 输出解析自升级子命令（binBase 是程序名，
// 用于识别 "opencode upgrade" 形态的命令行）。
// 在 Commands 区（"Commands:" 等之后）找 update/upgrade 子命令。两种形态：
//   - 裸子命令：  "  upgrade|update  Upgrade to latest"   → upgrade
//   - 带程序名：  "  opencode upgrade [target]    ..."   → upgrade
//     （首个 token 等于程序名时取第二个 token）
//
// 只检查命令名 token，不扫描描述文本，避免 "install plugin and update
// config" 里的 update 误命中。
func parseSelfUpgradeSub(out, binBase string) string {
	inCommands := false
	for _, line := range strings.Split(out, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "Commands:") || strings.HasPrefix(trim, "COMMANDS") ||
			strings.HasPrefix(trim, "Available commands:") {
			inCommands = true
			continue
		}
		if !inCommands {
			continue
		}
		fields := strings.Fields(trim)
		if len(fields) == 0 {
			continue
		}
		// 命令名 token：首个；若等于程序名（opencode upgrade → 取 upgrade）
		cands := []string{fields[0]}
		if binBase != "" && strings.EqualFold(fields[0], binBase) && len(fields) >= 2 {
			cands = append(cands, fields[1])
		}
		for _, f := range cands {
			low := strings.ToLower(f)
			if low == "update" || low == "upgrade" {
				return low
			}
			// 形如 "update|upgrade" / "upgrade|update" 复合名
			for _, p := range strings.Split(f, "|") {
				p = strings.ToLower(p)
				if p == "update" || p == "upgrade" {
					return p
				}
			}
		}
	}
	return ""
}

// stripExeExt 剥掉 Windows 可执行扩展名（claude.exe → claude，npm.cmd →
// npm）。扫描器在 Windows 上记录的二进制名带扩展名（discover 取目录文件名），
// 静态表键与 help 里的程序名都是裸名，查找/匹配前必须对齐。
func stripExeExt(name string) string {
	low := strings.ToLower(name)
	for _, ext := range []string{".exe", ".cmd", ".bat", ".com"} {
		if strings.HasSuffix(low, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

// stat 是 os.Stat 的薄封装（测试可注入）。
var stat = func(path string) (os.FileInfo, error) { return os.Stat(path) }

// HasCommand 报告该来源是否有可展示的官方升级命令（而非纯提示）。
// brew/npm/pipx/cargo 恒有命令；local-bin 仅命中已知官方脚本表（uv/poetry/rye）
// 时有命令；go/versioned/other/pyenv/未知 local-bin 只有提示，无命令。
// GUI 据此决定是否渲染「检查更新」按钮与结果弹窗（无命令不弹，避免展示
// 无意义的提示面板）。
// 注意：这是同步静态判断（不探测）。探测结果走 CheckTool/OfficialFor 的
// Command 字段，GUI 按钮显隐由 ToolUpgradeSupported 异步探测决定。
func HasCommand(installer scanner.Installer, name string) bool {
	name = stripExeExt(strings.TrimSpace(name))
	if _, ok := selfUpgradeSub[name]; ok {
		return true // 已知自升级子命令工具
	}
	switch installer {
	case scanner.InstBrew, scanner.InstNpm, scanner.InstPipx, scanner.InstCargo:
		return true
	case scanner.InstLocalBin:
		_, ok := localBinScripts[name]
		return ok
	default:
		return false
	}
}

// 命令/提示一律不编造：go 需要模块路径（工具名 ≠ 模块路径），versioned/
// other/pyenv 无统一升级方式，只给提示。
func OfficialCommand(installer scanner.Installer, name, binName string) Command {
	return officialCommand(installer, name, binName, "")
}

// probeCommand 对无静态命令的来源（versioned/other/go/pyenv 等）做 -h 探测：
// 命中自升级子命令 → 返回可代跑命令；未命中 → 返回通用提示（与静态一致）。
// ctx 可空（nil 时用默认超时）；bin 是探测二进制路径（空则不探测）。
func probeCommand(ctx context.Context, installer scanner.Installer, name, bin string) Command {
	if bin == "" {
		return officialCommand(installer, name, "", "")
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), probeTimeout)
		defer cancel()
	}
	sub := ProbeSelfUpgrade(ctx, bin)
	if sub == "" {
		// 探测失败/无子命令：回退静态提示（不编造）
		return officialCommand(installer, name, "", "")
	}
	// 探测到的自升级命令可代跑：<bin> <sub>（如 claude update / uv self update）
	return Command{
		Command:  name + " " + sub,
		Runnable: true,
		Bin:      bin,
		Args:     strings.Fields(sub),
	}
}

// officialCommand 是 OfficialCommand 的内部实现；pkg 是已解析的真实包名
// （仅 npm 用，调用方经 t.Package / NpmPackageFor 得到）。
func officialCommand(installer scanner.Installer, name, binName, pkg string) Command {
	name = strings.TrimSpace(name)
	// 已知自升级子命令工具（claude update / kimi upgrade / codegraph upgrade /
	// uv self update）：命令可代跑（<bin> <sub>），经 PATH 解析执行。
	// Windows 二进制名带扩展名（claude.exe），查表前对齐裸名。
	if sub, ok := selfUpgradeSub[stripExeExt(name)]; ok {
		return Command{
			Command:  name + " " + sub,
			Runnable: true,
			Bin:      name,
			Args:     strings.Fields(sub),
		}
	}
	switch installer {
	case scanner.InstBrew:
		return Command{Command: "brew upgrade " + name, Runnable: true, Bin: "brew", Args: []string{"upgrade", name}}
	case scanner.InstNpm:
		// 工具名可能是合并后的短名（npmToolID 映射，如 pi →
		// @earendil-works/pi-coding-agent，也可能是撞名的真实短名包）；
		// npm 按真实包名寻址，用错名会静默误报「已最新」或代跑失败。
		if pkg == "" {
			pkg = scanner.NpmPackageFor(name)
		}
		return Command{Command: "npm update -g " + pkg, Runnable: true, Bin: "npm", Args: []string{"update", "-g", pkg}}
	case scanner.InstPipx:
		return Command{Command: "pipx upgrade " + name, Runnable: true, Bin: "pipx", Args: []string{"upgrade", name}}
	case scanner.InstCargo:
		// 扫描器按二进制名归类（rg），cargo 按 crate 名寻址；调用方经
		// cargoCrateOf 把 pkg 解析为真实 crate 名（ripgrep），否则用二进制名
		// 会装错/装不到（code review #6）。
		if pkg == "" {
			pkg = name
		}
		return Command{Command: "cargo install " + pkg + " --force", Runnable: true, Bin: "cargo", Args: []string{"install", pkg, "--force"}}
	case scanner.InstLocalBin:
		if script, ok := localBinScripts[name]; ok {
			return Command{Command: script} // 仅展示，不代跑
		}
		return Command{Command: "重新运行官方安装脚本（见工具官网）"}
	case scanner.InstGo:
		return Command{Command: "重新执行当时的 go install（需模块路径）"}
	default: // versioned / other / pyenv / nodejs / rustup 等
		return Command{Command: "无统一官方升级命令（参考工具官网）"}
	}
}
