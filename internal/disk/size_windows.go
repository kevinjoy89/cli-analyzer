//go:build windows

package disk

import "os"

// fileSize returns apparent size on Windows (no block accounting available).
func fileSize(info os.FileInfo) int64 {
	return info.Size()
}
