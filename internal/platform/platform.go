// Package platform resolves per-OS data-root directories and PATH semantics.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RootKind identifies a class of data-root directory.
type RootKind string

const (
	XDGCache  RootKind = "xdg-cache"
	XDGData   RootKind = "xdg-data"
	XDGConfig RootKind = "xdg-config"
	XDGState  RootKind = "xdg-state"
	Home      RootKind = "home"
	// macOS only:
	MacAppSupport RootKind = "macos-application-support"
	MacCaches     RootKind = "macos-caches"
	MacPrefs      RootKind = "macos-preferences"
	// Windows only:
	AppData      RootKind = "appdata"
	LocalAppData RootKind = "localappdata"
)

// AllKinds is the enumeration order used by Roots().
var AllKinds = []RootKind{
	XDGCache, XDGData, XDGConfig, XDGState, Home,
	MacAppSupport, MacCaches, MacPrefs, AppData, LocalAppData,
}

// Root returns the base directory for kind, or "" when the kind does not
// apply to the current platform.
func Root(kind RootKind) string { return rootFor(kind) }

// Roots returns every applicable root on the current platform.
func Roots() map[RootKind]string {
	out := map[RootKind]string{}
	for _, k := range AllKinds {
		if v := Root(k); v != "" {
			out[k] = v
		}
	}
	return out
}

// HomeDir returns the user's home directory ("" if unknown).
func HomeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

// CacheRoot returns the per-user cache directory for this app.
func CacheRoot() string {
	for _, k := range []RootKind{XDGCache, MacCaches, LocalAppData} {
		if r := Root(k); r != "" {
			return filepath.Join(r, "cli-analyzer")
		}
	}
	if h := HomeDir(); h != "" {
		return filepath.Join(h, ".cache", "cli-analyzer")
	}
	return ".cli-analyzer"
}

// DataRoot returns the per-user application data directory for this app.
// 内置回收站、趋势历史等需要持久化的本地状态存放在此，与可随时清空的缓存目录语义分离
func DataRoot() string {
	for _, k := range []RootKind{MacAppSupport, XDGData, LocalAppData} {
		if r := Root(k); r != "" {
			return filepath.Join(r, "cli-analyzer")
		}
	}
	if h := HomeDir(); h != "" {
		return filepath.Join(h, ".local", "share", "cli-analyzer")
	}
	return ".cli-analyzer"
}

// PathDirs returns deduplicated, absolute PATH directories. When skipSystem is
// true, /usr/bin, /bin, /usr/sbin and /sbin are omitted (system-installed
// tools, rarely worth attributing).
func PathDirs(skipSystem bool) []string {
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	var out []string
	seen := map[string]bool{}
	for _, d := range strings.Split(os.Getenv("PATH"), sep) {
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if skipSystem && isSystemDir(abs) {
			continue
		}
		out = append(out, abs)
	}
	// 补齐 GUI 启动（Finder/Start Menu）时缺失的 shell-only 二进制目录，
	// 使 GUI 扫描与终端扫描结果一致（见 path_augment_unix.go）。
	return augmentUserDirs(seen, out)
}

func isSystemDir(d string) bool {
	switch d {
	case "/usr/bin", "/bin", "/usr/sbin", "/sbin":
		return true
	}
	return false
}

// OS returns a human-friendly platform label such as "macOS (arm64)" or
// "Linux (x86_64)". The raw Go GOOS ("darwin", "linux"…) is cryptic to end
// users, so the friendly name is what surfaces in the status bar.
func OS() string {
	label := map[string]string{
		"darwin":  "macOS",
		"linux":   "Linux",
		"windows": "Windows",
	}[runtime.GOOS]
	if label == "" {
		label = runtime.GOOS
	}
	return fmt.Sprintf("%s (%s)", label, runtime.GOARCH)
}

// ReadDir lists the entries of dir, returning an empty slice on error.
func ReadDir(dir string) ([]os.DirEntry, error) {
	return os.ReadDir(dir)
}
