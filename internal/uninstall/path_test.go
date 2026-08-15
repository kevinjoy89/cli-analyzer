package uninstall

import (
	"strings"
	"testing"
)

// TestWithPathCaseInsensitive 验证 WithPath 对 PATH 环境变量的替换是
// 大小写不敏感的：Windows 上系统环境变量名为 "Path"（而非 "PATH"），
// 只匹配 "PATH=" 前缀会保留旧变量并追加新变量，子进程拿到重复 PATH。
// GUI 与 CLI 代跑卸载命令共用此函数（行为一致）。
func TestWithPathCaseInsensitive(t *testing.T) {
	env := WithPath([]string{"Path=/old/one", "HOME=/x"}, "/new/path")
	pathCount := 0
	var last string
	for _, e := range env {
		upper := strings.ToUpper(e)
		if strings.HasPrefix(upper, "PATH=") {
			pathCount++
			last = e
		}
	}
	if pathCount != 1 {
		t.Errorf("PATH-like entries = %d, want exactly 1 (case-insensitive replacement); env=%v", pathCount, env)
	}
	if last != "PATH=/new/path" {
		t.Errorf("last PATH entry = %q, want %q", last, "PATH=/new/path")
	}
	foundHome := false
	for _, e := range env {
		if e == "HOME=/x" {
			foundHome = true
		}
	}
	if !foundHome {
		t.Errorf("HOME=/x lost after WithPath: %v", env)
	}
}
