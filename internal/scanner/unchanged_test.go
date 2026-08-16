package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 预置缓存 + 指纹后，ScanIfUnchanged 在无变更时应直接返回缓存
// （不执行全量扫描，结果 ScannedAt 与缓存一致），scanned 为 false。
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

	res, scanned, err := ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if scanned {
		t.Fatal("无变更时应命中缓存（scanned=false）")
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

	res, scanned, err := ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !scanned {
		t.Fatal("无指纹文件时应走全量扫描（scanned=true）")
	}
	if res.ScannedAt == cached.ScannedAt {
		t.Fatal("无指纹文件时应走全量扫描（结果不应等于旧缓存）")
	}
}

// 新工具安装检测：PATH 临时目录新增二进制（目录 mtime 变化）→ 指纹不一致
// → 自动全量扫描（scanned=true）。PATH 指向隔离临时目录，避免依赖机器 PATH。
func TestScanIfUnchangedDetectsNewPathBinary(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	pathDir := t.TempDir()
	oldPath := os.Getenv("PATH")
	sep := ":"
	if os.PathSeparator == '\\' {
		sep = ";"
	}
	t.Setenv("PATH", pathDir+sep+oldPath)

	bin := filepath.Join(pathDir, "tool")
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
	if err := SaveFingerprint(ComputeFingerprint(cached)); err != nil {
		t.Fatal(err)
	}

	// 首次：无变更 → 命中缓存
	res, scanned, err := ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if scanned {
		t.Fatal("初始状态应命中缓存")
	}
	if res.ScannedAt != cached.ScannedAt {
		t.Fatal("初始状态应返回缓存")
	}

	// 向 PATH 目录新增二进制（模拟新装工具）→ 目录 mtime 变化 → 自动全量
	newBin := filepath.Join(pathDir, "newtool")
	before, _ := os.Stat(pathDir)
	if err := os.WriteFile(newBin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 粗粒度文件系统（FAT 2s 等）目录 mtime 可能不变：显式推进，保证判定确定
	if after, _ := os.Stat(pathDir); after != nil && before != nil && after.ModTime().Equal(before.ModTime()) {
		_ = os.Chtimes(pathDir, time.Now(), time.Now().Add(time.Second))
	}
	res, scanned, err = ScanIfUnchanged(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !scanned {
		t.Fatal("PATH 目录新增二进制后应自动全量扫描（scanned=true）")
	}
	if res.ScannedAt == cached.ScannedAt {
		t.Fatal("自动全量后结果应不同于旧缓存")
	}
}
