// Package scanner discovers installed CLI tools, attributes their disk usage,
// and classifies cleanable space. It is the pure core shared by the CLI and
// the Wails GUI (which must never depend on this package's internals either —
// only its JSON contract).
package scanner

import (
	"strings"
	"time"
)

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
	// InstLocalBin 是官方脚本安装（astral uv、poetry、rye 等）：二进制直接
	// 放入用户 bin 目录（~/.local/bin 或 XDG_BIN_HOME），无包管理器，
	// 卸载 = 删除该二进制；数据目录由残留检测接管。
	InstLocalBin Installer = "local-bin"
	// InstPip 是 pip 家族（pip/pip3/pip3.x 命令）的安装来源：二进制通常由
	// brew python / python.org 安装器提供，但用户心智中 pip 是独立工具，
	// 家族归并后 pip 行持有二进制与版本。
	InstPip   Installer = "pip"
	InstOther Installer = "other"
)

// ProbeSafeInstaller 报告安装来源是否属于已知 CLI 生态（可安全执行其二进制
// 做版本探测）。InstOther（未知来源）不执行——GUI 应用内部件（SASE/Parallels/
// Warp 等）常以 InstOther 出现，执行其二进制可能触发系统权限（TCC）提示。
func ProbeSafeInstaller(i Installer) bool {
	switch i {
	case InstBrew, InstNpm, InstPipx, InstCargo, InstGo, InstPyenv, InstVersioned, InstRustup, InstNodejs, InstLocalBin, InstPip:
		return true
	}
	return false
}

// ProbeSafeBinary 报告对 (installer, real, name) 二进制做版本探测是否安全：
// 已知 CLI 安装器来源总是安全；InstOther（未知来源）仅在二进制不在 GUI
// 应用包内（.app/Contents 内部件执行可能触发 macOS TCC 权限提示）或命令名
// 是公认 CLI（OrbStack 把 docker/kubectl 放在 .app 包内但确实是命令行）
// 时允许。调用方（GUI probeAll / CLI FillVersions）用二进制路径判断，
// 而非仅安装来源。
func ProbeSafeBinary(i Installer, real, name string) bool {
	if ProbeSafeInstaller(i) {
		return true
	}
	if i != InstOther {
		return false
	}
	if !underAppBundle(real) {
		return true
	}
	switch strings.ToLower(name) {
	case "docker", "docker-compose", "kubectl":
		return true
	}
	return false
}

// underAppBundle 报告路径是否位于 GUI 应用包内（.app 目录下）。
func underAppBundle(p string) bool {
	low := strings.ToLower(p)
	return strings.Contains(low, ".app/") || strings.HasSuffix(low, ".app")
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
	// NoCache skips writing the on-disk cache.
	NoCache bool
	// ToolFilter, when non-empty, restricts results to tools whose name or
	// alias contains any of these substrings.
	ToolFilter []string
}

// Now returns the current local time as RFC3339 for ScannedAt.
func Now() string { return time.Now().Format(time.RFC3339) }
