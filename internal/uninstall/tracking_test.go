package uninstall

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTracking 把跟踪文件指向临时目录。
func setupTracking(t *testing.T) {
	t.Helper()
	// 目录只取一次：t.TempDir() 每次调用都返回**新**目录，写在闭包里会让
	// 读和写落到不同文件上。
	dir := t.TempDir()
	orig := trackingPath
	trackingPath = func() string { return filepath.Join(dir, "events.json") }
	t.Cleanup(func() { trackingPath = orig })
}

// TestRecordAndPending 验证卸载事件落盘后能在 Pending 里被看到。
func TestRecordAndPending(t *testing.T) {
	setupTracking(t)
	dir := t.TempDir()
	residue := filepath.Join(dir, ".mytool")
	if err := os.MkdirAll(residue, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(residue, "cfg.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordEvent("mytool", []string{residue}); err != nil {
		t.Fatal(err)
	}

	// 刚卸载（0 天）不算待处理
	if p, _ := Pending(7); len(p) != 0 {
		t.Errorf("fresh event should not be pending, got %v", p)
	}
	// minDays=0 时应出现，且字节数为现算值
	p, err := Pending(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 1 || p[0].Tool != "mytool" {
		t.Fatalf("pending = %+v", p)
	}
	if p[0].Bytes <= 0 {
		t.Errorf("bytes should be measured now, got %d", p[0].Bytes)
	}
}

// TestPendingSkipsCleanedResidue 验证残留清干净后不再提醒。
func TestPendingSkipsCleanedResidue(t *testing.T) {
	setupTracking(t)
	dir := t.TempDir()
	residue := filepath.Join(dir, ".gone")
	if err := os.MkdirAll(residue, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RecordEvent("gonetool", []string{residue}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(residue); err != nil {
		t.Fatal(err)
	}
	p, err := Pending(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 0 {
		t.Errorf("cleaned residue should not be reported, got %+v", p)
	}
}

// TestForgetRemovesEvent 验证 --forget 收尾。
func TestForgetRemovesEvent(t *testing.T) {
	setupTracking(t)
	dir := t.TempDir()
	residue := filepath.Join(dir, ".x")
	if err := os.MkdirAll(residue, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RecordEvent("x", []string{residue}); err != nil {
		t.Fatal(err)
	}
	if err := Forget("x"); err != nil {
		t.Fatal(err)
	}
	if p, _ := Pending(0); len(p) != 0 {
		t.Errorf("forgotten tool still pending: %+v", p)
	}
}

// TestRecordEventReplacesSameTool 验证同一工具重复卸载时保留最新一条。
func TestRecordEventReplacesSameTool(t *testing.T) {
	setupTracking(t)
	if err := RecordEvent("dup", []string{"/a"}); err != nil {
		t.Fatal(err)
	}
	if err := RecordEvent("dup", []string{"/b"}); err != nil {
		t.Fatal(err)
	}
	events, err := loadEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Residue[0] != "/b" {
		t.Errorf("events = %+v, want single latest", events)
	}
}

// TestCorruptTrackingFileIsIgnored 验证跟踪文件损坏时不影响卸载流程。
func TestCorruptTrackingFileIsIgnored(t *testing.T) {
	setupTracking(t)
	if err := os.WriteFile(trackingPath(), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadEvents(); err != nil {
		t.Errorf("corrupt file should be ignored, got %v", err)
	}
	if err := RecordEvent("y", nil); err != nil {
		t.Errorf("record over corrupt file should succeed, got %v", err)
	}
}
