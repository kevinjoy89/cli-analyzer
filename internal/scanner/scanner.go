package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/rules"
	"cli-analyzer/internal/trash"
)

// Scan 执行完整扫描管线（GUI 手动"重新扫描"、CLI --refresh 使用）。
func Scan(opts Options) (*ScanResult, error) {
	res, _, err := scan(opts, false)
	return res, err
}

// ScanIfUnchanged 优先返回缓存的扫描结果：指纹（mtime+size）未变化时
// 跳过全量扫描（GUI 启动 / CLI 非 --refresh 路径），否则执行全量扫描。
// 指纹文件缺失（首次运行）时保守走全量。手动"重新扫描"请用 Scan（强制全量）。
// 第二个返回值报告是否执行了全量扫描（缓存命中为 false）——调用方据此
// 决定是否追加历史快照，避免缓存命中（含 CLI 先扫描后 GUI 命中新缓存）
// 重复记录。
func ScanIfUnchanged(opts Options) (*ScanResult, bool, error) {
	return scan(opts, true)
}

func scan(opts Options, skipIfUnchanged bool) (*ScanResult, bool, error) {
	if skipIfUnchanged {
		if cached, err := LoadCache(); err == nil {
			cur := ComputeFingerprint(cached)
			if saved, err := LoadFingerprint(); err == nil && FingerprintsEqual(cur, saved) {
				return cached, false, nil // 无变更：直接返回缓存，不扫描、不写历史
			}
		}
	}
	start := time.Now()
	// 顺带清除内置回收站的过期项（静默，失败不影响扫描）
	trash.Sweep()
	ruleTable := rules.Load()

	execs := discoverExecs()

	tools := map[string]*toolBuilder{}
	order := []string{}
	addBinary := func(id string, installer Installer, b Binary, currentVer, installRoot, family string) {
		tb := tools[id]
		if tb == nil {
			tb = &toolBuilder{name: id, aliases: map[string]bool{}}
			tools[id] = tb
			order = append(order, id)
		}
		// 同一真实文件经多个 PATH 目录（/usr/local/bin 与 ~/.orbstack/bin 的
		// docker 符号链接都指向 OrbStack xbin）会重复入列——按 Real 去重
		for _, e := range tb.binaries {
			if e.Real == b.Real {
				return
			}
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
		// 家族合并工具的别名用归一化命令名（Windows 上去掉 .exe/.cmd 等扩展名，
		// 显示 corepack 而非 corepack.cmd）；普通工具保留原始入口名。
		alias := b.Name
		if family != "" {
			alias = normEntryName(b.Name)
		}
		if alias != id {
			tb.aliases[alias] = true
		}
		if family != "" {
			tb.family = family
		}
		tb.binaries = append(tb.binaries, b)
	}

	for _, ex := range execs {
		real := ex.Real
		if real == "" {
			var err error
			real, err = filepath.EvalSymlinks(ex.Path)
			if err != nil {
				real = ex.Path
			}
		}
		var size int64
		if st, err := os.Stat(real); err == nil && st.Mode().IsRegular() {
			size = st.Size()
		}
		c := classify(real, ex.Name)
		addBinary(c.ToolID, c.Installer, Binary{Name: ex.Name, Path: ex.Path, Real: real, Size: size}, c.CurrentVersion, c.InstallRoot, c.Family)
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

	// 扫描耗时必须在写缓存之前计算：SaveCache 序列化的是当前 res，
	// 若在写缓存之后才赋值，缓存 JSON 里的 scanTimeMs 恒为 0
	// （GUI 启动从缓存渲染时会显示"扫描耗时 0ms"）。
	res.ScanTimeMS = time.Since(start).Milliseconds()

	if !opts.NoCache {
		// Cache the full (unfiltered) result: a filtered CLI scan like
		// `scan npm` must not clobber the snapshot the GUI starts from.
		cached := res
		if len(opts.ToolFilter) > 0 {
			cached = finalize(tools, order, Options{})
			// finalize 生成的是新 ScanResult：必须补上主流程计算、filter
			// 无关的字段——ScanTimeMS（扫描耗时）与 Unattributed（孤儿
			// 列表）。此前漏补导致 `scan <filter> --refresh` 写出的缓存
			// scanTimeMs=0、孤儿列表为空（GUI 从缓存渲染时表现异常）。
			cached.ScanTimeMS = res.ScanTimeMS
			cached.Unattributed = res.Unattributed
		}
		_ = SaveCache(cached)
		// 指纹与缓存同写：仅当实际写入缓存时（NoCache 不写）；基于未过滤
		// 的 cached 计算，与 ScanIfUnchanged 的缓存命中判定保持一致。
		_ = SaveFingerprint(ComputeFingerprint(cached))
	}
	return res, true, nil
}

// findUnattributed walks every top-level dir under the data roots that no tool
// claims, reporting non-empty ones. Candidates pass the non-CLI exclusion
// system (self dirs, structural GUI signals, vendor table) — see the
// non-cli-exclusion capability. Always runs (sizes computed in parallel).
func findUnattributed(tools map[string]*toolBuilder, order []string, sizer *disk.Sizer) []DataDir {
	claimed := map[string]bool{}
	for _, id := range order {
		for n := range tools[id].aliasSet() {
			// 大小写不敏感：Windows 上 PATH 名（claude）与数据目录名（Claude）
			// 可能大小写不同，map 键必须归一化才不会把真工具目录当孤儿。
			claimed[strings.ToLower(n)] = true
		}
	}
	// cleanable 规则覆盖的路径也算认领：codex 的 ~/.cache/codex-runtimes 等
	// 已作为该工具的 cleanable 提供，不应再列为孤儿。
	for _, id := range order {
		for _, c := range tools[id].cleanables {
			claimTopLevel(claimed, c.Path)
		}
	}
	// dataDirs 同样认领：目录名与工具名不同的数据目录
	// （如 opencode 的 ~/.config/oh-my-opencode）不能被当成孤儿。
	for _, id := range order {
		for _, dd := range tools[id].dataDirs {
			claimTopLevel(claimed, dd.Path)
		}
	}
	var paths []string
	var pathKind = map[string]string{}
	// 孤儿遍历仅限 CLI 工具主导的数据根（平台不适用时解析为 "" 自动跳过）：
	// macOS/Linux 用 XDG 目录（含 state——pnpm store、工具运行状态等残留
	// 同样是可展示的未认领数据）；Windows 用 AppData/LocalAppData。
	// macOS Application Support/Caches/Preferences 是 GUI 应用主导（Safari、
	// App Store、Chrome…），只用于已认领工具的归因，不作为孤儿来源。
	for _, k := range []platform.RootKind{
		platform.XDGCache, platform.XDGData, platform.XDGConfig, platform.XDGState,
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
			name := e.Name()
			// 仅目录是孤儿候选：.DS_Store 等普通文件、指向文件的符号链接
			// 一律跳过（它们不是数据目录）。
			p := filepath.Join(root, name)
			if !isDirEntry(p, e) {
				continue
			}
			if claimed[strings.ToLower(name)] {
				continue
			}
			// 非 CLI 排除体系（确定性规则）：本应用自身、结构性 GUI 信号
			// （macOS 容器 bundle-id / Windows UWP 包族及 Packages 容器）、
			// 系统/共享结构目录（.mono、configstore、%LocalAppData%\Programs…）、
			// 应用更新器目录（<App>-updater）、Windows 已安装应用交叉验证
			// （注册表卸载项/开始菜单快捷方式）、非 CLI 厂商排除表
			// （microsoft 等系统目录一并覆盖）。
			if isSelfDataDir(name) ||
				platform.IsContainerBundleDir(name) ||
				platform.IsUWPFamilyDir(name) ||
				strings.EqualFold(name, "packages") || // Windows UWP 容器目录
				platform.IsSystemDataDir(k, name) ||
				platform.IsUpdaterDir(name) ||
				platform.ExcludedByVendorData(p, name) ||
				platform.IsInstalledAppDataDir(name) {
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

// isSelfDataDir 报告目录名是否为应用自身：数据/缓存根下的 cli-analyzer、
// 应用产品名（CLI Analyzer / CLI Analyzer.exe——exe 可能被复制或改名运行，
// 便携包 exe 名为 cli-analyzer.exe 时产品名目录仍应视为自身），以及与本
// 应用可执行文件同名的目录。
func isSelfDataDir(name string) bool {
	switch strings.ToLower(name) {
	case "cli-analyzer", "cli analyzer", "cli analyzer.exe":
		return true
	}
	full, base := selfExeNames()
	return full != "" && (strings.EqualFold(name, full) || strings.EqualFold(name, base))
}

var selfExeOnce sync.Once
var selfExeFull, selfExeBase string

// selfExeNames 返回运行中可执行文件的基名（含扩展名）与去扩展名两种形态。
func selfExeNames() (full, base string) {
	selfExeOnce.Do(func() {
		if p, err := os.Executable(); err == nil {
			selfExeFull = filepath.Base(p)
			selfExeBase = strings.TrimSuffix(selfExeFull, filepath.Ext(selfExeFull))
		}
	})
	return selfExeFull, selfExeBase
}

func goVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.GoVersion != "" {
		return bi.GoVersion
	}
	return runtime.Version()
}

// isDirEntry 报告 ReadDir 条目是否为目录：普通目录直接通过；符号链接
// 跟随到目录才算（~/.config 下指向外部的链接目录是数据目录），指向文件
// 的链接（如 .DS_Store 的链接形态）不是。
func isDirEntry(p string, e os.DirEntry) bool {
	if e.Type().IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// claimTopLevel 把 path 在任一扫描数据根下的顶层目录名加入 claimed。
// （cleanable 通常位于某数据根下；无法归属到根时忽略。）
// root 列表与 findUnattributed 保持一致（含 XDGState），防止状态根下的
// cleanable 被当成孤儿。
func claimTopLevel(claimed map[string]bool, path string) {
	for _, k := range []platform.RootKind{
		platform.XDGCache, platform.XDGData, platform.XDGConfig, platform.XDGState,
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
			claimed[strings.ToLower(rel)] = true
		}
		return
	}
}
