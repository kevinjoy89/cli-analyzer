//go:build windows

package updater

import "os/exec"

// OpenInstaller 以系统默认方式打开安装包（Windows: start，可打开 exe/zip）。
// 使用 "cmd /c start" 使打开操作脱离本进程，父进程退出后继续执行。
func OpenInstaller(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
