package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/rules"
)

// TestAttributeMergesCuratedAliases 验证规则表 curated 别名（claude→claude-code、
// pip→pip3）并入 JSON aliases：此前 aliases 只含二进制名推断值，真正的别名
// 从未进入 JSON。
func TestAttributeMergesCuratedAliases(t *testing.T) {
	tbl := rules.Load()
	tools := map[string]*toolBuilder{
		"claude": {name: "claude", aliases: map[string]bool{},
			binaries: []Binary{{Name: "claude", Path: "/x/claude", Real: "/x/claude"}}},
		"pip": {name: "pip", aliases: map[string]bool{},
			binaries: []Binary{{Name: "pip", Path: "/x/pip", Real: "/x/pip"}}},
	}
	attribute(tools, []string{"claude", "pip"}, tbl, Options{}, &disk.Sizer{})
	if !tools["claude"].aliases["claude-code"] {
		t.Errorf("curated alias claude-code missing: %v", tools["claude"].aliases)
	}
	if !tools["pip"].aliases["pip3"] {
		t.Errorf("curated alias pip3 missing: %v", tools["pip"].aliases)
	}
}

// TestAttributeSkipsFamilyCuratedAliases 验证家族合并工具不并入 curated 别名：
// nodejs 规则带 node/npm/npx/corepack/node-gyp 清单，但那只是家族命令清单，
// 并入会把未安装的 node-gyp 也算进「包含工具」（家族别名来自实际发现的二进制）。
func TestAttributeSkipsFamilyCuratedAliases(t *testing.T) {
	// 隔离 HOME：nodejs 规则的 ~/.npm 等数据目录解析到临时目录，
	// 避免测量真实 npm 缓存（本机实测 12s+，且加重并发包负载）。
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	tbl := rules.Load()
	tools := map[string]*toolBuilder{
		"nodejs": {name: "nodejs", family: "nodejs", aliases: map[string]bool{},
			binaries: []Binary{{Name: "node", Path: "/x/node", Real: "/x/node"}}},
	}
	attribute(tools, []string{"nodejs"}, tbl, Options{}, &disk.Sizer{})
	if len(tools["nodejs"].aliases) != 0 {
		t.Errorf("family tool must not merge curated aliases: %v", tools["nodejs"].aliases)
	}
}

// TestReAttributeKeepsNpmGlobal 验证 npm 全局包工具（pi/opencode 等，InstNpm +
// installRoot）的 shim 二进制不被 reAttributeVendors 挪进宿主工具
// （nodejs 的 %APPDATA%\npm 数据目录）——否则会变成空壳行。
func TestReAttributeKeepsNpmGlobal(t *testing.T) {
	npmDir := filepath.Join(t.TempDir(), "npm")
	pkgDir := filepath.Join(npmDir, "node_modules", "@earendil-works", "pi-coding-agent")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nodejs := &toolBuilder{name: "nodejs", installer: InstNodejs,
		dataDirs: []DataDir{{Path: npmDir}}}
	pi := &toolBuilder{name: "pi", installer: InstNpm, installRoot: pkgDir,
		binaries: []Binary{{Name: "pi.cmd", Real: filepath.Join(npmDir, "pi.cmd")}}}
	tools := map[string]*toolBuilder{"nodejs": nodejs, "pi": pi}

	reAttributeVendors(tools, []string{"nodejs", "pi"})

	if len(pi.binaries) != 1 {
		t.Errorf("pi lost its shim binary: %+v", pi.binaries)
	}
	if len(nodejs.binaries) != 0 {
		t.Errorf("nodejs must not swallow npm global shim: %+v", nodejs.binaries)
	}
}

// TestReAttributeVendorsNoAlias 验证寄居二进制（宿主工具捆绑的 rg/fd 等）移入
// 宿主工具后不冒充别名：它们仍出现在二进制列表，但 aliases 不新增。
func TestReAttributeVendorsNoAlias(t *testing.T) {
	hostDir := filepath.Join(t.TempDir(), "kimi")
	tools := map[string]*toolBuilder{
		"kimi": {name: "kimi", aliases: map[string]bool{}, installRoot: hostDir},
		"rg": {name: "rg", aliases: map[string]bool{},
			binaries: []Binary{{Name: "rg", Path: filepath.Join(hostDir, "rg"), Real: filepath.Join(hostDir, "rg")}}},
	}
	reAttributeVendors(tools, []string{"kimi", "rg"})
	kimi := tools["kimi"]
	if len(kimi.binaries) != 1 || kimi.binaries[0].Name != "rg" {
		t.Fatalf("rg should move to kimi's binaries: %+v", kimi.binaries)
	}
	if len(kimi.aliases) != 0 {
		t.Errorf("host aliases must not gain bundled binary: %v", kimi.aliases)
	}
	if len(tools["rg"].binaries) != 0 {
		t.Errorf("vendor tool should lose its moved binary: %+v", tools["rg"].binaries)
	}
}

// TestAttributeGoUsesRealGopath 验证 go 工具的数据目录按真实 GOPATH 归因：
// rules 表硬编码 ~/go/pkg/mod，自定义 GOPATH（go env GOPATH 或 GOPATH 变量）
// 下模块缓存完全漏归因（本机实测 929MB 不可见，且展示不存在的 ~/go/pkg/mod）。
func TestAttributeGoUsesRealGopath(t *testing.T) {
	gopath := t.TempDir()
	t.Setenv("GOPATH", gopath)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// 构造 go 工具（brew 安装 + 真实模块缓存）
	modDir := filepath.Join(gopath, "pkg", "mod")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dlDir := filepath.Join(modDir, "cache", "download")
	if err := os.MkdirAll(dlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tb := &toolBuilder{name: "go", installer: InstGo, aliases: map[string]bool{}}
	tools := map[string]*toolBuilder{"go": tb}
	order := []string{"go"}

	sizer := &disk.Sizer{}
	attribute(tools, order, rules.Load(), Options{}, sizer)

	found := false
	for _, dd := range tb.dataDirs {
		if dd.Path == modDir {
			found = true
			break
		}
	}
	if !found {
		got := make([]string, 0, len(tb.dataDirs))
		for _, dd := range tb.dataDirs {
			got = append(got, dd.Path)
		}
		t.Errorf("go 工具 dataDirs 不含真实 GOPATH 模块缓存 %q: %v", modDir, got)
	}
	// 硬编码的 ~/go/pkg/mod 不应出现（HOME 已隔离，该目录不存在）
	for _, dd := range tb.dataDirs {
		if dd.Path == filepath.Join(home, "go", "pkg", "mod") {
			t.Errorf("硬编码 ~/go/pkg/mod 仍存在: %q", dd.Path)
		}
	}
	// 模块下载缓存 cleanable 应指向真实路径
	foundDL := false
	for _, c := range tb.cleanables {
		if c.Path == dlDir {
			foundDL = true
			break
		}
	}
	if !foundDL {
		t.Errorf("go 工具 cleanables 不含真实 GOPATH 下载缓存 %q", dlDir)
	}
}

// TestAttributeSkipsSelfCache 验证应用自身的缓存目录（~/.cache/cli-analyzer）
// 不作为可清理项：通用规则按工具名匹配会把"清理工具自己"列为 SAFE——
// clean --all 会清掉应用缓存（扫描快照/探测缓存丢失，下次全量重扫）。
func TestAttributeSkipsSelfCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cacheRoot := filepath.Join(home, ".cache")
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	selfCache := filepath.Join(cacheRoot, "cli-analyzer")
	if err := os.MkdirAll(selfCache, 0o755); err != nil {
		t.Fatal(err)
	}
	tb := &toolBuilder{name: "cli-analyzer", installer: InstOther, aliases: map[string]bool{}}
	tools := map[string]*toolBuilder{"cli-analyzer": tb}
	order := []string{"cli-analyzer"}
	sizer := &disk.Sizer{}
	attribute(tools, order, rules.Load(), Options{}, sizer)
	for _, c := range tb.cleanables {
		if c.Path == selfCache {
			t.Errorf("应用自身缓存 %q 不应列为可处置项", selfCache)
		}
	}
	for _, dd := range tb.dataDirs {
		if dd.Path == selfCache {
			t.Errorf("应用自身缓存 %q 不应归因为可清理缓存（应跳过或 USER）", selfCache)
		}
	}
}

// TestAttributeAllDataDirsActionable 验证两级门槛移除后所有归因数据目录
// （cache/config/data…，安装根除外）都成为可处置项：Tier 只是标签，
// 不再只对 cache 类生成可清理项。
func TestAttributeAllDataDirsActionable(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	cfg := filepath.Join(t.TempDir(), "config")
	installRoot := filepath.Join(t.TempDir(), "install")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	tb := &toolBuilder{name: "zzz-attr-all", installer: InstOther,
		dataDirs: []DataDir{
			{Path: cache, Tier: TierSafe, Kind: "cache"},
			{Path: cfg, Tier: TierUser, Kind: "config"},
			// 安装根是工具本体，不作为可处置项（删除安装根 = 卸载）
			{Path: installRoot, Tier: TierUser, Kind: "install"},
		}}
	tb.addMeasure(cache)
	tb.addMeasure(cfg)
	tools := map[string]*toolBuilder{"zzz-attr-all": tb}

	attribute(tools, []string{"zzz-attr-all"}, rules.Load(), Options{}, &disk.Sizer{})

	kinds := map[string]bool{}
	for _, c := range tb.cleanables {
		kinds[c.Kind] = true
		if c.Path == installRoot {
			t.Errorf("安装根 %q 不应成为可处置项", installRoot)
		}
	}
	if !kinds["cache"] || !kinds["config"] {
		t.Errorf("cleanables = %+v, want both cache and config actionable items", tb.cleanables)
	}
}
