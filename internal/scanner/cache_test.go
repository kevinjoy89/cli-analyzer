package scanner

import (
	"testing"
)

// TestClearCacheIdempotent 验证无缓存时 clear 视为成功（幂等）：
// 首次运行/已清过的机器上"清除失败"是误导错误。
func TestClearCacheIdempotent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := ClearCache(); err != nil {
		t.Fatalf("无缓存时 ClearCache 应成功: %v", err)
	}
}

// TestFingerprintRoundTrip 验证指纹文件的原子写与读回。
// 隔离：unix 用 XDG_CACHE_HOME，Windows 用 LOCALAPPDATA（CacheRoot 的取值链）。
func TestFingerprintRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	entries := []FingerprintEntry{{Path: "/tmp/a", MTime: 1, Size: 2, IsDir: true}}
	if err := SaveFingerprint(entries); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if !FingerprintsEqual(got, entries) {
		t.Fatalf("读回不一致: %+v", got)
	}
	// ClearCache 应同时清指纹文件
	if err := ClearCache(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFingerprint(); err == nil {
		t.Fatal("ClearCache 后指纹文件应不存在")
	}
}
