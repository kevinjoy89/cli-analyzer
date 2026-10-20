package probe

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// setupCache 把探测缓存指向临时目录并重置内存态。
func setupCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	origRoot := cachePathFn
	cachePathFn = func() string { return filepath.Join(dir, "probe-versions.json") }
	t.Cleanup(func() { cachePathFn = origRoot })
	cacheOnce = sync.Once{}
	cacheMu.Lock()
	cache = map[string]entry{}
	cacheMu.Unlock()
	return dir
}

// TestCachedFailureIsRetried 校验失败结果不会永久缓存：首次失败后，
// 下一次调用必须重新探测，而不是永远返回空版本。
func TestCachedFailureIsRetried(t *testing.T) {
	setupCache(t)
	bin := filepath.Join(t.TempDir(), "faketool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}

	calls := 0
	origProbe := probeVersionFn
	probeVersionFn = func(string) (string, bool) { calls++; return "", false }
	t.Cleanup(func() { probeVersionFn = origProbe })

	// 前 maxCachedFailures 次都会真正调用探测
	for i := 0; i < maxCachedFailures; i++ {
		CachedOrRun(bin, bin, st.Size())
	}
	if calls != maxCachedFailures {
		t.Errorf("probe calls = %d, want %d (failures must be retried)", calls, maxCachedFailures)
	}

	// 达到上限后视为权威失败，不再重复探测
	CachedOrRun(bin, bin, st.Size())
	if calls != maxCachedFailures {
		t.Errorf("probe calls = %d, want %d (cached failure should stop retrying)", calls, maxCachedFailures)
	}
}

// TestCachedSuccessIsStable 校验成功结果命中缓存、不重复探测。
func TestCachedSuccessIsStable(t *testing.T) {
	setupCache(t)
	bin := filepath.Join(t.TempDir(), "faketool")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(bin)

	calls := 0
	origProbe := probeVersionFn
	probeVersionFn = func(string) (string, bool) { calls++; return "1.2.3", true }
	t.Cleanup(func() { probeVersionFn = origProbe })

	for i := 0; i < 3; i++ {
		v, ok := CachedOrRun(bin, bin, st.Size())
		if !ok || v != "1.2.3" {
			t.Fatalf("got %q ok=%v", v, ok)
		}
	}
	if calls != 1 {
		t.Errorf("probe calls = %d, want 1 (success cached)", calls)
	}
}

// TestRecoveredFailureReportsVersion 校验「先失败后成功」：重试拿到版本后，
// 缓存里的 ok 与 version 都更新（否则界面仍显示 —）。
func TestRecoveredFailureReportsVersion(t *testing.T) {
	setupCache(t)
	bin := filepath.Join(t.TempDir(), "faketool")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(bin)

	origProbe := probeVersionFn
	t.Cleanup(func() { probeVersionFn = origProbe })

	probeVersionFn = func(string) (string, bool) { return "", false }
	CachedOrRun(bin, bin, st.Size())

	probeVersionFn = func(string) (string, bool) { return "2.1.0", true }
	v, ok := CachedOrRun(bin, bin, st.Size())
	if !ok || v != "2.1.0" {
		t.Fatalf("retry should recover, got %q ok=%v", v, ok)
	}
	// 再取一次应直接命中成功结果
	probeVersionFn = func(string) (string, bool) { return "", false }
	v, ok = CachedOrRun(bin, bin, st.Size())
	if !ok || v != "2.1.0" {
		t.Errorf("cached success should stick, got %q ok=%v", v, ok)
	}
}
