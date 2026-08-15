package rules

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/platform"
)

func TestLoadCurated(t *testing.T) {
	tbl := Load()
	for _, name := range []string{"claude", "nodejs", "gh", "uv", "pyenv", "go", "rustup", "p10k"} {
		if tbl.Lookup(name) == nil {
			t.Errorf("curated rule %q missing", name)
		}
	}
}

func TestGenericDataDirs(t *testing.T) {
	dirs := GenericDataDirs("faketool")
	found := map[platform.RootKind]bool{}
	for _, d := range dirs {
		found[d.Root] = true
	}
	if !found[platform.XDGCache] || !found[platform.XDGData] || !found[platform.Home] {
		t.Errorf("generic dirs missing roots: %v", found)
	}
}

func TestResolveCleanableGlob(t *testing.T) {
	// p10k 缓存规则锚定 XDG cache 根：unix 语义，Windows 上该根不存在
	// （数据根为 %APPDATA%/%LOCALAPPDATA%），glob 无可解析目标。
	if runtime.GOOS == "windows" {
		t.Skip("XDG cache root does not apply on Windows")
	}
	td := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", td)
	// create p10k-style entries
	mkdir(t, filepath.Join(td, "p10k-wei"))
	mkdir(t, filepath.Join(td, "p10k-dump-wei.zsh"))
	mkdir(t, filepath.Join(td, "other"))

	var r CleanRule
	for _, cur := range curated {
		if cur.Name == "p10k" && len(cur.Cleanables) > 0 {
			r = cur.Cleanables[0]
		}
	}
	if r.Sub == "" {
		t.Fatal("p10k cleanable rule not found")
	}
	got := ResolveCleanable(r)
	if len(got) != 2 {
		t.Fatalf("expected 2 glob matches, got %v", got)
	}
}

// TestResolveCleanableGlobPrefixOnly 验证通配规则是前缀匹配而非任意包含：
// 目录名中间包含 base 片段但不以其开头时，不应被归为该工具的清理项
// （防止把用户自建目录误判为缓存/暂存目录）。
func TestResolveCleanableGlobPrefixOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG cache root does not apply on Windows")
	}
	td := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", td)
	mkdir(t, filepath.Join(td, "p10k-wei"))
	mkdir(t, filepath.Join(td, "my-p10k-cache")) // 包含 "p10k-" 但不以其开头

	var r CleanRule
	for _, cur := range curated {
		if cur.Name == "p10k" && len(cur.Cleanables) > 0 {
			r = cur.Cleanables[0]
		}
	}
	if r.Sub == "" {
		t.Fatal("p10k cleanable rule not found")
	}
	got := ResolveCleanable(r)
	if len(got) != 1 {
		t.Fatalf("prefix 规则应只匹配 1 项（p10k-* 前缀）, got %v", got)
	}
	if filepath.Base(got[0]) != "p10k-wei" {
		t.Errorf("匹配项应为 p10k-wei, got %v", got)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
