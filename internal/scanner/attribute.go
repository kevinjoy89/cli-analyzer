package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/rules"
)

// toolBuilder accumulates everything for one tool during a scan.
type toolBuilder struct {
	name        string
	installer   Installer
	installRoot string
	currentVer  string
	aliases     map[string]bool
	binaries    []Binary
	dataDirs    []DataDir
	cleanables  []Cleanable
	measure     []string // paths to walk for footprint/cleanable sizing
	footprint   int64
	version     string
	updatedAt   string
	homepage    string
	description string
	// family 是工具家族合并根名（"nodejs"）；普通单工具为空。
	family string
}

func (b *toolBuilder) addMeasure(p string) {
	if p == "" {
		return
	}
	for _, q := range b.measure {
		if q == p {
			return
		}
	}
	b.measure = append(b.measure, p)
}

func (b *toolBuilder) aliasSet() map[string]bool {
	out := map[string]bool{b.name: true}
	for a := range b.aliases {
		out[a] = true
	}
	return out
}

// attribute computes data dirs, cleanables and sizes for every tool. It is the
// Phase C + Phase D of the scan pipeline.
func attribute(tools map[string]*toolBuilder, order []string, ruleTable *rules.Table, opts Options, sizer *disk.Sizer) {
	for _, id := range order {
		tb := tools[id]

		// 应用自身不归因：通用规则按工具名匹配会把"清理工具自己"列为
		// SAFE 可清理项，clean --all 会清掉扫描快照/探测缓存（自伤）
		if isSelfDataDir(id) {
			continue
		}

		cur := ruleTable.Lookup(id)

		// go 工具的 pkg/mod 归因必须按真实 GOPATH：rules 表硬编码
		// ~/go/pkg/mod，自定义 GOPATH（go env GOPATH / ~/.config/go/env）
		// 下模块缓存完全漏归因且展示不存在的目录。值拷贝后改写规则，
		// 不影响共享规则表。
		if id == "go" && cur != nil {
			rc := *cur
			cur = &rc
			gopath := goEnvGopath()
			if gopath != "" {
				mod := filepath.Join(gopath, "pkg", "mod")
				for i := range cur.DataDirs {
					if cur.DataDirs[i].Root == platform.Home && cur.DataDirs[i].Sub == "go/pkg/mod" {
						cur.DataDirs[i] = rules.DataDirRule{Path: mod, Tier: rules.TierUser, Kind: "data"}
					}
				}
				for i := range cur.Cleanables {
					if cur.Cleanables[i].Root == platform.Home && cur.Cleanables[i].Sub == "go/pkg/mod/cache/download" {
						cur.Cleanables[i] = rules.CleanRule{Path: filepath.Join(mod, "cache", "download"), Tier: rules.TierSafe, Kind: "download", Desc: cur.Cleanables[i].Desc}
					}
				}
			}
		}

		// Data dirs: curated (authoritative) or generic name-matching.
		if cur != nil {
			for _, dr := range cur.DataDirs {
				if p := dr.Resolve(); p != "" {
					tb.dataDirs = append(tb.dataDirs, DataDir{Path: p, Tier: Tier(dr.Tier), Kind: dr.Kind})
					tb.addMeasure(p)
				}
			}
		} else {
			for _, dr := range rules.GenericDataDirs(id) {
				if p := dr.Resolve(); p != "" {
					tb.dataDirs = append(tb.dataDirs, DataDir{Path: p, Tier: Tier(dr.Tier), Kind: dr.Kind})
					tb.addMeasure(p)
				}
			}
		}

		// Install root (brew Cellar, versions/, node_modules/<pkg>, venv…).
		if tb.installRoot != "" {
			tb.dataDirs = append(tb.dataDirs, DataDir{Path: tb.installRoot, Tier: TierUser, Kind: "install"})
			tb.addMeasure(tb.installRoot)
		}

		// Cache-kind data dirs are automatically SAFE cleanables.
		for _, dd := range tb.dataDirs {
			if dd.Kind == "cache" && dd.Tier == TierSafe {
				tb.cleanables = append(tb.cleanables, Cleanable{
					ID: id + "|cache|" + dd.Path, Tool: id, Path: dd.Path,
					Tier: TierSafe, Kind: "cache", Desc: i18n.T("ui.kind.cache"),
				})
			}
		}

		// Curated cleanables (npm cache, GOCACHE, model caches…).
		if cur != nil {
			for _, cr := range cur.Cleanables {
				for _, p := range rules.ResolveCleanable(cr) {
					tb.cleanables = append(tb.cleanables, Cleanable{
						ID: id + "|" + cr.Kind + "|" + p, Tool: id, Path: p,
						Tier: Tier(cr.Tier), Kind: cr.Kind, Keep: cr.Keep, Desc: cr.Desc,
					})
					tb.addMeasure(p)
				}
			}
		}

		// Old versions / toolchains derived from the installer layout.
		tb.deriveOldVersions(id)
	}

	// Re-attribute vendored binaries (a tool shipping its own copy of another
	// command, e.g. kimi bundling rg/fd) to the owning tool.
	reAttributeVendors(tools, order)

	// Backups (*.bak / *.old): each file owned by exactly one tool, matched by
	// name (kimi.bak -> kimi) or the first tool with a binary in that dir.
	for path, tb := range collectBackups(tools, order) {
		id := tb.name
		tb.cleanables = append(tb.cleanables, Cleanable{
			ID: id + "|backup|" + path, Tool: id, Path: path,
			Tier: TierSafe, Kind: "backup", Desc: i18n.T("ui.kind.backup"),
		})
		tb.addMeasure(path)
	}

	// Display metadata: current version, newest install time, homepage/description.
	for _, id := range order {
		tb := tools[id]
		tb.version = tb.currentVersion()
		tb.updatedAt = tb.updatedAtTime()
		if m := ruleTable.Meta(id); m.Homepage != "" || m.Description != "" {
			tb.homepage = m.Homepage
			tb.description = i18n.T(m.Description)
		}
		// 规则表别名（claude→claude-code、pip→pip3、hf…）并入展示字段：
		// 此前 aliases 只含二进制名推断值，真正的别名从未进入 JSON。
		// 家族合并工具（nodejs）除外——其别名即实际合并进来的命令，curated
		// 别名只是家族清单，并入会把未安装的 corepack/node-gyp 也算进「包含工具」。
		if tb.family == "" {
			if cur := ruleTable.Lookup(id); cur != nil {
				for _, a := range cur.Aliases {
					if a != id {
						tb.aliases[a] = true
					}
				}
			}
		}
	}

	// ---- global sizing pass ----
	all := []string{}
	seen := map[string]bool{}
	for _, id := range order {
		for _, p := range tools[id].measure {
			if !seen[p] {
				seen[p] = true
				all = append(all, p)
			}
		}
	}
	sizes := sizer.WalkAll(all)

	for _, id := range order {
		tb := tools[id]
		for i := range tb.dataDirs {
			tb.dataDirs[i].Bytes = sizes[tb.dataDirs[i].Path]
		}
		for i := range tb.cleanables {
			tb.cleanables[i].Bytes = sizes[tb.cleanables[i].Path]
		}
		tb.footprint = footprintOf(tb, sizes)
	}

	// One-level size breakdown for SAFE cleanables, so the UI can show what is
	// inside (e.g. ~/.npm -> _cacache 9.5G, _npx 728M). USER items are hidden
	// in the UI anyway, so skip them to keep the rescan fast.
	for _, id := range order {
		tb := tools[id]
		for i := range tb.cleanables {
			c := &tb.cleanables[i]
			if c.Tier != TierSafe || c.Bytes <= 0 {
				continue
			}
			for p, n := range sizer.ChildrenSizes(c.Path) {
				c.Sub = append(c.Sub, SubEntry{
					Path: p, Bytes: n, ID: c.ID + "::" + p,
					Kind: subKind(c.Kind, filepath.Base(p)),
				})
			}
			sort.Slice(c.Sub, func(a, b int) bool { return c.Sub[a].Bytes > c.Sub[b].Bytes })
		}
	}
}

// subKind 给可清理项的直接子项一个精确类型，而不是无条件继承父项：
//
//	~/.npm/_logs → logs（npm 调试日志，不是缓存）
//	~/.npm/_cacache → cache（继承父项）
//
// 只识别有把握的类别（日志），其余继承父项 —— 不靠猜测扩大分类，宁缺毋滥。
func subKind(parentKind, name string) string {
	low := strings.ToLower(name)
	if low == "_logs" || strings.HasSuffix(low, ".log") {
		return "logs"
	}
	return parentKind
}

// reAttributeVendors moves binaries that live inside another tool's attributed
// dirs (data dir or install root) to that tool, so kimi's bundled rg/fd count
// toward kimi rather than appearing as standalone tools.
func reAttributeVendors(tools map[string]*toolBuilder, order []string) {
	type owner struct {
		tb   *toolBuilder
		dirs []string
	}
	var all []owner
	for _, id := range order {
		tb := tools[id]
		var dirs []string
		for _, dd := range tb.dataDirs {
			dirs = append(dirs, dd.Path)
		}
		if tb.installRoot != "" {
			dirs = append(dirs, tb.installRoot)
		}
		all = append(all, owner{tb, dirs})
	}
	type move struct {
		from, to *toolBuilder
		b        Binary
	}
	var moves []move
	for _, id := range order {
		tb := tools[id]
		kept := tb.binaries[:0]
		for _, b := range tb.binaries {
			// npm 全局包（InstNpm + installRoot，如 pi → @earendil-works/
			// pi-coding-agent）是真实工具，不是宿主工具捆绑的依赖——即使其
			// shim 位于其他工具的数据目录（nodejs 的 %APPDATA%\npm）也保留
			// 在自己的工具下，避免被吞成空壳行。
			if tb.installer == InstNpm && tb.installRoot != "" {
				kept = append(kept, b)
				continue
			}
			var target *toolBuilder
			for _, o := range all {
				if o.tb == tb {
					continue
				}
				for _, d := range o.dirs {
					if under(b.Real, d) {
						target = o.tb
						break
					}
				}
				if target != nil {
					break
				}
			}
			if target != nil {
				moves = append(moves, move{tb, target, b})
			} else {
				kept = append(kept, b)
			}
		}
		tb.binaries = kept
	}
	for _, m := range moves {
		m.to.binaries = append(m.to.binaries, m.b)
		// 寄居二进制（rg/fd 等）只是宿主工具捆绑的依赖，不是它的别名
		// （opencode/kimi 自带 ripgrep → 别名里不该出现 "rg"）。
		// 它们仍会出现在该工具的二进制列表中，只是不冒充别名。
	}
}

// collectBackups returns one owning tool per *.bak / *.old file found next to
// a discovered binary. Files that are themselves PATH executables are skipped
// (they are already accounted as their own tool).
func collectBackups(tools map[string]*toolBuilder, order []string) map[string]*toolBuilder {
	// A file that is itself a discovered command (e.g. agy.123.old in
	// ~/.local/bin) is already counted as a tool; don't double-count it.
	// Symlinked commands must also protect their real target: if kimi is a
	// symlink to kimi.bak, the .bak file is a live command, not a backup.
	isCommand := map[string]bool{}
	for _, id := range order {
		for _, b := range tools[id].binaries {
			isCommand[b.Path] = true
			isCommand[b.Real] = true
		}
	}
	seenDir := map[string]bool{}
	var paths []string
	for _, id := range order {
		tb := tools[id]
		for _, bin := range tb.binaries {
			dir := filepath.Dir(bin.Real)
			if seenDir[dir] {
				continue
			}
			seenDir[dir] = true
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				low := strings.ToLower(e.Name())
				if !strings.HasSuffix(low, ".bak") && !strings.HasSuffix(low, ".old") {
					continue
				}
				p := filepath.Join(dir, e.Name())
				if isCommand[p] {
					continue
				}
				paths = append(paths, p)
			}
		}
	}
	dirTool := map[string]*toolBuilder{}
	for _, id := range order {
		tb := tools[id]
		for _, bin := range tb.binaries {
			d := filepath.Dir(bin.Real)
			if _, ok := dirTool[d]; !ok {
				dirTool[d] = tb
			}
		}
	}
	out := map[string]*toolBuilder{}
	for _, p := range paths {
		owner := backupOwner(filepath.Base(p), tools)
		if owner == nil {
			owner = dirTool[filepath.Dir(p)]
		}
		if owner == nil {
			continue
		}
		out[p] = owner
	}
	return out
}

// backupOwner matches a backup basename (kimi.bak, agy.1784.old) to a tool by
// name, stripping version suffixes.
func backupOwner(base string, tools map[string]*toolBuilder) *toolBuilder {
	name := strings.ToLower(base)
	for _, suf := range []string{".bak", ".old"} {
		if strings.HasSuffix(name, suf) {
			name = strings.TrimSuffix(name, suf)
			break
		}
	}
	name = strings.TrimRight(name, ".0123456789")
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return nil
	}
	for _, tb := range tools {
		if strings.EqualFold(tb.name, name) {
			return tb
		}
		for a := range tb.aliases {
			if strings.EqualFold(a, name) {
				return tb
			}
		}
	}
	return nil
}

// deriveOldVersions adds old-version / toolchain cleanables for installers that
// keep multiple versions (versioned, brew, pyenv, rustup).
func (b *toolBuilder) deriveOldVersions(id string) {
	// pyenv's active version lives in ~/.pyenv/version and wins over the
	// brew-formula version of the pyenv command itself.
	if b.installer == InstPyenv {
		if v := pyenvDefaultVersion(); v != "" {
			b.currentVer = v
		}
	}
	if b.currentVer == "" {
		for _, bin := range b.binaries {
			if _, v, ok := versionedMatch(bin.Real); ok {
				b.currentVer = v
				break
			}
			if v, ok := toolchainVersion(bin.Real); ok {
				b.currentVer = v
				break
			}
		}
	}
	kind := ""
	root := ""
	switch b.installer {
	case InstVersioned:
		kind, root = "old-version", b.installRoot
	case InstBrew:
		// Do NOT auto-derive brew old-versions: a non-current Cellar dir (e.g.
		// fontconfig/2.17.1) may still be depended on by other formulae, and
		// deleting it breaks them. brew cleanup handles this safely; we leave
		// brew versions alone.
		return
	case InstPyenv:
		kind, root = "toolchain", pyenvVersionsPath()
	case InstRustup:
		kind, root = "toolchain", rustupToolchainsPath()
	}
	if kind == "" || root == "" || b.currentVer == "" {
		return
	}
	keep := "current symlink target: " + b.currentVer
	desc := i18n.T("ui.kind.oldVersion")
	tier := TierSafe
	if kind == "toolchain" {
		keep = "default toolchain: " + b.currentVer
		// pyenv/rustup toolchains are load-bearing: pip-installed commands hardcode
		// the interpreter path in their shebangs (e.g. ~/.pyenv/versions/3.6.15/bin),
		// and projects pin toolchains via rust-toolchain.toml. Nothing safe to
		// reference-check at scan time, so toolchains are display-only — removing
		// them is always manual (pyenv uninstall / rustup toolchain uninstall).
		tier = TierUser
		desc = i18n.T("ui.kind.oldToolchain")
	}
	currentPath := filepath.Join(root, b.currentVer)
	for _, v := range listNames(root) {
		if v == b.currentVer {
			continue
		}
		p := filepath.Join(root, v)
		b.cleanables = append(b.cleanables, Cleanable{
			ID: id + "|" + kind + "|" + p, Tool: id, Path: p,
			Tier: tier, Kind: kind, Keep: keep, Desc: desc + " " + id + " " + v,
			CurrentPath: currentPath,
		})
		b.addMeasure(p)
	}
}

// currentVersion returns the tool's version: the versioned/brew/pyenv current
// version when known, else the npm package.json version.
func (b *toolBuilder) currentVersion() string {
	if b.currentVer != "" {
		return b.currentVer
	}
	if b.installer == InstNpm && b.installRoot != "" {
		if data, err := os.ReadFile(filepath.Join(b.installRoot, "package.json")); err == nil {
			var p struct {
				Version string `json:"version"`
			}
			if json.Unmarshal(data, &p) == nil && p.Version != "" {
				return p.Version
			}
		}
	}
	return ""
}

// updatedAtTime returns the newest mtime (RFC3339) among the tool's install
// root and its binaries — a proxy for "last updated / installed".
func (b *toolBuilder) updatedAtTime() string {
	var latest int64
	consider := func(p string) {
		if st, err := os.Stat(p); err == nil {
			if t := st.ModTime().Unix(); t > latest {
				latest = t
			}
		}
	}
	if b.installRoot != "" {
		consider(b.installRoot)
	}
	for _, bin := range b.binaries {
		consider(bin.Real)
	}
	if latest == 0 {
		return ""
	}
	return time.Unix(latest, 0).Format(time.RFC3339)
}

// footprintOf sums the maximal measured dirs plus any binary files not inside
// a measured dir (standalone go/cargo/other binaries).
func footprintOf(tb *toolBuilder, sizes map[string]int64) int64 {
	total := sumMaximal(tb.measure, sizes)
	for _, bin := range tb.binaries {
		inside := false
		for _, p := range tb.measure {
			if under(bin.Real, p) {
				inside = true
				break
			}
		}
		if !inside && bin.Size > 0 {
			total += bin.Size
		}
	}
	return total
}

// sumMaximal sums sizes of paths that have no ancestor path in the same set.
func sumMaximal(paths []string, sizes map[string]int64) int64 {
	var total int64
	for i, p := range paths {
		isChild := false
		for j, q := range paths {
			if i != j && under(p, q) {
				isChild = true
				break
			}
		}
		if !isChild {
			total += sizes[p]
		}
	}
	return total
}

// toolchainVersion extracts the toolchain name from a rustup real path.
// Windows 上同时识别反斜杠分隔；unix 上保持仅正斜杠（反斜杠是 unix 合法
// 文件名字符，不能当分隔符切，行为与原实现逐位一致）。
func toolchainVersion(real string) (string, bool) {
	var segs []string
	if runtime.GOOS == "windows" {
		segs = strings.FieldsFunc(real, func(r rune) bool { return r == '/' || r == '\\' })
	} else {
		segs = strings.Split(real, "/")
	}
	for i := range segs {
		if segs[i] == "toolchains" && i+1 < len(segs) {
			return segs[i+1], true
		}
	}
	return "", false
}

func rustupToolchainsPath() string {
	// RUSTUP_HOME 可自定义（默认 ~/.rustup）
	if r := os.Getenv("RUSTUP_HOME"); r != "" {
		return filepath.Join(r, "toolchains")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".rustup", "toolchains")
	}
	return filepath.Join(".rustup", "toolchains")
}

func listNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		// Skip dotfiles (.DS_Store etc.) — they are never versions/toolchains.
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, e.Name())
	}
	return out
}

// pyenvDefaultVersion reads the active pyenv version from ~/.pyenv/version.
func pyenvDefaultVersion() string {
	if h, err := os.UserHomeDir(); err != nil {
		return ""
	} else {
		data, err := os.ReadFile(filepath.Join(h, ".pyenv", "version"))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

// finalize converts builders into the JSON contract, prunes empty tools,
// applies the tool filter and computes totals.
func finalize(tools map[string]*toolBuilder, order []string, opts Options) *ScanResult {
	var out []Tool
	totals := Totals{}
	for _, id := range order {
		tb := tools[id]
		// 应用自身从结果中完全剔除（attribute 已跳过归因，这里跳过展示）：
		// "清理工具自己"出现在列表里是噪音，且其缓存曾可被 SAFE 清理
		if isSelfDataDir(id) {
			continue
		}
		// nodejs 合并工具把 node 排到 Binaries[0]，让版本探测取到运行时版本。
		probeOrder(tb)
		var cleanSum, userSum int64
		for _, c := range tb.cleanables {
			if c.Tier == TierSafe {
				cleanSum += c.Bytes
			}
		}
		userSum = tb.footprint - cleanSum
		if userSum < 0 {
			userSum = 0
		}
		if tb.footprint == 0 && len(tb.binaries) == 0 && len(tb.cleanables) == 0 && len(tb.dataDirs) == 0 {
			continue
		}
		if !matchFilter(tb, opts.ToolFilter) {
			continue
		}
		aliases := make([]string, 0, len(tb.aliases))
		for a := range tb.aliases {
			aliases = append(aliases, a)
		}
		sort.Strings(aliases)
		// Drop cleanables whose path no longer exists (deleted between scans).
		kept := tb.cleanables[:0]
		for _, c := range tb.cleanables {
			if _, err := os.Lstat(c.Path); err == nil || c.Bytes > 0 {
				kept = append(kept, c)
			}
		}
		tb.cleanables = kept
		for i := range tb.cleanables {
			if tb.cleanables[i].Sub == nil {
				tb.cleanables[i].Sub = []SubEntry{}
			}
		}
		// 确定性排序：同大小清理项按路径升序
		sort.Slice(tb.cleanables, func(i, j int) bool {
			if tb.cleanables[i].Bytes != tb.cleanables[j].Bytes {
				return tb.cleanables[i].Bytes > tb.cleanables[j].Bytes
			}
			return tb.cleanables[i].Path < tb.cleanables[j].Path
		})
		// Marshal empty slices as [] rather than null for JSON consumers.
		if tb.binaries == nil {
			tb.binaries = []Binary{}
		}
		if tb.dataDirs == nil {
			tb.dataDirs = []DataDir{}
		}
		if tb.cleanables == nil {
			tb.cleanables = []Cleanable{}
		}
		out = append(out, Tool{
			Name: tb.name, Aliases: aliases, Installer: string(tb.installer),
			Version: tb.version, UpdatedAt: tb.updatedAt,
			Homepage: tb.homepage, Description: tb.description,
			Family:   tb.family,
			Binaries: tb.binaries, DataDirs: tb.dataDirs, Cleanables: tb.cleanables,
			Footprint: tb.footprint, Cleanable: cleanSum, User: userSum,
		})
		totals.Footprint += tb.footprint
		totals.Cleanable += cleanSum
		totals.User += userSum
	}
	// 确定性排序：同 footprint 的工具按名称升序——sort.Slice 不稳定，
	// 等值元素顺序随输入抖动，两次扫描的展示顺序会无规律变化
	sort.Slice(out, func(i, j int) bool {
		if out[i].Footprint != out[j].Footprint {
			return out[i].Footprint > out[j].Footprint
		}
		return out[i].Name < out[j].Name
	})

	roots := map[string][]string{}
	for _, k := range platform.AllKinds {
		if r := platform.Root(k); r != "" {
			roots[string(k)] = []string{r}
		}
	}

	return &ScanResult{
		ScannedAt: Now(), Platform: platform.OS(), GoVersion: goVersion(),
		Tools: out, Totals: totals, Roots: roots,
	}
}

func matchFilter(tb *toolBuilder, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	names := tb.aliasSet()
	for _, f := range filters {
		for n := range names {
			if strings.Contains(n, f) {
				return true
			}
		}
	}
	return false
}
