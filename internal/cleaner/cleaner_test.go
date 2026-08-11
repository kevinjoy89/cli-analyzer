package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

func mkScanResult(cleanables ...scanner.Cleanable) *scanner.ScanResult {
	return &scanner.ScanResult{
		Tools: []scanner.Tool{{Name: "t", Cleanables: cleanables}},
	}
}

// useTempTrash 将内置回收站根指向临时目录；不预创建目录，让测试覆盖
// 首次清理时回收站根尚不存在的路径，且避免写真实回收站
func useTempTrash(t *testing.T) {
	t.Helper()
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := trash.Root
	trash.Root = func() string { return rootDir }
	t.Cleanup(func() { trash.Root = orig })
}

func TestCleanRejectsUserTier(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|user|" + p, Tool: "t", Path: p, Bytes: 10, Tier: scanner.TierUser, Kind: "config",
	})
	report := Clean(res, []string{"t|user|" + p}, false)
	if len(report.Deleted) != 0 {
		t.Fatalf("USER item deleted: %v", report.Deleted)
	}
	if len(report.Skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %v", report.Skipped)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("USER dir was removed: %v", err)
	}
}

func TestCleanDeletesSafeItem(t *testing.T) {
	useTempTrash(t)
	td := t.TempDir()
	target := filepath.Join(td, "cache")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|cache|" + target, Tool: "t", Path: target, Bytes: 123, Tier: scanner.TierSafe, Kind: "cache",
	})
	report := Clean(res, []string{"t|cache|" + target}, false)
	if len(report.Deleted) != 1 || report.Freed != 123 {
		t.Fatalf("deleted=%v freed=%d", report.Deleted, report.Freed)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("safe dir still exists: %v", err)
	}
}

func TestCleanDryRunDoesNotDelete(t *testing.T) {
	td := t.TempDir()
	target := filepath.Join(td, "cache")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|cache|" + target, Tool: "t", Path: target, Bytes: 123, Tier: scanner.TierSafe, Kind: "cache",
	})
	report := Clean(res, []string{"t|cache|" + target}, true)
	if len(report.Deleted) != 1 || report.Freed != 123 {
		t.Fatalf("dry-run should report plan: %v", report.Deleted)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("dry-run removed the dir: %v", err)
	}
}

func TestCleanGuardRejectsSystemRoot(t *testing.T) {
	res := mkScanResult(scanner.Cleanable{
		ID: "t|old|/usr/bin", Tool: "t", Path: "/usr/bin", Bytes: 1, Tier: scanner.TierSafe, Kind: "old-version",
	})
	report := Clean(res, []string{"t|old|/usr/bin"}, false)
	if len(report.Deleted) != 0 || len(report.Skipped) != 1 {
		t.Fatalf("system root not guarded: %v", report)
	}
}

func TestCleanGuardRejectsCurrentVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "current")
	if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|old|" + p, Tool: "t", Path: p, Bytes: 1, Tier: scanner.TierSafe,
		Kind: "old-version", CurrentPath: p,
	})
	report := Clean(res, []string{"t|old|" + p}, false)
	if len(report.Deleted) != 0 || len(report.Skipped) != 1 {
		t.Fatalf("current version not guarded: %v", report)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("current file removed: %v", err)
	}
}

// TestCleanDeletesSubItem verifies a child of a SAFE cleanable can be deleted on
// its own, leaving the parent and its other children intact.
func TestCleanDeletesSubItem(t *testing.T) {
	useTempTrash(t)
	td := t.TempDir()
	parent := filepath.Join(td, "cache")
	child := filepath.Join(parent, "keep-me")
	drop := filepath.Join(parent, "drop-me")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(drop, 0o755); err != nil {
		t.Fatal(err)
	}
	subID := "t|cache|" + parent + "::" + drop
	res := mkScanResult(scanner.Cleanable{
		ID: "t|cache|" + parent, Tool: "t", Path: parent, Bytes: 200, Tier: scanner.TierSafe, Kind: "cache",
		Sub: []scanner.SubEntry{
			{Path: child, Bytes: 100, ID: "t|cache|" + parent + "::" + child},
			{Path: drop, Bytes: 100, ID: subID},
		},
	})
	report := Clean(res, []string{subID}, false)
	if len(report.Deleted) != 1 || report.Freed != 100 {
		t.Fatalf("deleted=%v freed=%d", report.Deleted, report.Freed)
	}
	if _, err := os.Stat(drop); !os.IsNotExist(err) {
		t.Fatalf("child dir still exists: %v", err)
	}
	if _, err := os.Stat(child); err != nil {
		t.Fatalf("sibling was removed: %v", err)
	}
	if _, err := os.Stat(parent); err != nil {
		t.Fatalf("parent was removed: %v", err)
	}
}

// TestCleanSubRejectsOutsideParent feeds a fabricated sub path that escapes the
// parent dir; guardSub must refuse it even though it is in the scan data.
func TestCleanSubRejectsOutsideParent(t *testing.T) {
	td := t.TempDir()
	parent := filepath.Join(td, "cache")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(td, "sibling")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|cache|" + parent, Tool: "t", Path: parent, Bytes: 10, Tier: scanner.TierSafe, Kind: "cache",
		Sub: []scanner.SubEntry{
			{Path: outside, Bytes: 99, ID: "t|cache|" + parent + "::" + outside},
		},
	})
	report := Clean(res, []string{"t|cache|" + parent + "::" + outside}, false)
	if len(report.Deleted) != 0 || len(report.Skipped) != 1 {
		t.Fatalf("outside-parent sub not guarded: %v", report)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("sibling was removed: %v", err)
	}
}

// TestCleanGuardRejectsTrashRoot verifies the built-in trash root itself can
// never be a clean target, even though it is technically a deletable dir.
func TestCleanGuardRejectsTrashRoot(t *testing.T) {
	useTempTrash(t)
	res := mkScanResult(scanner.Cleanable{
		ID: "t|old|" + trash.Root(), Tool: "t", Path: trash.Root(), Bytes: 1, Tier: scanner.TierSafe, Kind: "old-version",
	})
	report := Clean(res, []string{"t|old|" + trash.Root()}, false)
	if len(report.Deleted) != 0 || len(report.Skipped) != 1 {
		t.Fatalf("trash root not guarded: %v", report)
	}
}

// TestCleanPermanentSkipsTrash verifies the immediate-delete mode removes the
// item without routing it through the built-in trash.
func TestCleanPermanentSkipsTrash(t *testing.T) {
	useTempTrash(t)
	td := t.TempDir()
	target := filepath.Join(td, "cache")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|cache|" + target, Tool: "t", Path: target, Bytes: 10, Tier: scanner.TierSafe, Kind: "cache",
	})
	report := CleanPermanent(res, []string{"t|cache|" + target}, false)
	if len(report.Deleted) != 1 {
		t.Fatalf("deleted=%v", report.Deleted)
	}
	items, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("CleanPermanent 后回收站仍有 %d 项", len(items))
	}
}

// TestCleanSubUserParentRejected verifies a child of a USER cleanable is never
// deleted, mirroring the parent's hard gate.
func TestCleanSubUserParentRejected(t *testing.T) {
	td := t.TempDir()
	parent := filepath.Join(td, "data")
	child := filepath.Join(parent, "x")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|data|" + parent, Tool: "t", Path: parent, Bytes: 10, Tier: scanner.TierUser, Kind: "data",
		Sub: []scanner.SubEntry{
			{Path: child, Bytes: 9, ID: "t|data|" + parent + "::" + child},
		},
	})
	report := Clean(res, []string{"t|data|" + parent + "::" + child}, false)
	if len(report.Deleted) != 0 || len(report.Skipped) != 1 {
		t.Fatalf("USER-parent sub deleted: %v", report)
	}
	if _, err := os.Stat(child); err != nil {
		t.Fatalf("child was removed: %v", err)
	}
}
