package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/disk"
)

// TestFindUnattributedFilter verifies the non-CLI exclusion system inside
// findUnattributed: self dirs, structural GUI signals, vendor table.
func TestFindUnattributedFilter(t *testing.T) {
	base := t.TempDir()
	cacheRoot := filepath.Join(base, "cache")
	dataRoot := filepath.Join(base, "data")
	cfgRoot := filepath.Join(base, "config")
	for _, r := range []string{cacheRoot, dataRoot, cfgRoot} {
		if err := os.MkdirAll(r, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("XDG_DATA_HOME", dataRoot)
	t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	// macOS root 走 HOME，隔离避免读到真实目录
	t.Setenv("HOME", base)

	mkdir := func(p string) {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "payload.bin"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 真孤儿：应保留
	mkdir(filepath.Join(dataRoot, "dead-cli-tool"))
	// 被工具认领：应排除
	mkdir(filepath.Join(cacheRoot, "claimed-tool"))
	// 本应用自身：应排除
	mkdir(filepath.Join(dataRoot, "cli-analyzer"))
	// 结构性 GUI 信号：应排除
	mkdir(filepath.Join(cfgRoot, "com.apple.Safari"))
	mkdir(filepath.Join(dataRoot, "Some.App_12345678"))
	mkdir(filepath.Join(cacheRoot, "packages"))
	// 厂商排除表：应排除
	mkdir(filepath.Join(dataRoot, "Netsarang Computer"))
	// macOS Application Support（GUI 主导，非孤儿来源）：应排除——即使 HOME
	// 下有该目录也不应被遍历
	mkdir(filepath.Join(base, "Library", "Application Support", "App Store"))
	mkdir(filepath.Join(base, "Library", "Application Support", "Safari"))

	tools := map[string]*toolBuilder{"claimed-tool": {name: "claimed-tool", aliases: map[string]bool{}}}
	order := []string{"claimed-tool"}

	got := findUnattributed(tools, order, &disk.Sizer{})
	if len(got) != 1 {
		t.Fatalf("expected 1 orphan dir, got %d: %+v", len(got), got)
	}
	if got[0].Path != filepath.Join(dataRoot, "dead-cli-tool") {
		t.Errorf("orphan path = %q, want dead-cli-tool", got[0].Path)
	}
	if got[0].Bytes <= 0 {
		t.Errorf("orphan bytes should be sized, got %d", got[0].Bytes)
	}
	if got[0].Root != "xdg-data" {
		t.Errorf("orphan root = %q, want xdg-data", got[0].Root)
	}
	if got[0].Tier != TierUser {
		t.Errorf("orphan tier = %q, want user", got[0].Tier)
	}
}
