package config

import (
	"os"
	"sync"
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

// ---- update config ----

func TestUpdateDefaultsEnabled(t *testing.T) {
	withTempRoot(t)
	cfg := Default()
	if !cfg.Update.CheckUpdatesEnabled() {
		t.Error("CheckUpdatesEnabled() = false, want true (default)")
	}
}

// 旧 config.json（无 update 段）加载后应兼容：CheckUpdates 为 nil → 默认 true
func TestLegacyConfigWithoutUpdateSection(t *testing.T) {
	withTempRoot(t)
	if err := os.WriteFile(Path(), []byte(`{"trash":{"retentionDays":7}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Load()
	if !cfg.Update.CheckUpdatesEnabled() {
		t.Error("legacy config: CheckUpdatesEnabled() = false, want true")
	}
	if cfg.Update.LastCheckAt != "" {
		t.Errorf("LastCheckAt = %q, want empty", cfg.Update.LastCheckAt)
	}
}

func TestUpdateConfigRoundTrip(t *testing.T) {
	withTempRoot(t)
	cfg := Default()
	off := false
	cfg.Update.CheckUpdates = &off
	cfg.Update.IgnoredVersion = "0.3.0"
	cfg.Update.LastCheckAt = "2026-01-02T15:04:05Z"
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := Load()
	if got.Update.CheckUpdatesEnabled() {
		t.Error("CheckUpdatesEnabled() = true, want false after save")
	}
	if got.Update.IgnoredVersion != "0.3.0" {
		t.Errorf("IgnoredVersion = %q, want 0.3.0", got.Update.IgnoredVersion)
	}
	if got.Update.LastCheckAt != "2026-01-02T15:04:05Z" {
		t.Errorf("LastCheckAt = %q, want saved value", got.Update.LastCheckAt)
	}
}

func TestExplicitFalseStaysFalseAfterNormalize(t *testing.T) {
	withTempRoot(t)
	if err := os.WriteFile(Path(), []byte(`{"update":{"checkUpdates":false}}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg := Load()
	if cfg.Update.CheckUpdatesEnabled() {
		t.Error("explicit false got normalized back to true")
	}
}

// ---- language ----

func TestLanguageDefaultsToAuto(t *testing.T) {
	withTempRoot(t)
	cfg := Load()
	if cfg.Language != LangAuto {
		t.Errorf("Language = %q, want auto", cfg.Language)
	}
}

func TestLanguageInvalidFallsBackToAuto(t *testing.T) {
	withTempRoot(t)
	if err := os.WriteFile(Path(), []byte(`{"language":"ja-JP"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if cfg := Load(); cfg.Language != LangAuto {
		t.Errorf("Language = %q, want auto", cfg.Language)
	}
}

func TestLanguageRoundTrip(t *testing.T) {
	withTempRoot(t)
	cfg := Default()
	cfg.Language = LangEn
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := Load(); got.Language != LangEn {
		t.Errorf("Language = %q, want en", got.Language)
	}
}

// TestSaveConcurrent 验证并发 Save 不丢更新：固定 ".tmp" 名在并发写时
// 互相覆盖导致一个 Rename 失败（配置更新静默丢失），唯一 tmp 名后
// 两次并发写都成功且最终文件是合法 JSON。
func TestSaveConcurrent(t *testing.T) {
	withTempRoot(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c := Default()
			c.Trash.RetentionDays = n + 1
			if err := Save(c); err != nil {
				t.Errorf("并发 Save 失败: %v", err)
			}
		}(i)
	}
	wg.Wait()
	got := Load()
	if got.Trash.RetentionDays < 1 {
		t.Errorf("配置损坏: %+v", got)
	}
}
