package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if brewPrefix() == "" {
		t.Skip("no brew prefix on this machine")
	}
	c := classify(brewPrefix()+"/Cellar/gh/2.5.0/bin/gh", "gh")
	if c.ToolID != "gh" || c.Installer != InstBrew || c.CurrentVersion != "2.5.0" {
		t.Errorf("brew: %+v", c)
	}
}

func TestClassifyNpmScoped(t *testing.T) {
	// @anthropic-ai/claude-code → claude（npmToolID 映射：规则行与 npm 行归并）
	c := classify(filepath.Join(brewPrefix(), "lib/node_modules/@anthropic-ai/claude-code/bin/claude"), "claude")
	if c.ToolID != "claude" || c.Installer != InstNpm {
		t.Errorf("npm scoped: %+v", c)
	}
	// 未映射的 scoped 包保持包名
	c2 := classify(filepath.Join(brewPrefix(), "lib/node_modules/@openai/codex/bin/codex.js"), "codex")
	if c2.ToolID != "codex" || c2.Installer != InstNpm {
		t.Errorf("npm scoped codex: %+v", c2)
	}
}

// TestClassifyPipFamily 验证 pip/pip3/pip3.x 命令归并为一条 pip（含 brew
// Cellar 里的 pip3——家族检查在 brew 分支之前）。pyenv shims 的 pip3 保留给 pyenv。
func TestClassifyPipFamily(t *testing.T) {
	for _, name := range []string{"pip", "pip3", "pip3.9", "pip3.12"} {
		c := classify(filepath.Join("/opt/homebrew/Cellar/python@3.12/3.12.13_4/bin", name), name)
		if c.ToolID != "pip" || c.Installer != InstPip || c.Family != "pip" {
			t.Errorf("%s → %+v, want pip/%s family=pip", name, c, InstPip)
		}
	}
	if c := classify(filepath.Join(pyenvShimsPath(), "pip3"), "pip3"); c.ToolID != "pyenv" {
		t.Errorf("pyenv pip3 → %+v, want pyenv", c)
	}
	// pipx/pipenv 等独立产品不归并
	for _, name := range []string{"pipx", "pipenv"} {
		if _, ok := pipFamilyRoot(name); ok {
			t.Errorf("%s 不应归入 pip 家族", name)
		}
	}
}

// TestClassifyCompanionFamily 验证伴随命令归并：mimo→mimocode、
// kimi-webbridge/kimi-slides→kimi、opencode-plugins→opencode。
func TestClassifyCompanionFamily(t *testing.T) {
	cases := map[string]string{
		"mimo":             "mimocode",
		"kimi-webbridge":   "kimi",
		"kimi-slides":      "kimi",
		"opencode-plugins": "opencode",
	}
	for name, want := range cases {
		c := classify(filepath.Join("/Users/x/."+name+"/bin", name), name)
		if c.ToolID != want || c.Family != want || c.Installer != InstOther {
			t.Errorf("%s → %+v, want %s family=%s other", name, c, want, want)
		}
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
	// ~/.local/bin 是官方脚本安装落点（astral uv、poetry 等）→ InstLocalBin；
	// 其他散放路径仍是 InstOther
	if c := classify(filepath.Join(home, ".local/bin/mytool"), "mytool"); c.ToolID != "mytool" || c.Installer != InstLocalBin {
		t.Errorf("local bin: %+v", c)
	}
	if c := classify(filepath.Join(home, "bin/mytool"), "mytool"); c.ToolID != "mytool" || c.Installer != InstOther {
		t.Errorf("other: %+v", c)
	}
}

// TestClassifyNodejsFamily 验证 Node.js 运行时家族合并：
// Windows 安装目录里的 node.exe / npm.cmd / npx.cmd / corepack.cmd / node-gyp.cmd
// 各自独立成行的问题 → 全部归并为一条 "nodejs"。
func TestClassifyNodejsFamily(t *testing.T) {
	for _, name := range []string{"node.exe", "npm.cmd", "npx.cmd", "corepack.cmd", "node-gyp.cmd", "npm", "install_tools.bat", "nodevars.bat"} {
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
	// Git 家族工具不并入 nodejs：git.exe 归入 git 家族（非 nodejs），
	// 保持 family=git；yarn/pnpm 是独立包管理器，不属于任何家族。
	c2 := classify(`C:\Program Files\Git\cmd\git.exe`, "git.exe")
	if c2.ToolID != "git" || c2.Family != "git" {
		t.Errorf("git.exe → %+v, want git family", c2)
	}
	// yarn/pnpm 是独立包管理器，不属于 nodejs 家族
	c4 := classify(filepath.Join(`C:\Program Files\nodejs`, "yarn.cmd"), "yarn.cmd")
	if c4.ToolID != "yarn.cmd" || c4.Installer != InstOther {
		t.Errorf("yarn.cmd → %+v, want standalone", c4)
	}
}

// TestNodejsFamilyBrewStaysIndependent 验证 brew 公式 node 保持原名（已按公式
// 合并；改名会破坏 brew uninstall node），且不参与家族合并。依赖机器上有 brew
// prefix（macOS/CI ubuntu 有 linuxbrew），无 brew 环境跳过。
func TestNodejsFamilyBrewStaysIndependent(t *testing.T) {
	if brewPrefix() == "" {
		t.Skip("no brew prefix on this machine")
	}
	c := classify(brewPrefix()+"/Cellar/node/22.14.0/bin/node", "node")
	if c.ToolID != "node" || c.Installer != InstBrew || c.Family != "" {
		t.Errorf("brew node → %+v, want node/%s family empty", c, InstBrew)
	}
}

// TestClassifyGitFamily 验证 Git 家族合并：Git for Windows cmd/ 目录中的
// git.exe/git-lfs.exe/scalar.exe/tig.exe/start-ssh-agent.cmd 等归并为一条
// "git"；gh/hub 是独立产品不合并；brew 公式 git-lfs 保持独立。
func TestClassifyGitFamily(t *testing.T) {
	dir := t.TempDir()
	// 目录内放 git.exe 标记（gitInstallRoot 判定安装根）
	if err := os.WriteFile(filepath.Join(dir, "git.exe"), []byte("not-a-real-pe"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"git.exe", "git-lfs.exe", "scalar.exe", "tig.exe",
		"git-receive-pack.exe", "git-upload-pack.exe", "git-upload-archive.exe",
		"start-ssh-agent.cmd", "start-ssh-pageant.cmd",
	} {
		c := classify(filepath.Join(dir, name), name)
		if c.ToolID != "git" || c.Family != "git" {
			t.Errorf("%s → %+v, want git/git family", name, c)
		}
		if c.InstallRoot != dir {
			t.Errorf("%s InstallRoot = %q, want %q", name, c.InstallRoot, dir)
		}
	}
	// 非家族：gh/hub 是独立 CLI 产品
	for _, name := range []string{"gh.exe", "hub.exe"} {
		if c := classify(filepath.Join(dir, name), name); c.ToolID != name || c.Family != "" {
			t.Errorf("%s → %+v, want independent tool", name, c)
		}
	}
	// 无 git.exe 标记的目录：家族命中但安装根为空（按文件计大小）
	empty := t.TempDir()
	if c := classify(filepath.Join(empty, "tig.exe"), "tig.exe"); c.Family != "git" || c.InstallRoot != "" {
		t.Errorf("tig without git marker → %+v, want family git + empty install root", c)
	}
	// brew 公式 git-lfs 保持独立公式身份（brew 分支优先于家族）
	if brewPrefix() != "" {
		c := classify(brewPrefix()+"/Cellar/git-lfs/3.7.1/bin/git-lfs", "git-lfs")
		if c.ToolID != "git-lfs" || c.Family != "" {
			t.Errorf("brew git-lfs → %+v, want independent formula", c)
		}
	}
}

// TestClassifyNpmGlobalShim 验证 npm 全局 shim（Windows %APPDATA%\npm 根部
// 的 <name>.cmd）解析为对应的 npm 包工具：pi → @earendil-works/pi-coding-agent
// （bin=pi）、opencode → opencode-ai（bin=opencode）；无法解析时保持 InstOther。
func TestClassifyNpmGlobalShim(t *testing.T) {
	prefix := t.TempDir()
	mkPkg := func(rel, bin string) string {
		pkgDir := filepath.Join(prefix, "node_modules", rel)
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := "{\"name\":\"" + filepath.Base(rel) + "\",\"bin\":{\"" + bin + "\":\"dist/cli.js\"}}"
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		return pkgDir
	}
	piDir := mkPkg(filepath.Join("@earendil-works", "pi-coding-agent"), "pi")
	ocDir := mkPkg("opencode-ai", "opencode")

	// 作用域包 shim：pi.cmd → pi
	c := classify(filepath.Join(prefix, "pi.cmd"), "pi.cmd")
	if c.ToolID != "pi" || c.Installer != InstNpm || c.InstallRoot != piDir || c.Family != "" {
		t.Errorf("pi.cmd → %+v, want pi/%s root=%s", c, InstNpm, piDir)
	}
	// 普通包 shim：opencode.cmd → opencode（bin 名而非包名 opencode-ai）
	c = classify(filepath.Join(prefix, "opencode.cmd"), "opencode.cmd")
	if c.ToolID != "opencode" || c.Installer != InstNpm || c.InstallRoot != ocDir {
		t.Errorf("opencode.cmd → %+v, want opencode/%s root=%s", c, InstNpm, ocDir)
	}
	// 无 node_modules 的目录：保持 InstOther（原行为）
	plain := t.TempDir()
	if c := classify(filepath.Join(plain, "mytool.cmd"), "mytool.cmd"); c.ToolID != "mytool.cmd" || c.Installer != InstOther {
		t.Errorf("mytool.cmd → %+v, want InstOther", c)
	}
	// 非 shim 形态（.exe 直接放根部）：不触发 npm 全局解析
	if c := classify(filepath.Join(prefix, "pi.exe"), "pi.exe"); c.ToolID != "pi.exe" || c.Installer != InstOther {
		t.Errorf("pi.exe → %+v, want InstOther", c)
	}
	// 代理 bin 不匹配：corepack 声明 yarn/pnpm 代理 bin，但包名与 shim 名
	// 无关（CI 的 Node 安装即含 corepack），yarn.cmd 必须保持独立
	corepackDir := filepath.Join(prefix, "node_modules", "corepack")
	if err := os.MkdirAll(corepackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corepackDir, "package.json"),
		[]byte(`{"name":"corepack","bin":{"yarn":"dist/yarn.js","pnpm":"dist/pnpm.js"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"yarn.cmd", "pnpm.cmd"} {
		if c := classify(filepath.Join(prefix, name), name); c.ToolID != name || c.Installer != InstOther {
			t.Errorf("%s → %+v, want standalone (corepack proxy bin must not match)", name, c)
		}
	}
}

// TestNodejsProbeOrder 验证 nodejs 合并工具把 node 排到 Binaries[0]，
// 让版本探测（--version）取到运行时版本而不是 corepack/npm。
// git 合并工具同理把 git 排到 Binaries[0]（而非 git-lfs/tig）。
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
	// git 家族：git 排到最前
	git := &toolBuilder{name: "git", binaries: []Binary{
		{Name: "git-lfs.exe"}, {Name: "start-ssh-agent.cmd"}, {Name: "git.exe"},
	}}
	probeOrder(git)
	if git.binaries[0].Name != "git.exe" {
		t.Errorf("git binaries[0] = %q, want git.exe", git.binaries[0].Name)
	}
}

func TestUnder(t *testing.T) {
	// 用本机分隔符构造路径，Windows 与 unix 通用
	a := filepath.Join("a", "b")
	if !under(filepath.Join(a, "c"), a) {
		t.Error(filepath.Join(a, "c") + " should be under " + a)
	}
	if !under(a, a) {
		t.Error(a + " should be under itself")
	}
	if under(filepath.Join("a", "bc"), a) {
		t.Error(filepath.Join("a", "bc") + " should NOT be under " + a)
	}
}

// TestUnderFold 验证 Windows 大小写不敏感路径包含：GOPATH 环境变量
// （C:\Users\X\Go\bin）与 EvalSymlinks 解析的真实路径大小写可能不同
// （C:\Users\X\go\bin\tool），大小写敏感比较会让 go/cargo/pyenv 归因失效
// （工具被归为 other，安装根与卸载建议丢失）。
func TestUnderFold(t *testing.T) {
	// 正斜杠形式：underFold 内部 ToSlash 归一化，与平台无关可测
	if !underFold(`C:/Users/X/go/bin/tool`, `C:/Users/X/Go/bin`) {
		t.Error("不同大小写应判为包含（Windows 文件系统大小写不敏感）")
	}
	if !underFold(`C:/Users/X/Go/bin`, `C:/Users/X/go/bin`) {
		t.Error("反向大小写也应判为包含")
	}
	if underFold(`C:/Users/X/binaries`, `C:/Users/X/bin`) {
		t.Error("前缀必须带分隔符边界（binaries 不是 bin 的子路径）")
	}
	if underFold(``, ``) {
		t.Error("空路径不应判为包含")
	}
}

// TestPathSplitPlatformContract 固化分隔符处理的平台契约：Windows 上反斜杠
// 是分隔符；unix 上反斜杠是合法文件名字符，必须保留在段内（防止回归把
// unix 上含反斜杠的真实目录名错误切分）。
func TestPathSplitPlatformContract(t *testing.T) {
	if runtime.GOOS == "windows" {
		got := splitPath(`C:\tools\versions\2.0\bin`)
		if len(got) != 5 || got[0] != "C:" || got[4] != "bin" {
			t.Errorf("windows splitPath = %v, want [C: tools versions 2.0 bin]", got)
		}
		if v, ok := toolchainVersion(`C:\Users\u\.rustup\toolchains\stable\bin\cargo`); !ok || v != "stable" {
			t.Errorf("windows toolchainVersion = %q, %v; want stable", v, ok)
		}
		return
	}
	// unix：反斜杠保留在段内（合法文件名字符）。
	// 注意 strings.Split 在路径前导 / 处产生空段：["", home, u, tools, a\b, versions, 2.0]
	got := splitPath(`/home/u/tools/a\b/versions/2.0`)
	if got[4] != `a\b` || got[5] != "versions" {
		t.Errorf("unix splitPath must keep backslash inside segment: %v", got)
	}
	// unix：反斜杠不参与 "toolchains" 段匹配
	if _, ok := toolchainVersion(`/home/u/.rustup/toolchains\stable/bin/cargo`); ok {
		t.Error("unix toolchainVersion must not split on backslash")
	}
}

// TestSubKind 验证子项精确分类：日志类子项不继承父项 cache 类型。
func TestSubKind(t *testing.T) {
	cases := []struct{ parent, name, want string }{
		{"cache", "_logs", "logs"},       // npm 调试日志
		{"cache", "debug-0.log", "logs"}, // *.log 文件
		{"cache", "npm-debug.log", "logs"},
		{"cache", "_cacache", "cache"}, // 其余继承父项
		{"cache", "_npx", "cache"},
		{"cache", "_locks", "cache"}, // 不靠猜测扩大分类
		{"cache", "anonymous-cli-metrics.json", "cache"},
		{"download", "_logs", "logs"}, // 日志识别与父项类型无关
		{"cache", "catalog", "cache"}, // 不能把 catalog 误判为日志
	}
	for _, c := range cases {
		if got := subKind(c.parent, c.name); got != c.want {
			t.Errorf("subKind(%q, %q) = %q, want %q", c.parent, c.name, got, c.want)
		}
	}
}

// TestVersionedMatchSkipsNvmLayout 验证 nvm 布局不得被误判为 versioned
// installer：~/.nvm/versions/<家族名>/<版本>/bin/<cmd> 的 "versions" 段后是
// node/iojs 等家族名而非版本号，误判会把 node/npm/npx 全部归为名为 ".nvm"
// 的工具、nodejs 家族合并失效、当前版本被推断为 "node"。
func TestVersionedMatchSkipsNvmLayout(t *testing.T) {
	if _, v, ok := versionedMatch("/Users/x/.nvm/versions/node/v22.14.0/bin/node"); ok {
		t.Errorf("nvm 布局被误判为 versioned installer: v=%q", v)
	}
	if _, v, ok := versionedMatch(`C:\Users\x\.nvm\versions\node\v22.14.0\bin\node`); ok {
		t.Errorf("nvm 布局（Windows 反斜杠）被误判: v=%q", v)
	}
	// 标准 versioned installer（claude/mavis）仍应命中
	if _, v, ok := versionedMatch("/Users/x/.claude/versions/0.2.5/bin/claude"); !ok || v != "0.2.5" {
		t.Errorf("标准 versioned 布局应命中，got v=%q ok=%v", v, ok)
	}
	if _, v, ok := versionedMatch("/Users/x/.mavis/versions/v1.2.3/bin/mavis"); !ok || v != "v1.2.3" {
		t.Errorf("v 前缀版本目录应命中，got v=%q ok=%v", v, ok)
	}
}

// TestBrewCellarPrefixTrailingSlash 验证 HOMEBREW_PREFIX 带尾斜杠时
// Cellar 前缀不产生双斜杠（手拼 pre+"/Cellar/" 会让 brew 归因全部失效）。
// 期望值用 filepath.Join 构造：Windows 上分隔符为反斜杠且 Join 会把
// "/opt/homebrew" 规范化为 "\opt\homebrew"，与实现同构才能比较。
func TestBrewCellarPrefixTrailingSlash(t *testing.T) {
	want := filepath.Join("/opt", "homebrew", "Cellar") + string(filepath.Separator)
	if got := brewCellarPrefix("/opt/homebrew"); got != want {
		t.Errorf("无尾斜杠前缀: got %q want %q", got, want)
	}
	if got := brewCellarPrefix("/opt/homebrew/"); got != want {
		t.Errorf("带尾斜杠前缀应规范化: got %q want %q", got, want)
	}
	// 真实路径匹配验证：前缀后跟公式名应命中
	real := brewCellarPrefix("/opt/homebrew/") + filepath.Join("python", "3.13", "bin", "python3")
	if !strings.HasPrefix(real, brewCellarPrefix("/opt/homebrew")) {
		t.Error("带尾斜杠 vs 无尾斜杠的前缀应一致匹配")
	}
}

// TestEnvOverridablePaths 验证自定义环境变量（PYENV_ROOT/RUSTUP_HOME/
// CARGO_HOME）下的安装路径被识别——此前硬编码 ~/.pyenv、~/.rustup、
// ~/.cargo，用户自定义根目录时版本目录/工具链归因全部失效。
func TestEnvOverridablePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pyenvRoot := t.TempDir()
	t.Setenv("PYENV_ROOT", pyenvRoot)
	if got := pyenvVersionsPath(); got != filepath.Join(pyenvRoot, "versions") {
		t.Errorf("pyenvVersionsPath = %q, want %q（应识别 PYENV_ROOT）", got, filepath.Join(pyenvRoot, "versions"))
	}
	if got := pyenvShimsPath(); got != filepath.Join(pyenvRoot, "shims") {
		t.Errorf("pyenvShimsPath = %q, want %q", got, filepath.Join(pyenvRoot, "shims"))
	}

	rustupHome := t.TempDir()
	t.Setenv("RUSTUP_HOME", rustupHome)
	if got := rustupToolchainsPath(); got != filepath.Join(rustupHome, "toolchains") {
		t.Errorf("rustupToolchainsPath = %q, want %q（应识别 RUSTUP_HOME）", got, filepath.Join(rustupHome, "toolchains"))
	}

	cargoHome := t.TempDir()
	t.Setenv("CARGO_HOME", cargoHome)
	if got := cargoBin(); got != filepath.Join(cargoHome, "bin") {
		t.Errorf("cargoBin = %q, want %q（应识别 CARGO_HOME）", got, filepath.Join(cargoHome, "bin"))
	}
}

// TestClassifyGitkJoinsGitFamily 验证 gitk/git-gui（随 Git 分发的 GUI 附属）
// 归并进 git 家族：unix 上它们是可执行脚本（IsConsoleExe 恒真），此前不入
// gitFamily 表导致每个 git 安装都多出独立 GUI 工具行（噪音，且无卸载来源）。
func TestClassifyGitkJoinsGitFamily(t *testing.T) {
	for _, name := range []string{"gitk", "git-gui"} {
		c := classify("/usr/local/git/bin/"+name, name)
		if c.Family != "git" {
			t.Errorf("classify(%s).Family = %q, want git", name, c.Family)
		}
	}
}

// TestClassifyRustupInstaller 验证 ~/.cargo/bin/rustup 归 InstRustup 而非
// InstCargo：InstCargo 不推导工具链清理项，rustup 的旧工具链展示会缺失。
func TestClassifyRustupInstaller(t *testing.T) {
	c := classify(filepath.Join(cargoBin(), "rustup"), "rustup")
	if c.Installer != InstRustup {
		t.Errorf("rustup installer = %q, want rustup（此前归 cargo，工具链细分丢失）", c.Installer)
	}
	// cargo 本身仍是 cargo 来源
	c2 := classify(filepath.Join(cargoBin(), "cargo"), "cargo")
	if c2.Installer != InstCargo {
		t.Errorf("cargo installer = %q, want cargo", c2.Installer)
	}
}
