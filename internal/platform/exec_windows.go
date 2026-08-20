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

// ExecExtensions 返回 Windows 可执行扩展名（PATHEXT，缺省 .com/.exe/.bat/
// .cmd）。cmdexec.ResolveCommand 用它对裸命令名补全扩展名（"claude" →
// claude.exe），与 IsExecutable 的判定口径一致。
func ExecExtensions() []string {
	pathext := os.Getenv("PATHEXT")
	if strings.TrimSpace(pathext) == "" {
		return []string{".com", ".exe", ".bat", ".cmd"}
	}
	var out []string
	for _, e := range strings.Split(pathext, ";") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}
