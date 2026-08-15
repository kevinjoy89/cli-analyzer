package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/trash"
)

// withTrashRoot 将内置回收站根指向临时目录（cli 层测试隔离用）
func withTrashRoot(t *testing.T) {
	t.Helper()
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := trash.Root
	trash.Root = func() string { return rootDir }
	t.Cleanup(func() { trash.Root = orig })
}

// seedTrashItem 在回收站中造一个可删除项目
func seedTrashItem(t *testing.T) {
	t.Helper()
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := trash.Trash(src, trash.Item{Tool: "t", Kind: "cache", Bytes: 1}); err != nil {
		t.Fatal(err)
	}
}

// TestTrashEmptyRequiresConfirmation 验证 `trash empty` 是破坏性操作：
// 无 --yes 且 stdin 无确认输入（EOF）时必须拒绝删除（此前无确认直接清空）。
func TestTrashEmptyRequiresConfirmation(t *testing.T) {
	pinZhCN(t)
	withTrashRoot(t)
	seedTrashItem(t)

	oldStdin := stdin
	stdin = strings.NewReader("") // EOF → confirmPrompt 返回 false
	t.Cleanup(func() { stdin = oldStdin })
	captureStdout(t)

	if code := runTrash([]string{"empty"}); code != 0 {
		t.Fatalf("code = %d, want 0 (cancelled)", code)
	}
	items, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("确认被拒绝后项目不应被删除，剩余 %d 项", len(items))
	}
}

// TestTrashEmptyYesFlag 验证 `trash empty --yes` 清空全部项目
// （永久删除语义由 trash 包 TestPurge 锁定：不经过系统回收站）
func TestTrashEmptyYesFlag(t *testing.T) {
	pinZhCN(t)
	withTrashRoot(t)
	seedTrashItem(t)
	captureStdout(t)

	if code := runTrash([]string{"empty", "--yes"}); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	items, _ := trash.List()
	if len(items) != 0 {
		t.Errorf("--yes 清空后仍有 %d 项", len(items))
	}
}
