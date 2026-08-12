//go:build darwin

package updater

import (
	"os/exec"
	"syscall"
)

// OpenInstaller 以系统默认方式打开安装包（macOS: open，可打开 dmg）。
// 以 detached 方式启动：父进程退出后子进程继续存活（先打开后退出流程的前提）。
func OpenInstaller(path string) error {
	cmd := exec.Command("open", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
