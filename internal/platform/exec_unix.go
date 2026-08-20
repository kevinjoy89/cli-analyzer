//go:build !windows

package platform

import "os"

// IsExecutable reports whether the file info refers to a regular file with any
// execute bit set (Unix semantics).
func IsExecutable(f os.FileInfo) bool {
	return f.Mode().IsRegular() && f.Mode().Perm()&0o111 != 0
}

// ExecExtensions is empty outside Windows: no extension completion needed.
func ExecExtensions() []string { return nil }
