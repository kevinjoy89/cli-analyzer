//go:build windows

package platform

import (
	"os/exec"
	"syscall"
)

// hideConsoleWindow 阻止 GUI 启动的子进程闪出控制台窗口
// （更新安装器的 cmd /c start、卸载代跑的 npm 等）。CREATE_NO_WINDOW
// 让子进程不创建新的控制台窗口；HideWindow 同时把窗口状态设为隐藏。
func HideConsoleWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
