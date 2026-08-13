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
		// 已知 GUI 产品安装目录（NetSarang 等）整目录跳过：该目录下没有用户
		// CLI 工具，无论 exe 叫什么（Xshell/Xftp 的 xftpcl.exe 也是其组件）。
		if isVendorInstallDir(dir) {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		// 两遍扫描：第一遍判定本目录是否存在 GUI 主程序（如 Xshell.exe），
		// 并收集控制台候选；第二遍过滤。
		type cand struct {
			entry os.DirEntry
			full  string
		}
		var cands []cand
		dirHasGUI := false
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
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink != 0 {
				out = append(out, resolveSymlinkExec(full, name)...)
				continue
			}
			if !platform.IsExecutable(info) {
				continue
			}
			console := true
			if strings.HasSuffix(low, ".exe") {
				console = platform.IsConsoleExe(full)
			}
			if !console {
				// GUI 主程序：本身不是工具，但标记本目录为 GUI 应用安装目录
				dirHasGUI = true
				continue
			}
			cands = append(cands, cand{e, full})
		}
		for _, c := range cands {
			// GUI 应用安装目录（Xshell、Xftp…）里的内部控制台助手
			// （installanchorservice.exe、RealCmdModule.exe…）不是工具。
			if dirHasGUI && isVendorHelper(c.full) {
				continue
			}
			out = append(out, execEntry{Path: c.full, Name: c.entry.Name()})
		}
	}
	return out
}

// isVendorHelper 报告文件名是否命中 GUI 应用安装目录内的内部助手命名模式。
// 仅在该目录同时存在 GUI 主程序时才会被调用，故可安全覆盖 install/module/
// agent 等通用词（命令行工具目录不会触发）。
func isVendorHelper(path string) bool {
	stem := strings.ToLower(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	for _, p := range []string{"install", "setup", "unins", "update", "updater",
		"helper", "repair", "module", "anchor", "service", "svc", "agent"} {
		if strings.Contains(stem, p) {
			return true
		}
	}
	return false
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

// vendorGUIAppDirs 是已知 GUI 产品安装目录的厂商标识（路径任意层级命中即整体跳过）。
// 与 isVendorHelper 互补：helper 模式管"通用命名助手"，这里管"整个厂商目录"。
var vendorGUIAppDirs = []string{"netsarang"}

// isVendorInstallDir 报告目录路径（小写）是否命中已知 GUI 产品厂商。
func isVendorInstallDir(dir string) bool {
	low := strings.ToLower(dir)
	for _, v := range vendorGUIAppDirs {
		if strings.Contains(low, v) {
			return true
		}
	}
	return false
}
