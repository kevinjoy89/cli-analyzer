package trash

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withTrashRoot 将回收站根指向临时目录；故意不预创建目录，让所有测试
// 天然覆盖"首次清理时回收站根尚不存在"的真实路径
func withTrashRoot(t *testing.T) {
	t.Helper()
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := Root
	Root = func() string { return rootDir }
	t.Cleanup(func() { Root = orig })
}

// mkSource 创建一个含单个文件的源目录
func mkSource(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// itemIDs 返回回收站当前全部项目 id
func itemIDs(t *testing.T) []string {
	t.Helper()
	items, err := List()
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

// TestTrashCreatesTrashRoot verifies first-time cleanup works even when the
// trash root directory does not exist yet (regression: sameFS used to fail
// because os.Stat on the missing root errored, refusing the move).
func TestTrashCreatesTrashRoot(t *testing.T) {
	// 不预创建回收站根目录，模拟首次清理
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := Root
	Root = func() string { return rootDir }
	t.Cleanup(func() { Root = orig })
	src := mkSource(t, "first-clean")
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatalf("Trash: %v", err)
	}
	if _, err := os.Stat(rootDir); err != nil {
		t.Fatalf("回收站根目录未被自动创建: %v", err)
	}
	if got := itemIDs(t); len(got) != 1 {
		t.Fatalf("回收站项目数 = %d, want 1", len(got))
	}
}

func TestTrashRestoreRoundTrip(t *testing.T) {
	withTrashRoot(t)
	src := mkSource(t, "npm-cache")
	if err := Trash(src, Item{Tool: "npm", Kind: "cache", Bytes: 10}); err != nil {
		t.Fatalf("Trash: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("移入后原路径仍存在: %v", err)
	}
	if got := itemIDs(t); len(got) != 1 {
		t.Fatalf("回收站项目数 = %d, want 1", len(got))
	}
	restored, err := Restore(itemIDs(t)[0])
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored != src {
		t.Errorf("restored = %q, want %q", restored, src)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("还原后原路径不存在: %v", err)
	}
	if got := itemIDs(t); len(got) != 0 {
		t.Errorf("还原后回收站仍残留 %d 项", len(got))
	}
}

func TestTrashCrossFilesystemRejected(t *testing.T) {
	withTrashRoot(t)
	src := mkSource(t, "other-fs")
	orig := devOfFn
	devOfFn = func(p string) (uint64, error) {
		if p == src {
			return 99, nil // 源设备号
		}
		return 1, nil // 回收站设备号
	}
	defer func() { devOfFn = orig }()
	if err := Trash(src, Item{}); !errors.Is(err, ErrCrossFilesystem) {
		t.Fatalf("Trash err = %v, want ErrCrossFilesystem", err)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("拒绝后原路径被改动: %v", err)
	}
}

func TestRestoreConflictRenames(t *testing.T) {
	withTrashRoot(t)
	src := mkSource(t, "conflict")
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	// 重建原路径制造冲突
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(itemIDs(t)[0])
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored == src {
		t.Errorf("restored 不应等于被占用的原路径 %q", src)
	}
	if _, err := os.Stat(restored); err != nil {
		t.Fatalf("改名还原的目标不存在: %v", err)
	}
}

func TestSweepClearsExpired(t *testing.T) {
	withTrashRoot(t)
	src := mkSource(t, "expiring")
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	// 把到期时间改为过去
	dirs, _ := os.ReadDir(Root())
	itemDir := filepath.Join(Root(), dirs[0].Name())
	meta, err := readInfo(itemDir)
	if err != nil {
		t.Fatal(err)
	}
	meta.ExpiresAt = time.Now().Add(-time.Hour).Format(time.RFC3339)
	if err := writeInfo(itemDir, *meta); err != nil {
		t.Fatal(err)
	}
	// 模拟系统回收站不可用，验证降级为彻底删除
	orig := systemTrashFn
	systemTrashFn = func(string) error { return errNoSystemTrash }
	defer func() { systemTrashFn = orig }()
	removed, errs := Sweep()
	if removed != 1 {
		t.Fatalf("Sweep removed = %d, want 1; errs=%v", removed, errs)
	}
	if got := itemIDs(t); len(got) != 0 {
		t.Errorf("Sweep 后仍有 %d 项", len(got))
	}
}

func TestSweepKeepsUnexpired(t *testing.T) {
	withTrashRoot(t)
	src := mkSource(t, "keep")
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	removed, errs := Sweep()
	if removed != 0 {
		t.Fatalf("未过期项被清除: %d, errs=%v", removed, errs)
	}
}

func TestInfoStats(t *testing.T) {
	withTrashRoot(t)
	a := mkSource(t, "a")
	b := mkSource(t, "b")
	if err := Trash(a, Item{Tool: "t", Kind: "cache", Bytes: 100}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // 保证时间戳不同
	if err := Trash(b, Item{Tool: "t", Kind: "cache", Bytes: 200}); err != nil {
		t.Fatal(err)
	}
	info := Info()
	if info.Items != 2 {
		t.Errorf("Items = %d, want 2", info.Items)
	}
	if info.TotalBytes != 300 {
		t.Errorf("TotalBytes = %d, want 300", info.TotalBytes)
	}
	if info.EarliestExp == "" {
		t.Error("EarliestExp 为空")
	}
}

func TestPurge(t *testing.T) {
	withTrashRoot(t)
	var trashed []string
	orig := systemTrashFn
	systemTrashFn = func(p string) error { trashed = append(trashed, p); return os.RemoveAll(p) }
	defer func() { systemTrashFn = orig }()
	src := mkSource(t, "purge-me")
	if err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	deleted, errs := Purge(itemIDs(t))
	if len(deleted) != 1 || len(errs) != 0 {
		t.Fatalf("deleted=%v errs=%v", deleted, errs)
	}
	if got := itemIDs(t); len(got) != 0 {
		t.Errorf("Purge 后仍有 %d 项", len(got))
	}
	// 默认配置（system-trash）下 Purge 应移入系统回收站，而非直接删除
	if len(trashed) != 1 {
		t.Errorf("默认配置下 Purge 应调用系统回收站，实际调用 %d 次", len(trashed))
	}
}

func TestPurgeRejectsBadID(t *testing.T) {
	withTrashRoot(t)
	deleted, errs := Purge([]string{"../evil"})
	if len(deleted) != 0 || len(errs) != 1 {
		t.Fatalf("deleted=%v errs=%v", deleted, errs)
	}
}

// TestListEmptySerializesAsArray verifies an empty trash serializes to []
// rather than null (regression: nil slice marshals to "null" and crashes the
// frontend's JSON.parse when the trash panel refreshes after emptying).
func TestListEmptySerializesAsArray(t *testing.T) {
	withTrashRoot(t)
	items, err := List()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(items)
	if string(b) != "[]" {
		t.Errorf("空回收站应序列化为 [], got %s", b)
	}
}

func TestRefineKind(t *testing.T) {
	cases := []struct{ kind, original, want string }{
		{"cache", "/Users/wei/.npm/_logs", "logs"},   // 修复前旧条目：日志误记成缓存
		{"cache", "/Users/wei/.npm/debug-0.log", "logs"},
		{"cache", "/Users/wei/.npm/_cacache", "cache"}, // 真缓存不动
		{"logs", "/Users/wei/.npm/_logs", "logs"},      // 已是精确类型不动
		{"download", "/Users/wei/x/downloads/a.log", "download"}, // 非 cache 类型不干预
		{"cache", "/Users/wei/foo/catalog", "cache"},   // catalog 不是日志
	}
	for _, c := range cases {
		if got := refineKind(c.kind, c.original); got != c.want {
			t.Errorf("refineKind(%q, %q) = %q, want %q", c.kind, c.original, got, c.want)
		}
	}
}
