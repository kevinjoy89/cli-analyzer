package upgrade

import (
	"strings"

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
var localBinScripts = map[string]string{
	"uv":     "curl -LsSf https://astral.sh/uv/install.sh | sh",
	"poetry": "curl -sSL https://install.python-poetry.org | python3 -",
	"rye":    "curl -sSf https://rye.astral.sh/get | bash",
}

// HasCommand 报告该来源是否有可展示的官方升级命令（而非纯提示）。
// brew/npm/pipx/cargo 恒有命令；local-bin 仅命中已知官方脚本表（uv/poetry/rye）
// 时有命令；go/versioned/other/pyenv/未知 local-bin 只有提示，无命令。
// GUI 据此决定是否渲染「检查更新」按钮与结果弹窗（无命令不弹，避免展示
// 无意义的提示面板）。
func HasCommand(installer scanner.Installer, name string) bool {
	name = strings.TrimSpace(name)
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

// officialCommand 是 OfficialCommand 的内部实现；pkg 是已解析的真实包名
// （仅 npm 用，调用方经 t.Package / NpmPackageFor 得到）。
func officialCommand(installer scanner.Installer, name, binName, pkg string) Command {
	name = strings.TrimSpace(name)
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
