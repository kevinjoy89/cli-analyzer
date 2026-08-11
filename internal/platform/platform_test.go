package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootXDGEnv(t *testing.T) {
	// XDG env vars must take precedence over defaults.
	td := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(td, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(td, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(td, "config"))
	if got := Root(XDGCache); got != filepath.Join(td, "cache") {
		t.Errorf("XDGCache = %q", got)
	}
	if got := Root(XDGData); got != filepath.Join(td, "data") {
		t.Errorf("XDGData = %q", got)
	}
	if got := Root(XDGConfig); got != filepath.Join(td, "config") {
		t.Errorf("XDGConfig = %q", got)
	}
}

func TestHome(t *testing.T) {
	if HomeDir() == "" {
		t.Error("HomeDir() should not be empty in a test environment")
	}
}

func TestPathDirsDedup(t *testing.T) {
	orig := os.Getenv("PATH")
	defer os.Setenv("PATH", orig)
	t.Setenv("PATH", "/a:/a:/b:/usr/bin")
	dirs := PathDirs(true)
	// "/a" deduped, "/usr/bin" skipped, "/b" kept.
	if len(dirs) != 2 || dirs[0] != "/a" || dirs[1] != "/b" {
		t.Errorf("PathDirs = %v, want [/a /b]", dirs)
	}
	if !strings.Contains(strings.Join(PathDirs(false), " "), "/usr/bin") {
		t.Error("skipSystem=false should include /usr/bin")
	}
}

func TestRootsNoEmpty(t *testing.T) {
	roots := Roots()
	if _, ok := roots[Home]; !ok {
		t.Error("Roots should always include Home")
	}
}
