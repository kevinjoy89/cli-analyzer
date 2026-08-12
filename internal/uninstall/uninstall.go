// Package uninstall 提供 CLI 工具的标准卸载与残留清理能力。
//
// 设计要点（见 change add-tool-uninstall design.md）：
//   - 标准卸载命令按安装来源映射，仅 brew/npm/pipx/cargo 可代跑；
//     go/pyenv/versioned/other 只给提示，不做整包强卸。
//   - 残留 = 规则表数据目录 ∪ 扫描快照归因目录，卸载后仍存在者。
//   - 残留清理是唯一允许触碰 USER 级数据的路径，硬约束为必须经内置
//     回收站（可恢复）；本包不提供任何永久删除变体。
package uninstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
)

// Official 是某个工具的标准卸载建议。
type Official struct {
	// Command 是展示给用户的完整命令（go/pyenv 等为提示命令）。
	Command string `json:"command"`
	// Runnable 表示工具能否代为执行（brew/npm/pipx/cargo）。
	Runnable bool `json:"runnable"`
	// Bin / Args 是代跑用的可执行与参数（Runnable 时有效）。
	Bin  string   `json:"-"`
	Args []string `json:"-"`
}

// blocklist 是拒绝卸载的系统关键工具（含 cli-analyzer 自身）。
var blocklist = map[string]bool{
	"python": true, "python3": true, "python3.13": true, "python3.12": true,
	"node": true, "npm": true, "npx": true, "corepack": true,
	"git": true, "docker": true, "podman": true, "go": true, "gofmt": true,
	"brew": true, "bash": true, "zsh": true, "sh": true, "fish": true,
	"bun": true, "yarn": true, "pnpm": true, "deno": true,
	"rustc": true, "cargo": true, "rustup": true,
	"sudo": true, "ssh": true, "curl": true, "wget": true, "vim": true,
	"cli-analyzer": true,
}

// IsBlocked 报告该工具名是否命中系统关键工具黑名单。
func IsBlocked(name string) bool {
	return blocklist[strings.ToLower(strings.TrimSpace(name))]
}

// OfficialCommand 返回按安装来源映射的标准卸载建议。
// binName 是 PATH 上的命令名（go 来源的提示需要它）。
func OfficialCommand(installer scanner.Installer, name, binName string) Official {
	name = strings.TrimSpace(name)
	switch installer {
	case scanner.InstBrew:
		return Official{Command: "brew uninstall " + name, Runnable: true, Bin: "brew", Args: []string{"uninstall", name}}
	case scanner.InstNpm:
		return Official{Command: "npm uninstall -g " + name, Runnable: true, Bin: "npm", Args: []string{"uninstall", "-g", name}}
	case scanner.InstPipx:
		return Official{Command: "pipx uninstall " + name, Runnable: true, Bin: "pipx", Args: []string{"uninstall", name}}
	case scanner.InstCargo:
		return Official{Command: "cargo uninstall " + name, Runnable: true, Bin: "cargo", Args: []string{"uninstall", name}}
	case scanner.InstGo:
		// go install 无包管理器：删除 GOPATH/bin 下的二进制（仅提示，不代跑）
		return Official{Command: fmt.Sprintf("rm $(go env GOPATH)/bin/%s", binName)}
	case scanner.InstPyenv:
		// pyenv 托管的解释器：在对应版本内卸载包（仅提示）
		return Official{Command: "pyenv 版本内执行 pip uninstall / pipx uninstall（提示，不代跑）"}
	default: // versioned / other / rustup 等：无统一标准卸载命令
		return Official{}
	}
}

// ResolveCommand 在（含增强的）PATH 目录中解析命令的绝对路径。
// GUI 从 Finder 启动时进程 PATH 是系统最小集，直接 exec "npm" 会报
// "executable file not found in $PATH"——与扫描漏工具的根因相同。
func ResolveCommand(bin string) (string, error) {
	for _, dir := range platform.PathDirs(false) {
		p := filepath.Join(dir, bin)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && platform.IsExecutable(st) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: %q not found on PATH", os.ErrNotExist, bin)
}

// AugmentedPathEnv 返回含增强目录的 PATH 环境变量，供卸载命令的子进程使用：
// npm 等工具内部会再派生子进程（如 node），子进程继承的 PATH 必须完整。
func AugmentedPathEnv() string {
	return strings.Join(platform.PathDirs(false), string(os.PathListSeparator))
}
