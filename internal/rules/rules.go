// Package rules defines the curated two-tier attribution and cleanable rules
// for known CLI tools, plus a generic resolver for anything not in the table.
package rules

import (
	"path/filepath"
	"strings"

	"cli-analyzer/internal/platform"
)

// Tier strings mirror scanner.Tier but are defined here to avoid an import
// cycle (scanner imports rules). Values must stay in sync with scanner.Tier.
const (
	TierSafe = "safe"
	TierUser = "user"
)

// DataDirRule describes one directory to attribute to a tool.
type DataDirRule struct {
	Root platform.RootKind // base root; ignored when Path is set
	Sub  string            // dir name under Root
	Path string            // literal absolute path (takes precedence)
	Tier string
	Kind string // config|data|cache|install|state
}

// CleanRule describes one cleanable item.
type CleanRule struct {
	Root platform.RootKind
	Sub  string // dir name under Root, e.g. "claude"
	Path string // literal absolute path (takes precedence); may end in *
	Tier string
	Kind string // old-version|cache|backup|toolchain|download
	Keep string
	Desc string
}

// Rule is the curated entry for one tool.
type Rule struct {
	Name        string
	Aliases     []string
	Installer   string
	Homepage    string // official website, shown in the GUI detail panel
	Description string // one-line summary in Chinese
	DataDirs    []DataDirRule
	Cleanables  []CleanRule
}

// Resolve returns the absolute path for a data-dir rule ("" if not applicable).
func (r DataDirRule) Resolve() string {
	if r.Path != "" {
		return r.Path
	}
	base := platform.Root(r.Root)
	if base == "" {
		return ""
	}
	if r.Sub == "" {
		return base
	}
	return filepath.Join(base, r.Sub)
}

// Resolve returns the absolute path for a cleanable rule ("" if not
// applicable, including when the base root does not exist on this OS).
func (r CleanRule) Resolve() string {
	if r.Path != "" {
		return r.Path
	}
	base := platform.Root(r.Root)
	if base == "" {
		return ""
	}
	if r.Sub == "" {
		return base
	}
	return filepath.Join(base, r.Sub)
}

// Table is the loaded rule set.
type Table struct {
	byName map[string]*Rule
	order  []string
}

// Load builds the table from the curated entries.
func Load() *Table {
	t := &Table{byName: map[string]*Rule{}}
	for i := range curated {
		r := &curated[i]
		t.byName[r.Name] = r
		t.order = append(t.order, r.Name)
	}
	return t
}

// Lookup returns the curated rule for a tool name (exact match only).
func (t *Table) Lookup(name string) *Rule { return t.byName[name] }

// Meta returns display metadata (homepage + description) for a tool.
func (t *Table) Meta(name string) Meta {
	if m, ok := metaByName[name]; ok {
		return m
	}
	if r := t.byName[name]; r != nil && (r.Homepage != "" || r.Description != "") {
		return Meta{Homepage: r.Homepage, Description: r.Description}
	}
	return Meta{}
}

// Names returns the curated tool names in definition order.
func (t *Table) Names() []string { return append([]string(nil), t.order...) }

// ResolveCleanable resolves a cleanable rule to concrete paths. A trailing "*"
// in either the literal Path or the Sub name expands to all matching entries in
// the parent directory. Returns empty when nothing matches.
func ResolveCleanable(r CleanRule) []string {
	pattern := ""
	if r.Path != "" {
		pattern = r.Path
	} else if r.Root != "" && strings.Contains(r.Sub, "*") {
		if base := platform.Root(r.Root); base != "" {
			pattern = filepath.Join(base, r.Sub)
		}
	}
	if pattern != "" && strings.HasSuffix(pattern, "*") {
		dir := filepath.Dir(strings.TrimSuffix(pattern, "*"))
		base := filepath.Base(strings.TrimSuffix(pattern, "*"))
		entries, err := platform.ReadDir(dir)
		if err != nil {
			return nil
		}
		var out []string
		for _, e := range entries {
			// 通配语义是前缀匹配：`codex-runtime-install-*` 只匹配以该
			// 片段开头的目录；Contains 会把任意位置包含该片段的目录
			// （如 my-codex-runtime-install-backup）误判为清理项。
			if strings.HasPrefix(e.Name(), base) {
				out = append(out, filepath.Join(dir, e.Name()))
			}
		}
		return out
	}
	if p := r.Resolve(); p != "" {
		return []string{p}
	}
	return nil
}
