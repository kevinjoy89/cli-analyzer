package trash

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/config"
)

// setupTrashRoot 把回收站根指向临时目录，并固定回收站配置。
func setupTrashRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	orig := Root
	Root = func() string { return root }
	t.Cleanup(func() { Root = orig })
	restore := config.SetDataRoot(t.TempDir())
	t.Cleanup(restore)
	return root
}

// TestTrashWritesManifest 验证移入时写下清单（文件数 + 顶层项）。
func TestTrashWritesManifest(t *testing.T) {
	setupTrashRoot(t)
	src := filepath.Join(t.TempDir(), "payload")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"a.txt", "sub/b.txt", "sub/c.txt"} {
		if err := os.WriteFile(filepath.Join(src, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 3}); err != nil {
		t.Fatal(err)
	}
	items, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	m, ok := ManifestOf(items[0].ID)
	if !ok {
		t.Fatal("manifest missing")
	}
	if m.Files != 3 {
		t.Errorf("manifest files = %d, want 3", m.Files)
	}
	if len(m.TopPaths) != 2 {
		t.Errorf("topPaths = %v, want 2 entries", m.TopPaths)
	}
}

// TestRestoreVerifyMatches 验证恢复后核对通过。
func TestRestoreVerifyMatches(t *testing.T) {
	setupTrashRoot(t)
	base := t.TempDir()
	src := filepath.Join(base, "payload")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	items, _ := List()
	id := items[0].ID
	manifest, hasManifest := ManifestOf(id) // Restore 之前取
	restored, err := Restore(id)
	if err != nil {
		t.Fatal(err)
	}
	v := VerifyItem(restored, manifest, hasManifest)
	if !v.Checked || !v.OK {
		t.Errorf("verify = %+v, want checked+ok", v)
	}
	if v.Files != 1 {
		t.Errorf("files = %d, want 1", v.Files)
	}
}

// TestVerifyItemDetectsLoss 验证内容缺失时核对不通过。
func TestVerifyItemDetectsLoss(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kept.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{Files: 2, TopPaths: []string{"kept.txt", "lost.txt"}}
	v := VerifyItem(dir, manifest, true)
	if v.OK {
		t.Error("missing file should fail verification")
	}
	if len(v.Missing) != 1 || v.Missing[0] != "lost.txt" {
		t.Errorf("missing = %v", v.Missing)
	}
}

// TestVerifyItemWithoutManifest 验证旧项目（无清单）不算失败。
func TestVerifyItemWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	v := VerifyItem(dir, Manifest{}, false)
	if v.Checked {
		t.Error("no manifest should mean unchecked")
	}
	if !v.OK {
		t.Error("unchecked should not be a failure")
	}
}
