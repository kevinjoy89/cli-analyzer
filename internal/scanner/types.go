// Package scanner discovers installed CLI tools, attributes their disk usage,
// and classifies cleanable space. It is the pure core shared by the CLI and
// the Wails GUI (which must never depend on this package's internals either —
// only its JSON contract).
package scanner

import "time"

// Tier classifies an item for the two-tier safety model.
type Tier string

const (
	// TierSafe items (caches, old versions, package-manager caches) may be
	// deleted after per-item confirmation.
	TierSafe Tier = "safe"
	// TierUser items (config, data, history, venvs) are shown but never
	// auto-deleted; the cleaner hard-rejects them regardless of flags.
	TierUser Tier = "user"
)

// Installer identifies how a tool was installed, which drives its data-dir
// attribution and cleanable rules.
type Installer string

const (
	InstVersioned Installer = "versioned" // keeps versions/<v>; old = cleanable
	InstBrew      Installer = "brew"
	InstNpm       Installer = "npm"
	InstPipx      Installer = "pipx"
	InstPyenv     Installer = "pyenv"
	InstGo        Installer = "go"
	InstCargo     Installer = "cargo"
	InstRustup    Installer = "rustup"
	InstOther     Installer = "other"
)

// Binary is one executable on PATH attributed to a tool.
type Binary struct {
	Name string `json:"name"` // base name as seen on PATH
	Path string `json:"path"` // as found on PATH (may be a symlink)
	Real string `json:"real"` // resolved real path
	Size int64  `json:"size"` // file size, not a directory
}

// DataDir is one attributed data/configuration/cache directory.
type DataDir struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Tier  Tier   `json:"tier"`
	Kind  string `json:"kind"`           // config|data|cache|install|state
	Root  string `json:"root,omitempty"` // 所在数据根类型（孤儿数据专用）
}

// SubEntry is one direct child (file or dir) of a cleanable path, used to
// render one level of breakdown in the UI (e.g. ~/.npm -> _cacache, _npx).
// ID is stable across scans: "<parent cleanable ID>::<sub path>".
type SubEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	ID    string `json:"id"`
}

// Cleanable is one deletable item.
type Cleanable struct {
	ID    string `json:"id"` // stable: tool|kind|path
	Tool  string `json:"tool"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Tier  Tier   `json:"tier"`
	Kind  string `json:"kind"` // old-version|cache|backup|toolchain|download
	Keep  string `json:"keep"` // what is never deleted (e.g. "current symlink target")
	Desc  string `json:"desc"` // human sentence for the confirm prompt
	// Sub is the size breakdown of Path's direct children (SAFE items only),
	// sorted by Bytes descending. Empty for files and non-readable dirs.
	Sub []SubEntry `json:"sub"`
	// CurrentPath, when set, is the tool's current version path — the cleaner
	// refuses to delete an old-version item equal to it. Not serialized.
	CurrentPath string `json:"-"`
}

// Tool aggregates everything attributed to one CLI identity.
type Tool struct {
	Name        string      `json:"name"`
	Aliases     []string    `json:"aliases"`
	Installer   string      `json:"installer"`
	Version     string      `json:"version"`     // current version, when known
	UpdatedAt   string      `json:"updatedAt"`   // RFC3339 mtime of newest install file
	Homepage    string      `json:"homepage"`    // official website (curated metadata)
	Description string      `json:"description"` // one-line summary (curated metadata)
	Binaries    []Binary    `json:"binaries"`
	DataDirs    []DataDir   `json:"dataDirs"`
	Cleanables  []Cleanable `json:"cleanables"`
	// Footprint is the union size of all maximal attributed dirs.
	Footprint int64 `json:"footprintBytes"`
	// Cleanable is the sum of SAFE items.
	Cleanable int64 `json:"cleanableBytes"`
	// User is Footprint minus Cleanable.
	User int64 `json:"userBytes"`
}

// Totals aggregates across all tools.
type Totals struct {
	Footprint int64 `json:"footprintBytes"`
	Cleanable int64 `json:"cleanableBytes"`
	User      int64 `json:"userBytes"`
}

// ScanResult is the single JSON contract shared by CLI and GUI.
type ScanResult struct {
	ScannedAt    string              `json:"scannedAt"`
	ScanTimeMS   int64               `json:"scanTimeMs"`
	Platform     string              `json:"platform"`
	GoVersion    string              `json:"goVersion"`
	Tools        []Tool              `json:"tools"`
	Totals       Totals              `json:"totals"`
	Roots        map[string][]string `json:"roots"` // RootKind -> dirs walked
	Unattributed []DataDir           `json:"unattributed,omitempty"`
	Errors       int                 `json:"walkErrors"`
}

// CleanReport describes the outcome of a cleanup run.
type CleanReport struct {
	DryRun  bool     `json:"dryRun"`
	Deleted []string `json:"deleted"`
	Freed   int64    `json:"freedBytes"`
	Skipped []string `json:"skipped"`
	Errors  []string `json:"errors"`
}

// Options controls a scan.
type Options struct {
	// Full measures unmatched entries under each data root and reports them
	// in Unattributed.
	Full bool
	// Refresh ignores the on-disk cache and rescans.
	Refresh bool
	// NoCache skips writing the on-disk cache.
	NoCache bool
	// ToolFilter, when non-empty, restricts results to tools whose name or
	// alias contains any of these substrings.
	ToolFilter []string
}

// Now returns the current local time as RFC3339 for ScannedAt.
func Now() string { return time.Now().Format(time.RFC3339) }
