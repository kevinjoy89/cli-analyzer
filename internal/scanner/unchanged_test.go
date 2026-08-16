package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// 预置缓存 + 指纹后，ScanIfUnchanged 在无变更时应直接返回缓存
// （不执行全量扫描，结果 ScannedAt 与缓存一致）。
func TestScanIfUnchangedCacheHit(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	dir := t.TempDir()
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cached := &ScanResult{
		ScannedAt: "2026-08-15T00:00:00+08:00",
		Tools: []Tool{{
			Name:     "tool",
			Binaries: []Binary{{Name: "tool", Path: bin, Real: bin, Size: 16}},
		}},
		Totals: Totals{Footprint: 16},
		Roots:  map[string][]string{},
	}
	if err := SaveCache(cached); err != nil {
		t.Fatal(err)
	}
	if err := SaveFingerprint(ComputeFingerprint(cached)); err != nil {
		t.Fatal(err)
	}

	res, err := ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ScannedAt != cached.ScannedAt {
		t.Fatalf("未变更时应返回缓存 (ScannedAt %s), got %s", cached.ScannedAt, res.ScannedAt)
	}
	if res.Totals.Footprint != 16 {
		t.Fatalf("缓存内容应原样返回, got %+v", res.Totals)
	}
}

// 指纹文件缺失（首次运行）时保守走全量扫描：结果不应是假缓存。
func TestScanIfUnchangedNoFingerprintFallsBack(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	dir := t.TempDir()
	bin := filepath.Join(dir, "tool")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	cached := &ScanResult{
		ScannedAt: "2026-08-15T00:00:00+08:00",
		Tools:     []Tool{{Name: "tool", Binaries: []Binary{{Name: "tool", Path: bin, Real: bin, Size: 16}}}},
		Roots:     map[string][]string{},
	}
	if err := SaveCache(cached); err != nil {
		t.Fatal(err)
	}
	// 不写指纹文件

	res, err := ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ScannedAt == cached.ScannedAt {
		t.Fatal("无指纹文件时应走全量扫描（结果不应等于旧缓存）")
	}
}
