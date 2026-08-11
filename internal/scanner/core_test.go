package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/rules"
)

// ---- pure helpers ----

func TestSumMaximal(t *testing.T) {
	sizes := map[string]int64{"/a": 10, "/a/b": 6, "/a/b/c": 2, "/x": 7}
	paths := []string{"/a", "/a/b", "/a/b/c", "/x"}
	// "/a/b" and "/a/b/c" are children of "/a"; only maximal paths count.
	if got := sumMaximal(paths, sizes); got != 17 {
		t.Errorf("sumMaximal = %d, want 17", got)
	}
}

func TestFootprintOf(t *testing.T) {
	// A binary inside a measured dir is covered by it; a standalone binary is
	// added to the footprint.
	tb := &toolBuilder{measure: []string{"/opt/tool"}, binaries: []Binary{
		{Real: "/opt/tool/bin/x", Size: 100}, // inside → not added
		{Real: "/usr/bin/free", Size: 50},    // outside → added
	}}
	sizes := map[string]int64{"/opt/tool": 1000}
	if got := footprintOf(tb, sizes); got != 1050 {
		t.Errorf("footprintOf = %d, want 1050", got)
	}
}

func TestBackupOwner(t *testing.T) {
	tools := map[string]*toolBuilder{
		"kimi": {name: "kimi"},
		"gh":   {name: "gh", aliases: map[string]bool{"hub": true}},
	}
	cases := []struct {
		base string
		want string // "" means nil owner
	}{
		{"kimi.bak", "kimi"},
		{"kimi.1784.old", "kimi"},
		{"hub.old", "gh"},
		{"nope.bak", ""},
	}
	for _, c := range cases {
		o := backupOwner(c.base, tools)
		if c.want == "" {
			if o != nil {
				t.Errorf("backupOwner(%q) = %q, want nil", c.base, o.name)
			}
			continue
		}
		if o == nil || o.name != c.want {
			t.Errorf("backupOwner(%q) = %+v, want %q", c.base, o, c.want)
		}
	}
}

func TestMatchFilter(t *testing.T) {
	tb := &toolBuilder{name: "cli-analyzer", aliases: map[string]bool{"ca": true}}
	if !matchFilter(tb, nil) {
		t.Error("nil filter should match all")
	}
	if !matchFilter(tb, []string{"ca"}) {
		t.Error("alias should match")
	}
	if !matchFilter(tb, []string{"analyzer"}) {
		t.Error("name substring should match")
	}
	if matchFilter(tb, []string{"zzz"}) {
		t.Error("non-matching filter should fail")
	}
}

func TestToolchainVersion(t *testing.T) {
	v, ok := toolchainVersion("/home/u/.rustup/toolchains/stable-x86_64/bin/cargo")
	if !ok || v != "stable-x86_64" {
		t.Errorf("toolchainVersion = %q, %v; want stable-x86_64", v, ok)
	}
	if _, ok := toolchainVersion("/usr/bin/go"); ok {
		t.Error("toolchainVersion should fail without a toolchains segment")
	}
}

// ---- old-version / toolchain derivation (uses a temp install layout) ----

func TestDeriveOldVersionsVersioned(t *testing.T) {
	versions := filepath.Join(t.TempDir(), "versions")
	for _, v := range []string{"2.1.0", "2.0.0"} {
		if err := os.MkdirAll(filepath.Join(versions, v), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(filepath.Join(versions, ".hidden"), 0o755) // dotfiles are skipped
	b := &toolBuilder{name: "claude", installer: InstVersioned, installRoot: versions, currentVer: "2.1.0"}
	b.deriveOldVersions("claude")

	if len(b.cleanables) != 1 {
		t.Fatalf("cleanables = %+v, want exactly 1", b.cleanables)
	}
	c := b.cleanables[0]
	if c.Tier != TierSafe || c.Kind != "old-version" {
		t.Errorf("cleanable = %+v, want SAFE old-version", c)
	}
	if !strings.Contains(c.Path, "2.0.0") {
		t.Errorf("path = %q, want it to point at 2.0.0", c.Path)
	}
	if c.CurrentPath != filepath.Join(versions, "2.1.0") {
		t.Errorf("CurrentPath = %q, want %q", c.CurrentPath, filepath.Join(versions, "2.1.0"))
	}
}

func TestDeriveOldVersionsPyenv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // isolate pyenv root
	versions := filepath.Join(home, ".pyenv", "versions")
	for _, v := range []string{"3.12.0", "3.11.5"} {
		os.MkdirAll(filepath.Join(versions, v), 0o755)
	}
	b := &toolBuilder{name: "pyenv", installer: InstPyenv, currentVer: "3.12.0"}
	b.deriveOldVersions("pyenv")

	if len(b.cleanables) != 1 {
		t.Fatalf("cleanables = %+v, want exactly 1", b.cleanables)
	}
	// Toolchains are load-bearing (shebangs reference the interpreter), so they
	// must be USER (display-only), never SAFE.
	if c := b.cleanables[0]; c.Tier != TierUser || c.Kind != "toolchain" {
		t.Errorf("cleanable = %+v, want USER toolchain", c)
	}
}

func TestDeriveOldVersionsRustup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // isolate rustup root
	toolchains := filepath.Join(home, ".rustup", "toolchains")
	for _, v := range []string{"stable", "nightly"} {
		os.MkdirAll(filepath.Join(toolchains, v, "bin"), 0o755)
	}
	b := &toolBuilder{name: "rustup", installer: InstRustup, binaries: []Binary{
		{Real: filepath.Join(toolchains, "stable", "bin", "cargo")},
	}}
	b.deriveOldVersions("rustup")

	if b.currentVer != "stable" {
		t.Fatalf("currentVer = %q, want stable (inferred from real path)", b.currentVer)
	}
	if len(b.cleanables) != 1 {
		t.Fatalf("cleanables = %+v, want exactly 1", b.cleanables)
	}
	if c := b.cleanables[0]; c.Tier != TierUser || c.Kind != "toolchain" {
		t.Errorf("cleanable = %+v, want USER toolchain", c)
	}
}

// ---- finalize / totals ----

func TestFinalize(t *testing.T) {
	tools := map[string]*toolBuilder{
		"npm": {
			name:      "npm",
			footprint: 1000,
			binaries:  []Binary{{Name: "npm", Size: 5}},
			cleanables: []Cleanable{
				{Tool: "npm", Path: "/cache/a", Bytes: 400, Tier: TierSafe},
				{Tool: "npm", Path: "/cache/b", Bytes: 200, Tier: TierUser},
			},
		},
		"empty": {name: "empty"}, // no footprint/bins/cleanables → pruned
	}
	res := finalize(tools, []string{"npm", "empty"}, Options{})

	if len(res.Tools) != 1 || res.Tools[0].Name != "npm" {
		t.Fatalf("tools = %+v", res.Tools)
	}
	if res.Totals.Footprint != 1000 || res.Totals.Cleanable != 400 || res.Totals.User != 600 {
		t.Errorf("totals = %+v", res.Totals)
	}
	if res.Tools[0].Cleanable != 400 {
		t.Errorf("tool cleanable = %d, want 400", res.Tools[0].Cleanable)
	}
}

// ---- cache round-trip (isolated via XDG_CACHE_HOME) ----

func TestCacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	res := &ScanResult{ScannedAt: Now(), Tools: []Tool{{Name: "npm"}}}

	if err := SaveCache(res); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}
	loaded, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if len(loaded.Tools) != 1 || loaded.Tools[0].Name != "npm" {
		t.Errorf("loaded = %+v", loaded.Tools)
	}
	if _, ok := CacheInfo(); !ok {
		t.Error("CacheInfo should report the cache exists")
	}

	if err := ClearCache(); err != nil {
		t.Fatalf("ClearCache: %v", err)
	}
	if _, ok := CacheInfo(); ok {
		t.Error("CacheInfo should report the cache is gone after clear")
	}
	if _, err := LoadCache(); err == nil {
		t.Error("LoadCache should fail after clear")
	}
}

// ---- attribute: cache-kind data dirs become SAFE cleanables ----

func TestAttributeCacheKindCleanable(t *testing.T) {
	root := t.TempDir()
	tb := &toolBuilder{name: "zzz-attr-tool", installer: InstOther,
		dataDirs: []DataDir{{Path: root, Tier: TierSafe, Kind: "cache"}}}
	tb.addMeasure(root)
	tools := map[string]*toolBuilder{"zzz-attr-tool": tb}

	attribute(tools, []string{"zzz-attr-tool"}, rules.Load(), Options{}, &disk.Sizer{})

	found := false
	for _, c := range tb.cleanables {
		if c.Kind == "cache" && c.Tier == TierSafe && c.Path == root {
			found = true
		}
	}
	if !found {
		t.Errorf("cleanables = %+v, want a SAFE cache item", tb.cleanables)
	}
}
