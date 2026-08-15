package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFilteredScanCacheComplete 验证带 ToolFilter 的扫描写缓存时，缓存是
// 完整快照：ScanTimeMS 与 Unattributed 必须保留。此前 filter 路径会重新
// finalize 生成新 ScanResult——ScanTimeMS 恒 0、Unattributed 恒 nil，
// GUI 从该缓存渲染时显示"扫描耗时 0ms"且孤儿列表为空（上一轮修复只覆盖
// 了无 filter 的主路径，filter 路径是同一 bug 的漏网分支）。
func TestFilteredScanCacheComplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only PATH 注入用例")
	}
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mytool-zz"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	// 未认领的孤儿候选目录：任何工具都不认领它
	orphan := filepath.Join(cacheHome, "zz-unclaimed-dir")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Scan(Options{}); err != nil {
		t.Fatal(err)
	}
	// 带 filter 的扫描（等价于 `scan mytool --refresh`）：重写缓存
	if _, err := Scan(Options{ToolFilter: []string{"mytool"}}); err != nil {
		t.Fatal(err)
	}
	cached, err := LoadCache()
	if err != nil {
		t.Fatal(err)
	}
	if cached.ScanTimeMS <= 0 {
		t.Errorf("filtered cache scanTimeMs = %d, want > 0", cached.ScanTimeMS)
	}
	foundOrphan := false
	for _, u := range cached.Unattributed {
		if u.Path == orphan {
			foundOrphan = true
			break
		}
	}
	if !foundOrphan {
		t.Errorf("filtered cache 丢失 Unattributed 孤儿目录 %s (got %v)", orphan, cached.Unattributed)
	}
}
