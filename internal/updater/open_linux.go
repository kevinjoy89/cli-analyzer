//go:build linux

package updater

import (
	"os/exec"
	"syscall"
)

// OpenInstaller 以系统默认方式打开安装包（Linux: xdg-open，可打开 deb/压缩包）。
// 以 detached 方式启动，父进程退出后继续存活。
// 无桌面环境时 xdg-open 会失败，调用方应展示文件路径供手动处理。
func OpenInstaller(path string) error {
	cmd := exec.Command("xdg-open", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
