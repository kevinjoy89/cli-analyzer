//go:build !windows

package platform

// IsConsoleExe is a no-op outside Windows: POSIX executability already
// implies a command (no PE subsystem concept).
func IsConsoleExe(path string) bool { return true }
