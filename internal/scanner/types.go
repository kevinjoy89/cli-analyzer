// Package scanner discovers installed CLI tools, attributes their disk usage,
// and classifies cleanable space. It is the pure core shared by the CLI and
// the Wails GUI (which must never depend on this package's internals either —
// only its JSON contract).
package scanner

import "time"

// Tier classifies an item by what kind of directory it is. Since the two-tier
// gate was removed it is an informational label only — the cleaner never
// blocks on it; the user decides what to dispose of.
type Tier string

const (
	// TierSafe labels items that are typically safe to remove (caches, old
	// versions, package-manager caches). Informational only.
	TierSafe Tier = "safe"
	// TierUser labels config/data/history/venv-style directories. Removing them
	// is riskier, so they carry this label — but nothing blocks the user from
	// choosing to dispose of them.
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
	// InstNodejs 是 Node.js 运行时家族合并工具的安装来源（Windows 官方安装器、
	// nvm-windows / Volta / scoop 等，或 unix 上 fnm 之类的 node_modules 布局）。
	// node/npm/npx/corepack/node-gyp 归并为一条 "nodejs"，避免 Windows 上
	// 同一目录被扫出多个互不相干的小工具。
	InstNodejs Installer = "nodejs"
	InstOther  Installer = "other"
)

// ProbeSafeInstaller 报告安装来源是否属于已知 CLI 生态（可安全执行其二进制
// 做版本探测）。InstOther（未知来源）不执行——GUI 应用内部件（SASE/Parallels/
// Warp 等）常以 InstOther 出现，执行其二进制可能触发系统权限（TCC）提示。
func ProbeSafeInstaller(i Installer) bool {
	switch i {
	case InstBrew, InstNpm, InstPipx, InstCargo, InstGo, InstPyenv, InstVersioned, InstRustup, InstNodejs:
		return true
	}
	return false
}

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
// Kind is the child's precise type: usually inherited from the parent, but
// distinguishable children get their own (e.g. ~/.npm/_logs is "logs", not
// "cache"). The cleaner stores this kind on trash items so the trash list
// classifies each moved path accurately.
type SubEntry struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	ID    string `json:"id"`
	Kind  string `json:"kind"`
}

// Cleanable is one user-actionable attributed directory (every attributed
// data dir except the tool's own install root).
type Cleanable struct {
	ID    string `json:"id"` // stable: tool|kind|path
	Tool  string `json:"tool"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
	Tier  Tier   `json:"tier"` // informational label (safe|user), not a gate
	Kind  string `json:"kind"` // cache|config|data|old-version|backup|toolchain|download|state|logs
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
	Version     string      `json:"version"`          // current version, when known
	UpdatedAt   string      `json:"updatedAt"`        // RFC3339 mtime of newest install file
	Homepage    string      `json:"homepage"`         // official website (curated metadata)
	Description string      `json:"description"`      // one-line summary (curated metadata)
	Family      string      `json:"family,omitempty"` // 家族合并根名（如 "nodejs"）；空 = 普通单工具。前端据此把 aliases 展示为「包含工具」而非「别名」。
	Binaries    []Binary    `json:"binaries"`
	DataDirs    []DataDir   `json:"dataDirs"`
	Cleanables  []Cleanable `json:"cleanables"`
	// Footprint is the union size of all maximal attributed dirs.
	Footprint int64 `json:"footprintBytes"`
	// Cleanable is the sum of all actionable items (attributed dirs except the
	// install root).
	Cleanable int64 `json:"cleanableBytes"`
	// User is Footprint minus Cleanable (install roots, standalone binaries…).
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
