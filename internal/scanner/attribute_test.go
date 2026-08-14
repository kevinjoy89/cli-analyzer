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
