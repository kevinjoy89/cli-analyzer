//go:build windows

package updater

import (
	"os/exec"
	"strings"

	"cli-analyzer/internal/platform"
)

// OpenInstaller 以系统默认方式打开安装包（Windows: start，可打开 exe/zip）。
// 使用 "cmd /c start" 使打开操作脱离本进程，父进程退出后继续执行；
// hideConsoleWindow 隐藏 cmd 控制台窗口，避免更新安装时闪出黑窗。
func OpenInstaller(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", quoteStartArg(path))
	platform.HideConsoleWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// quoteStartArg 给路径加双引号：cmd 的 start 按空格拆参数，Downloads 路径
// 含空格（用户名如 "John Doe"）时不加引号会把路径拆成多个参数导致打开失败；
// 含 cmd 元字符（& | < > ^ ( ) 等，Windows 用户名/文件名合法字符）时同样
// 会被 cmd 解释为命令拼接或重定向，必须一并加引号。
func quoteStartArg(path string) string {
	if strings.HasPrefix(path, `"`) {
		return path
	}
	if strings.ContainsAny(path, " \t&|<>^()%!") {
		return `"` + path + `"`
	}
	return path
}
