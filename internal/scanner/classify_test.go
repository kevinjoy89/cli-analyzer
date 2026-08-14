package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// classify tests build paths from the *actual* environment (home dir, GOPATH,
// brew prefix, pyenv root) rather than hard-coding a machine's paths, so they
// hold on any developer/CI machine.
func TestClassifyVersioned(t *testing.T) {
	home, _ := os.UserHomeDir()
	c := classify(filepath.Join(home, ".local/share/claude/versions/2.1.226"), "claude")
	if c.ToolID != "claude" || c.Installer != InstVersioned || c.CurrentVersion != "2.1.226" {
		t.Errorf("versioned: %+v", c)
	}
}

func TestClassifyBrew(t *testing.T) {
	// Requires a brew prefix on the test machine; fall back to HOMEBREW_PREFIX.
	c := classify(brewPrefix()+"/Cellar/gh/2.5.0/bin/gh", "gh")
	if c.ToolID != "gh" || c.Installer != InstBrew || c.CurrentVersion != "2.5.0" {
		t.Errorf("brew: %+v", c)
	}
}

func TestClassifyNpmScoped(t *testing.T) {
	c := classify(filepath.Join(brewPrefix(), "lib/node_modules/@anthropic-ai/claude-code/bin/claude"), "claude")
	if c.ToolID != "@anthropic-ai/claude-code" || c.Installer != InstNpm {
		t.Errorf("npm scoped: %+v", c)
	}
}

func TestClassifyPyenvShim(t *testing.T) {
	c := classify(filepath.Join(pyenvShimsPath(), "python3"), "python3")
	if c.ToolID != "pyenv" || c.Installer != InstPyenv {
		t.Errorf("pyenv shim: %+v", c)
	}
	c2 := classify(filepath.Join(pyenvVersionsPath(), "3.12.2", "bin", "python3"), "python3")
	if c2.ToolID != "pyenv" || c2.Installer != InstPyenv {
		t.Errorf("pyenv build: %+v", c2)
	}
}

func TestClassifyGoCargoOther(t *testing.T) {
	if c := classify(filepath.Join(goBin(), "dlv"), "dlv"); c.Installer != InstGo {
		t.Errorf("go: %+v", c)
	}
	if c := classify(filepath.Join(cargoBin(), "cargo"), "cargo"); c.Installer != InstCargo {
		t.Errorf("cargo: %+v", c)
	}
	home, _ := os.UserHomeDir()
	if c := classify(filepath.Join(home, ".local/bin/mytool"), "mytool"); c.ToolID != "mytool" || c.Installer != InstOther {
		t.Errorf("other: %+v", c)
	}
}

// TestClassifyNodejsFamily 验证 Node.js 运行时家族合并：
// Windows 安装目录里的 node.exe / npm.cmd / npx.cmd / corepack.cmd / node-gyp.cmd
// 各自独立成行的问题 → 全部归并为一条 "nodejs"。
func TestClassifyNodejsFamily(t *testing.T) {
	for _, name := range []string{"node.exe", "npm.cmd", "npx.cmd", "corepack.cmd", "node-gyp.cmd", "npm"} {
		c := classify(filepath.Join(`C:\Program Files\nodejs`, name), name)
		if c.ToolID != "nodejs" || c.Installer != InstNodejs || c.Family != "nodejs" {
			t.Errorf("%s → %+v, want nodejs/%s family=nodejs", name, c, InstNodejs)
		}
	}
	// Windows 目录内存在 node.exe 时，安装根 = 该目录（整目录计入占用）
	c := classify(filepath.Join(t.TempDir(), "node.exe"), "node.exe")
	if c.InstallRoot != "" {
		t.Errorf("temp dir has no node.exe, install root should be empty: %+v", c)
	}
	// fnm 布局（unix）：node_modules 里的 npm/npx/corepack 也归 nodejs
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".local/share/fnm/node-versions/v22.0.0/installation/lib/node_modules")
	for _, pkg := range []string{"npm", "corepack"} {
		c := classify(filepath.Join(base, pkg, "bin", "npm-cli.js"), "npm")
		if c.ToolID != "nodejs" || c.Installer != InstNodejs || c.Family != "nodejs" {
			t.Errorf("fnm %s → %+v, want nodejs/%s family=nodejs", pkg, c, InstNodejs)
		}
	}
	// 非家族工具不受影响：Windows 保持原名（含扩展名），family 为空
	c2 := classify(`C:\Program Files\Git\cmd\git.exe`, "git.exe")
	if c2.ToolID != "git.exe" || c2.Installer != InstOther || c2.Family != "" {
		t.Errorf("git.exe → %+v, want git.exe/%s family empty", c2, InstOther)
	}
	// brew 公式 node 保持原名（已按公式合并；改名会破坏 brew uninstall node），
	// 且不是家族合并，family 为空
	c3 := classify(brewPrefix()+"/Cellar/node/22.14.0/bin/node", "node")
	if c3.ToolID != "node" || c3.Installer != InstBrew || c3.Family != "" {
		t.Errorf("brew node → %+v, want node/%s family empty", c3, InstBrew)
	}
	// yarn/pnpm 是独立包管理器，不属于 nodejs 家族
	c4 := classify(filepath.Join(`C:\Program Files\nodejs`, "yarn.cmd"), "yarn.cmd")
	if c4.ToolID != "yarn.cmd" || c4.Installer != InstOther {
		t.Errorf("yarn.cmd → %+v, want standalone", c4)
	}
}

// TestNodejsProbeOrder 验证 nodejs 合并工具把 node 排到 Binaries[0]，
// 让版本探测（--version）取到运行时版本而不是 corepack/npm。
func TestNodejsProbeOrder(t *testing.T) {
	tb := &toolBuilder{name: "nodejs", binaries: []Binary{
		{Name: "npx.cmd"}, {Name: "node.exe"}, {Name: "npm.cmd"},
	}}
	probeOrder(tb)
	if tb.binaries[0].Name != "node.exe" {
		t.Errorf("binaries[0] = %q, want node.exe", tb.binaries[0].Name)
	}
	// 非 nodejs 工具不动
	other := &toolBuilder{name: "gh", binaries: []Binary{{Name: "gh"}, {Name: "gh.exe"}}}
	probeOrder(other)
	if other.binaries[0].Name != "gh" {
		t.Errorf("non-nodejs tool must not be reordered")
	}
}

func TestUnder(t *testing.T) {
	if !under("/a/b/c", "/a/b") {
		t.Error("/a/b/c should be under /a/b")
	}
	if !under("/a/b", "/a/b") {
		t.Error("/a/b should be under /a/b")
	}
	if under("/a/bc", "/a/b") {
		t.Error("/a/bc should NOT be under /a/b")
	}
}

// TestSubKind 验证子项精确分类：日志类子项不继承父项 cache 类型。
func TestSubKind(t *testing.T) {
	cases := []struct{ parent, name, want string }{
		{"cache", "_logs", "logs"},          // npm 调试日志
		{"cache", "debug-0.log", "logs"},    // *.log 文件
		{"cache", "npm-debug.log", "logs"},
		{"cache", "_cacache", "cache"},      // 其余继承父项
		{"cache", "_npx", "cache"},
		{"cache", "_locks", "cache"},        // 不靠猜测扩大分类
		{"cache", "anonymous-cli-metrics.json", "cache"},
		{"download", "_logs", "logs"},       // 日志识别与父项类型无关
		{"cache", "catalog", "cache"},       // 不能把 catalog 误判为日志
	}
	for _, c := range cases {
		if got := subKind(c.parent, c.name); got != c.want {
			t.Errorf("subKind(%q, %q) = %q, want %q", c.parent, c.name, got, c.want)
		}
	}
}
