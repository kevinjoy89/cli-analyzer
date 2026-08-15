package trash

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"cli-analyzer/internal/config"
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

// TestRemoveExpiredFallbackKeepsInfoOnFailure 验证系统回收站不可用且降级
// 删除也失败（权限）时，info.json 与项目目录必须保留（待下次重试）——
// 此前忽略 RemoveAll 错误仍删 info，数据变成不可见也不可恢复的孤儿。
func TestRemoveExpiredFallbackKeepsInfoOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		// chmod 0o555 依赖 POSIX 权限语义（Windows 目录的只读位不阻止
		// RemoveDirectory），Windows 上无法用此方式模拟删除失败
		t.Skip("POSIX 权限语义用例")
	}
	withTrashRoot(t)
	origSys := systemTrashFn
	systemTrashFn = func(string) error { return errors.New("no system trash") }
	t.Cleanup(func() { systemTrashFn = origSys })

	itemDir := filepath.Join(Root(), "expired_item")
	if err := os.MkdirAll(filepath.Join(itemDir, "_data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(itemDir, "_data", "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeInfo(itemDir, Item{
		ID: "expired_item", Original: "/orig", Tool: "t", Kind: "cache",
		Bytes: 1, TrashedAt: "2020-01-01T00:00:00Z", ExpiresAt: "2020-01-02T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}
	// 数据目录只读 → 降级 RemoveAll 失败
	if err := os.Chmod(filepath.Join(itemDir, "_data"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(itemDir, "_data"), 0o755) })

	err := removeExpired(itemDir, config.ExpireActionSystemTrash)
	if err == nil {
		t.Fatal("removeExpired 应返回降级删除错误")
	}
	if _, err := os.Stat(infoPathOf(itemDir)); err != nil {
		t.Errorf("降级删除失败时 info.json 必须保留, stat err=%v", err)
	}
}

// TestTrashWriteInfoFailureRollsBack 验证元数据落盘失败（磁盘满/权限）时
// 数据必须回滚到原路径：此前 Rename 已把数据移入回收站但 info.json 写失败，
// 项目既不在回收站列表、也无法恢复，变成永久孤儿数据。
func TestTrashWriteInfoFailureRollsBack(t *testing.T) {
	withTrashRoot(t)
	origWrite := writeInfoFn
	writeInfoFn = func(_ string, _ Item) error { return errors.New("simulated info write failure") }
	t.Cleanup(func() { writeInfoFn = origWrite })

	src := mkSource(t, "data")
	err := Trash(src, Item{Tool: "t", Kind: "cache", Bytes: 1})
	if err == nil {
		t.Fatal("Trash 应返回元数据写入错误")
	}
	// 数据必须回到原路径，不能残留在回收站
	if _, err := os.Stat(src); err != nil {
		t.Errorf("数据应回滚到原路径, stat err=%v", err)
	}
	if got := itemIDs(t); len(got) != 0 {
		t.Errorf("回滚后回收站不应残留项目, got %v", got)
	}
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
	// Purge 是"清空/彻底删除"的显式语义（CLI empty / GUI Delete permanently），
	// 必须永久删除而非转系统回收站；此前复用过期配置导致空间未释放。
	if len(trashed) != 0 {
		t.Errorf("Purge 应永久删除（不经过系统回收站），实际调用系统回收站 %d 次", len(trashed))
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
		{"cache", "/Users/wei/.npm/_logs", "logs"}, // 修复前旧条目：日志误记成缓存
		{"cache", "/Users/wei/.npm/debug-0.log", "logs"},
		{"cache", "/Users/wei/.npm/_cacache", "cache"},           // 真缓存不动
		{"logs", "/Users/wei/.npm/_logs", "logs"},                // 已是精确类型不动
		{"download", "/Users/wei/x/downloads/a.log", "download"}, // 非 cache 类型不干预
		{"cache", "/Users/wei/foo/catalog", "cache"},             // catalog 不是日志
	}
	for _, c := range cases {
		if got := refineKind(c.kind, c.original); got != c.want {
			t.Errorf("refineKind(%q, %q) = %q, want %q", c.kind, c.original, got, c.want)
		}
	}
}

// TestRestoreRecreatesParentDir 验证恢复时原父目录已被删除（用户清理了
// 上级目录）也能恢复：os.Rename 到不存在的父目录会失败，恢复应重新创建
// 父目录（此前恢复直接报错，数据困在回收站）。
func TestRestoreRecreatesParentDir(t *testing.T) {
	withTrashRoot(t)
	parent := t.TempDir()
	src := filepath.Join(parent, "app-data")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Trash(src, Item{Tool: "t", Kind: "data", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
	// 用户随后删除了原父目录
	if err := os.RemoveAll(parent); err != nil {
		t.Fatal(err)
	}
	ids := itemIDs(t)
	if len(ids) != 1 {
		t.Fatalf("items = %v", ids)
	}
	restored, err := Restore(ids[0])
	if err != nil {
		t.Fatalf("Restore 在父目录缺失时应重建父目录: %v", err)
	}
	if restored != src {
		t.Errorf("restored = %q, want %q", restored, src)
	}
	if _, err := os.Stat(filepath.Join(src, "x")); err != nil {
		t.Errorf("恢复后数据不可读: %v", err)
	}
}
