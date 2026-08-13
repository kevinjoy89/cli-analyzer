//go:build !windows

package platform

// isSystemDir reports whether d is an OS-managed bin directory that is skipped
// during scans (system-installed tools are not user tools).
func isSystemDir(d string) bool {
	switch d {
	case "/usr/bin", "/bin", "/usr/sbin", "/sbin":
		return true
	}
	return false
}
