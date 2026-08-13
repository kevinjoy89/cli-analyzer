//go:build windows

package probe

import "os/exec"

// setupProcessGroup 在 Windows 上无进程组概念；控制台窗口已由
// platform.HideConsoleWindow 隐藏。
func setupProcessGroup(cmd *exec.Cmd) {}

// killGroup 终止探测子进程（Windows 无进程组）。
func killGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
