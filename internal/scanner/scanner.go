package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/rules"
	"cli-analyzer/internal/trash"
)

// Scan runs the full pipeline: discover PATH executables, classify them into
// tools, attribute data dirs + install roots, size everything in parallel and
// classify cleanables. Unless Options.NoCache is set, the result is cached.
func Scan(opts Options) (*ScanResult, error) {
	start := time.Now()
	// 顺带清除内置回收站的过期项（静默，失败不影响扫描）
	trash.Sweep()
	ruleTable := rules.Load()

	execs := discoverExecs()

	tools := map[string]*toolBuilder{}
	order := []string{}
	addBinary := func(id string, installer Installer, b Binary, currentVer, installRoot string) {
		tb := tools[id]
		if tb == nil {
			tb = &toolBuilder{name: id, aliases: map[string]bool{}}
			tools[id] = tb
			order = append(order, id)
		}
		if tb.installer == "" || (tb.installer == InstOther && installer != InstOther) ||
			installer == InstPyenv || installer == InstRustup {
			tb.installer = installer
		}
		if tb.installRoot == "" || (installer == InstPyenv && tb.installRoot != pyenvVersionsPath()) {
			tb.installRoot = installRoot
		}
		if tb.currentVer == "" {
			tb.currentVer = currentVer
		}
		if b.Name != id {
			tb.aliases[b.Name] = true
		}
		tb.binaries = append(tb.binaries, b)
	}

	for _, ex := range execs {
		real, err := filepath.EvalSymlinks(ex.Path)
		if err != nil {
			real = ex.Path
		}
		var size int64
		if st, err := os.Stat(real); err == nil && st.Mode().IsRegular() {
			size = st.Size()
		}
		c := classify(real, ex.Name)
		addBinary(c.ToolID, c.Installer, Binary{Name: ex.Name, Path: ex.Path, Real: real, Size: size}, c.CurrentVersion, c.InstallRoot)
	}

	// Seed curated tools that have no PATH binary (p10k, puppeteer…).
	for _, name := range ruleTable.Names() {
		if _, ok := tools[name]; ok {
			continue
		}
		r := ruleTable.Lookup(name)
		tools[name] = &toolBuilder{name: name, installer: Installer(r.Installer), aliases: map[string]bool{}}
		order = append(order, name)
	}

	// 测量时排除内置回收站，避免自我归因/自我清理
	sizer := &disk.Sizer{Skip: map[string]bool{trash.Root(): true}}
	attribute(tools, order, ruleTable, opts, sizer)

	res := finalize(tools, order, opts)
	// 孤儿数据始终计算（非 CLI 排除体系过滤后），GUI 与 CLI 共用。
	res.Unattributed = findUnattributed(tools, order, sizer)
	res.Errors = sizer.Errors

	if !opts.NoCache {
		// Cache the full (unfiltered) result: a filtered CLI scan like
		// `scan npm` must not clobber the snapshot the GUI starts from.
		cached := res
		if len(opts.ToolFilter) > 0 {
			cached = finalize(tools, order, Options{Full: opts.Full})
		}
		_ = SaveCache(cached)
	}
	res.ScanTimeMS = time.Since(start).Milliseconds()
	return res, nil
}

// findUnattributed walks every top-level dir under the data roots that no tool
// claims, reporting non-empty ones. Candidates pass the non-CLI exclusion
// system (self dirs, structural GUI signals, vendor table) — see the
// non-cli-exclusion capability. Always runs (sizes computed in parallel).
func findUnattributed(tools map[string]*toolBuilder, order []string, sizer *disk.Sizer) []DataDir {
	claimed := map[string]bool{}
	for _, id := range order {
		for n := range tools[id].aliasSet() {
			claimed[n] = true
		}
	}
	// cleanable 规则覆盖的路径也算认领：codex 的 ~/.cache/codex-runtimes 等
	// 已作为该工具的 cleanable 提供，不应再列为孤儿。
	for _, id := range order {
		for _, c := range tools[id].cleanables {
			claimTopLevel(claimed, c.Path)
		}
	}
	var paths []string
	var pathKind = map[string]string{}
	// 孤儿遍历仅限 CLI 工具主导的数据根（平台不适用时解析为 "" 自动跳过）：
	// macOS/Linux 用 XDG 目录；Windows 用 AppData/LocalAppData。
	// macOS Application Support/Caches/Preferences 是 GUI 应用主导（Safari、
	// App Store、Chrome…），只用于已认领工具的归因，不作为孤儿来源。
	for _, k := range []platform.RootKind{
		platform.XDGCache, platform.XDGData, platform.XDGConfig,
		platform.AppData, platform.LocalAppData,
	} {
		root := platform.Root(k)
		if root == "" {
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if claimed[e.Name()] {
				continue
			}
			p := filepath.Join(root, e.Name())
			// 非 CLI 排除体系（确定性规则）：本应用自身、结构性 GUI 信号
			// （macOS 容器 bundle-id / Windows UWP 包族及 Packages 容器）、
			// 非 CLI 厂商排除表（microsoft 等系统目录一并覆盖）。
			if isSelfDataDir(e.Name()) ||
				platform.IsContainerBundleDir(e.Name()) ||
				platform.IsUWPFamilyDir(e.Name()) ||
				strings.EqualFold(e.Name(), "packages") || // Windows UWP 容器目录
				platform.ExcludedByVendorData(p, e.Name()) {
				continue
			}
			paths = append(paths, p)
			pathKind[p] = string(k)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	sizes := sizer.WalkAll(paths)
	var out []DataDir
	for _, p := range paths {
		if sizes[p] > 0 {
			out = append(out, DataDir{Path: p, Bytes: sizes[p], Tier: TierUser, Kind: "data", Root: pathKind[p]})
		}
	}
	return out
}

// isSelfDataDir 报告目录名是否为应用自身（数据/缓存根下的 cli-analyzer）。
func isSelfDataDir(name string) bool {
	return strings.EqualFold(name, "cli-analyzer")
}

func goVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.GoVersion != "" {
		return bi.GoVersion
	}
	return runtime.Version()
}

// claimTopLevel 把 path 在任一扫描数据根下的顶层目录名加入 claimed。
// （cleanable 通常位于某数据根下；无法归属到根时忽略。）
func claimTopLevel(claimed map[string]bool, path string) {
	for _, k := range []platform.RootKind{
		platform.XDGCache, platform.XDGData, platform.XDGConfig,
		platform.AppData, platform.LocalAppData,
	} {
		root := platform.Root(k)
		if root == "" || !strings.HasPrefix(path, root+string(filepath.Separator)) {
			continue
		}
		rel := strings.TrimPrefix(path, root+string(filepath.Separator))
		if i := strings.IndexByte(rel, filepath.Separator); i > 0 {
			rel = rel[:i]
		}
		if rel != "" {
			claimed[rel] = true
		}
		return
	}
}
