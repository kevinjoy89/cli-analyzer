//go:build !windows

package probe

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup 让探测子进程独立进程组，便于超时后整组终止。
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup 终止探测子进程的整个进程组。
func killGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
