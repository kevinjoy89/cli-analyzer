package rules

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/platform"
)

func TestLoadCurated(t *testing.T) {
	tbl := Load()
	for _, name := range []string{"claude", "npm", "gh", "uv", "pyenv", "go", "rustup", "p10k"} {
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

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
