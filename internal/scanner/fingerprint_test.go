package scanner

import (
	"os"
	"path/filepath"
	"testing"
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
	if len(fp) != 2 {
		t.Fatalf("指纹条目数 = %d, want 2 (binary + data dir)", len(fp))
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
	if binFp.MTime != st.ModTime().Unix() || binFp.Size != st.Size() || binFp.IsDir {
		t.Fatalf("指纹与 os.Stat 不一致: %+v vs mtime=%d size=%d", binFp, st.ModTime().Unix(), st.Size())
	}
}
