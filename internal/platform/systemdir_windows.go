//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// isSystemDir reports whether d is an OS-managed directory on the PATH.
// C:\Windows\System32 (ssh.exe, cmd.exe, wsl.exe, winload.exe…), SysWOW64 and
// every subdirectory under %WINDIR% are OS-managed, not user tools — they are
// skipped exactly like /usr/bin on unix. Matching is case-insensitive and
// tolerant of a missing WINDIR env (defaults to C:\Windows).
func isSystemDir(d string) bool {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	low := strings.ToLower(filepath.Clean(d))
	wl := strings.ToLower(filepath.Clean(windir))
	return low == wl || strings.HasPrefix(low, wl+`\`)
}
