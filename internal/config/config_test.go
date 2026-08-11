package config

import (
	"os"
	"testing"
)

// withTempRoot 将 dataRoot 指向临时目录，保证测试隔离真实文件系统
func withTempRoot(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	orig := dataRoot
	dataRoot = func() string { return root }
	t.Cleanup(func() { dataRoot = orig })
}

func TestLoadReturnsDefaultsWhenMissing(t *testing.T) {
	withTempRoot(t)
	cfg := Load()
	if cfg.Trash.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7", cfg.Trash.RetentionDays)
	}
	if cfg.Trash.ExpireAction != ExpireActionSystemTrash {
		t.Errorf("ExpireAction = %q, want %q", cfg.Trash.ExpireAction, ExpireActionSystemTrash)
	}
	if !cfg.Trash.UseTrash {
		t.Error("UseTrash = false, want true")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempRoot(t)
	cfg := Default()
	cfg.Trash.RetentionDays = 30
	cfg.Trash.ExpireAction = ExpireActionPermanent
	cfg.Trash.UseTrash = false
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := Load()
	if got.Trash != cfg.Trash {
		t.Errorf("round trip mismatch: got %+v, want %+v", got.Trash, cfg.Trash)
	}
}

func TestInvalidValuesFallBackToDefaults(t *testing.T) {
	withTempRoot(t)
	// 合法 JSON 但含非法字段值
	if err := os.WriteFile(Path(), []byte(`{"trash":{"retentionDays":-1,"expireAction":"bogus"}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Load()
	if cfg.Trash.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7", cfg.Trash.RetentionDays)
	}
	if cfg.Trash.ExpireAction != ExpireActionSystemTrash {
		t.Errorf("ExpireAction = %q, want %q", cfg.Trash.ExpireAction, ExpireActionSystemTrash)
	}
}

func TestCorruptJSONFallsBackToDefaults(t *testing.T) {
	withTempRoot(t)
	if err := os.WriteFile(Path(), []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Load()
	if cfg.Trash.RetentionDays != 7 {
		t.Errorf("RetentionDays = %d, want 7", cfg.Trash.RetentionDays)
	}
}
