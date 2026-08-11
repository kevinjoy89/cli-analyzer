//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// IsExecutable reports whether the file matches a PATHEXT extension on Windows.
func IsExecutable(f os.FileInfo) bool {
	if !f.Mode().IsRegular() {
		return false
	}
	ext := strings.ToLower(filepath.Ext(f.Name()))
	for _, e := range strings.Split(os.Getenv("PATHEXT"), ";") {
		if e != "" && ext == strings.ToLower(e) {
			return true
		}
	}
	return false
}
