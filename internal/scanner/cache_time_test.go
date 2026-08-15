package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestScanTimeMSInCache 验证缓存结果携带真实的扫描耗时：
// Scan() 把 res 序列化进缓存发生在 res.ScanTimeMS 赋值之前，导致
// 缓存 JSON 里 scanTimeMs 恒为 0（GUI 启动从缓存渲染时显示"扫描耗时 0ms"）。
func TestScanTimeMSInCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only PATH 注入用例")
	}
	// 隔离缓存根与数据根，避免读写真实用户目录
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mytool-zz"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	if _, err := Scan(Options{}); err != nil {
		t.Fatal(err)
	}
	cached, err := LoadCache()
	if err != nil {
		t.Fatal(err)
	}
	if cached.ScanTimeMS <= 0 {
		t.Errorf("cached scanTimeMs = %d, want > 0 (ScanTimeMS must be set before SaveCache)", cached.ScanTimeMS)
	}
}
