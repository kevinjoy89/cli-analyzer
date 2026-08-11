//go:build !windows

package disk

import (
	"os"
	"syscall"
)

// fileSize returns physical bytes allocated on disk (sparse-file accurate).
func fileSize(info os.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Blocks > 0 {
		return st.Blocks * 512
	}
	return info.Size()
}
