package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"cli-analyzer/internal/platform"
)

// execEntry is one executable discovered on PATH.
type execEntry struct {
	Path string // full path as found (may be a symlink)
	Name string // base name
}

// discoverExecs enumerates executables on PATH, resolving symlinks so that
// versioned installers and symlink-to-directory commands are found too.
// System dirs (/usr/bin etc.) are skipped.
func discoverExecs() []execEntry {
	var out []execEntry
	for _, dir := range platform.PathDirs(true) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Type().IsDir() {
				continue
			}
			name := e.Name()
			// *.bak / *.old leftovers are not commands; they surface later as
			// SAFE backup cleanables instead of phantom tool rows.
			low := strings.ToLower(name)
			if strings.HasSuffix(low, ".bak") || strings.HasSuffix(low, ".old") {
				continue
			}
			full := filepath.Join(dir, name)
			// 非 CLI 排除表：路径片段命中 GUI 厂商（NetSarang 等）且名称不在
			// 该厂商纯 CLI 例外（aws/gcloud/az…）中 → 整体跳过。
			if platform.ExcludedByVendor(dir, name) {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				out = append(out, resolveSymlinkExec(full, name)...)
				continue
			}
			if !platform.IsExecutable(info) || !platform.IsConsoleExe(full) {
				continue
			}
			out = append(out, execEntry{Path: full, Name: name})
		}
	}
	return out
}

// resolveSymlinkExec resolves a symlinked PATH entry: if it points at a file it
// is a normal command; if at a directory, look for <dir>/bin/<name> or
// <dir>/<name> (e.g. sdkman current links). Returns zero or one entries.
func resolveSymlinkExec(full, name string) []execEntry {
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		return nil
	}
	st, err := os.Stat(real)
	if err != nil {
		return nil
	}
	if st.IsDir() {
		for _, cand := range []string{filepath.Join(real, "bin", name), filepath.Join(real, name)} {
			ci, err := os.Stat(cand)
			if err == nil && ci.Mode().IsRegular() && platform.IsExecutable(ci) && platform.IsConsoleExe(full) {
				return []execEntry{{Path: full, Name: name}}
			}
		}
		return nil
	}
	if platform.IsExecutable(st) && platform.IsConsoleExe(full) {
		return []execEntry{{Path: full, Name: name}}
	}
	return nil
}

// vendorGUIAppDirs 已由 platform.MatchVendorExclusion / ExcludedByVendor 取代
// （internal/platform/vendorexclusion.go）：路径片段精确匹配 + 纯 CLI 例外。
