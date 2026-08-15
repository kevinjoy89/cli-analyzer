package cleaner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/i18n"

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

// TestCleanDeletesUserTier 验证两级门槛移除后 USER 级（config/data）项目与
// SAFE 级同等可处置：Tier 只是信息标签，不再导致 Skipped。默认走内置回收站。
func TestCleanDeletesUserTier(t *testing.T) {
	useTempTrash(t)
	p := filepath.Join(t.TempDir(), "config")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	res := mkScanResult(scanner.Cleanable{
		ID: "t|user|" + p, Tool: "t", Path: p, Bytes: 10, Tier: scanner.TierUser, Kind: "config",
	})
	report := Clean(res, []string{"t|user|" + p}, false)
	if len(report.Deleted) != 1 || report.Freed != 10 {
		t.Fatalf("USER item should be deleted now: deleted=%v freed=%d", report.Deleted, report.Freed)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("USER item must not be skipped (tier is a label): %v", report.Skipped)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("USER dir was not moved to trash: %v", err)
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

// TestCleanSubUserParentDeleted 验证 USER 级可处置项的子项同样可独立删除
// （门槛移除后与 SAFE 项行为一致），且仍受 guardSub 的父目录边界保护。
func TestCleanSubUserParentDeleted(t *testing.T) {
	useTempTrash(t)
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
	if len(report.Deleted) != 1 {
		t.Fatalf("USER-parent sub should be deleted now: %v", report)
	}
	if _, err := os.Stat(child); !os.IsNotExist(err) {
		t.Fatalf("child was not removed: %v", err)
	}
}

// TestCleanSubUsesPreciseKind 验证子项移入回收站时使用扫描器给出的精确类型
// （~/.npm/_logs → logs，而非父项的 cache）；旧缓存子项无 Kind 字段时回退
// 父项类型，回收站条目类型随之正确。
func TestCleanSubUsesPreciseKind(t *testing.T) {
	useTempTrash(t)
	td := t.TempDir()
	parent := filepath.Join(td, "npm")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	logsDir := filepath.Join(parent, "_logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentID := "t|cache|" + parent

	// 子项带精确类型：入库 logs
	res := mkScanResult(scanner.Cleanable{
		ID: parentID, Tool: "t", Path: parent, Bytes: 100, Tier: scanner.TierSafe, Kind: "cache",
		Sub: []scanner.SubEntry{
			{Path: logsDir, Bytes: 100, ID: parentID + "::" + logsDir, Kind: "logs"},
		},
	})
	report := Clean(res, []string{parentID + "::" + logsDir}, false)
	if len(report.Deleted) != 1 {
		t.Fatalf("deleted=%v", report.Deleted)
	}
	items, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Kind != "logs" {
		t.Fatalf("trash item kind = %+v, want logs", items)
	}

	// 旧缓存子项无 Kind：回退父项 cache
	drop := filepath.Join(parent, "drop")
	if err := os.MkdirAll(drop, 0o755); err != nil {
		t.Fatal(err)
	}
	res2 := mkScanResult(scanner.Cleanable{
		ID: parentID, Tool: "t", Path: parent, Bytes: 100, Tier: scanner.TierSafe, Kind: "cache",
		Sub: []scanner.SubEntry{
			{Path: drop, Bytes: 100, ID: parentID + "::" + drop},
		},
	})
	report2 := Clean(res2, []string{parentID + "::" + drop}, false)
	if len(report2.Deleted) != 1 {
		t.Fatalf("deleted=%v", report2.Deleted)
	}
	items2, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, it := range items2 {
		kinds[it.Kind] = true
	}
	if !kinds["logs"] || !kinds["cache"] {
		t.Fatalf("trash kinds = %v, want logs and cache", kinds)
	}
}

// TestCleanGuardRejectsWindowsRoots 验证 Windows 系统根被 guard 拒绝
// （大小写不敏感纵深防御）：任何来源的删除请求都不能命中系统目录。
func TestCleanGuardRejectsWindowsRoots(t *testing.T) {
	for _, p := range []string{`C:\Windows`, `c:\windows`, `C:\Program Files`, `C:\PROGRAM FILES (X86)`} {
		if reason := guard(&scanner.Cleanable{Path: p, Tier: scanner.TierSafe}); reason == "" {
			t.Errorf("guard(%q) 应拒绝 Windows 系统根", p)
		}
	}
	// 合法 cleanable（用户缓存）仍放行（unix 上无法真实创建，直接检查路径判断）
	if reason := guard(&scanner.Cleanable{Path: filepath.Join(t.TempDir(), "cache"), Tier: scanner.TierSafe}); reason != "" {
		t.Errorf("合法路径被拒绝: %v", reason)
	}
}

// TestCleanMessagesLocalized 验证 cleaner 的用户可见消息走 i18n（三语有 key、
// 输出不含硬编码英文原文）——此前 guard/Skipped 消息全部英文硬编码，
// 中文界面用户看到英文提示。门槛移除后 Skipped 只剩 guard 消息，此处守护
// guard 消息的本地化。
func TestCleanMessagesLocalized(t *testing.T) {
	i18n.SetLocale("zh-CN")
	t.Cleanup(func() { i18n.SetLocale("zh-CN") })
	// guard 消息同样本地化
	if reason := guard(&scanner.Cleanable{Path: "/", Tier: scanner.TierSafe}); strings.Contains(reason, "forbidden system root") {
		t.Errorf("guard 消息未本地化: %q", reason)
	}
}
