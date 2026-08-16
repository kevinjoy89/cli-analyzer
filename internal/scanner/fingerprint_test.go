package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cli-analyzer/internal/platform"
)

// 用临时目录构造扫描结果，验证指纹采集与比较语义。
func TestComputeFingerprintAndEqual(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "tool")
	data := filepath.Join(dir, "data")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(data, 0o755); err != nil {
		t.Fatal(err)
	}
	res := &ScanResult{
		Tools: []Tool{{
			Name:     "tool",
			Binaries: []Binary{{Name: "tool", Path: bin, Real: bin, Size: 16}},
			DataDirs: []DataDir{{Path: data}},
		}},
	}
	fp := ComputeFingerprint(res)
	// 指纹 = 结果测量路径（2）+ 当前存在的 PATH 发现目录数（动态：测试机
	// 的 home/系统 PATH 目录可能真实存在，不能写死）。
	want := 2 + existingPathDirs()
	if len(fp) != want {
		t.Fatalf("指纹条目数 = %d, want %d (2 result paths + %d PATH dirs)", len(fp), want, existingPathDirs())
	}
	// 排序稳定 + 等值（顺序无关）
	if !FingerprintsEqual(fp, fp) {
		t.Fatal("同指纹应相等")
	}
	// 路径消失 = 变更
	shorter := fp[:1]
	if FingerprintsEqual(fp, shorter) {
		t.Fatal("条目缺失应判定为变更")
	}
	// mtime 变化 = 变更（touch 二进制）
	future := fp[0].MTime + 1000
	fp2 := append([]FingerprintEntry(nil), fp...)
	fp2[0].MTime = future
	if FingerprintsEqual(fp, fp2) {
		t.Fatal("mtime 变化应判定为变更")
	}
	// 大小变化 = 变更
	fp3 := append([]FingerprintEntry(nil), fp...)
	fp3[1].Size = fp3[1].Size + 1
	if FingerprintsEqual(fp, fp3) {
		t.Fatal("size 变化应判定为变更")
	}
	// 真实 stat 与手动 stat 一致（mtime 语义验证）：按路径定位二进制条目
	st, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	var binFp *FingerprintEntry
	for i := range fp {
		if fp[i].Path == filepath.Clean(bin) {
			binFp = &fp[i]
			break
		}
	}
	if binFp == nil {
		t.Fatalf("二进制路径未出现在指纹中: %v", fp)
	}
	if binFp.MTime != st.ModTime().UnixNano() || binFp.Size != st.Size() || binFp.IsDir {
		t.Fatalf("指纹与 os.Stat 不一致: %+v vs mtime=%d size=%d", binFp, st.ModTime().UnixNano(), st.Size())
	}
}

// existingPathDirs 统计当前 PATH 发现目录中真实存在的目录数（测试辅助：
// ComputeFingerprint 的 PATH 条目断言需要动态计数）。
func existingPathDirs() int {
	n := 0
	for _, d := range platform.PathDirs(true) {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			n++
		}
	}
	return n
}

// PATH 发现目录变更检测：目录 mtime 变化（新增文件）→ 指纹不一致。
func TestFingerprintDetectsPathDirChange(t *testing.T) {
	pathDir := t.TempDir()
	oldPath := os.Getenv("PATH")
	sep := ":"
	if os.PathSeparator == '\\' {
		sep = ";"
	}
	t.Setenv("PATH", pathDir+sep+oldPath)

	res := &ScanResult{Roots: map[string][]string{}}
	fp1 := ComputeFingerprint(res)
	if !containsPathDir(fp1, pathDir) {
		t.Fatalf("指纹应包含 PATH 发现目录条目: %v", fp1)
	}
	// 向 PATH 目录新增文件 → 目录 mtime 变化 → 指纹不一致
	before, _ := os.Stat(pathDir)
	if err := os.WriteFile(filepath.Join(pathDir, "newtool"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 粗粒度文件系统（FAT 2s 等）目录 mtime 可能不变：显式推进，保证判定确定
	if after, _ := os.Stat(pathDir); after != nil && before != nil && after.ModTime().Equal(before.ModTime()) {
		_ = os.Chtimes(pathDir, time.Now(), time.Now().Add(time.Second))
	}
	fp2 := ComputeFingerprint(res)
	if FingerprintsEqual(fp1, fp2) {
		t.Fatal("PATH 目录新增文件后指纹应变化")
	}
	// 目录 mtime 未变（无内容变化）→ 指纹一致
	fp3 := ComputeFingerprint(res)
	if !FingerprintsEqual(fp2, fp3) {
		t.Fatal("无内容变化时指纹应稳定")
	}
}

func containsPathDir(entries []FingerprintEntry, dir string) bool {
	clean := filepath.Clean(dir)
	for i := range entries {
		if entries[i].Path == clean {
			return true
		}
	}
	return false
}
