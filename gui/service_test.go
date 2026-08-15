package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// useTempTrash 将内置回收站根指向临时目录，避免测试写入真实回收站
func useTempTrash(t *testing.T) {
	t.Helper()
	rootDir := filepath.Join(t.TempDir(), "trash")
	orig := trash.Root
	trash.Root = func() string { return rootDir }
	t.Cleanup(func() { trash.Root = orig })
}

// TestOrphanTrashRejectsNonUnattributed 验证 OrphanTrash 只允许移入当前扫描
// 结果中 Unattributed 列表内的路径：前端（或任何调用方）传入任意路径时，
// 不在列表内的路径必须被拒绝，不能直接移入回收站。
func TestOrphanTrashRejectsNonUnattributed(t *testing.T) {
	useTempTrash(t)
	s := NewScannerService()
	base := t.TempDir()
	// 合法孤儿：出现在扫描结果的 Unattributed 中
	realOrphan := filepath.Join(base, "real-orphan")
	if err := os.MkdirAll(realOrphan, 0o755); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.last = &scanner.ScanResult{
		Unattributed: []scanner.DataDir{{Path: realOrphan, Bytes: 1}},
	}
	s.mu.Unlock()

	// 非法路径：不在 Unattributed 中（模拟调用方传入任意目录）
	evil := filepath.Join(base, "evil-orphan")
	if err := os.MkdirAll(evil, 0o755); err != nil {
		t.Fatal(err)
	}

	out := s.OrphanTrash([]string{realOrphan, evil})
	var r struct {
		Trashed []string `json:"trashed"`
		Errors  []string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v; raw=%s", err, out)
	}
	if len(r.Trashed) != 1 || r.Trashed[0] != realOrphan {
		t.Errorf("trashed = %v, want only %s", r.Trashed, realOrphan)
	}
	if len(r.Errors) == 0 {
		t.Errorf("errors 应为空或包含拒绝信息, got %v", r.Errors)
	}
	// 合法路径已被移入回收站；非法路径必须原封不动
	if _, err := os.Stat(realOrphan); !os.IsNotExist(err) {
		t.Errorf("unattributed dir should have been trashed, stat err=%v", err)
	}
	if _, err := os.Stat(evil); err != nil {
		t.Errorf("non-unattributed dir must NOT be touched, stat err=%v", err)
	}
}

// TestOrphanTrashNoScanRejectsAll 验证无扫描结果（s.last 为空）时全部拒绝，
// 防止空快照误放行任意路径。
func TestOrphanTrashNoScanRejectsAll(t *testing.T) {
	useTempTrash(t)
	s := NewScannerService()
	base := t.TempDir()
	p := filepath.Join(base, "orphan")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	out := s.OrphanTrash([]string{p})
	var r struct {
		Trashed []string `json:"trashed"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Trashed) != 0 {
		t.Errorf("无扫描结果时不应移入任何路径, trashed=%v", r.Trashed)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("无扫描结果时路径必须保持原样, stat err=%v", err)
	}
}
