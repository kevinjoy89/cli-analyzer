//go:build windows

package updater

import (
	"os/exec"

	"cli-analyzer/internal/platform"
)

// OpenInstaller 以系统默认方式打开安装包（Windows: start，可打开 exe/zip）。
// 使用 "cmd /c start" 使打开操作脱离本进程，父进程退出后继续执行；
// hideConsoleWindow 隐藏 cmd 控制台窗口，避免更新安装时闪出黑窗。
func OpenInstaller(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	platform.HideConsoleWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
