//go:build !windows

package platform

import "os/exec"

// hideConsoleWindow 在非 Windows 平台为空操作（无控制台窗口概念）。
func HideConsoleWindow(cmd *exec.Cmd) {}
