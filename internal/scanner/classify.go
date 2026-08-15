package scanner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// classification is the outcome of resolving one PATH executable.
type classification struct {
	ToolID         string // canonical identity (formula/pkg/binary name)
	Installer      Installer
	CurrentVersion string // for versioned / brew: version of this binary
	InstallRoot    string // install dir to size (Cellar/<f>, versions/, node_modules/<pkg>…)
	// Family 是工具家族合并的根名（如 "nodejs"），仅当该二进制因家族合并
	// 归并进聚合工具时非空；brew 公式 node 等保持独立身份的工具为空。
	Family string
}

// classify maps a resolved real path to a tool identity. Match order matters
// (most specific first); entryName is the base name as seen on PATH.
func classify(real, entryName string) classification {
	if real == "" {
		if id, ok := nodejsFamilyRoot(entryName); ok {
			return classification{ToolID: id, Installer: InstNodejs, Family: id}
		}
		return classification{ToolID: entryName, Installer: InstOther}
	}

	// pyenv: shims and builds resolve under ~/.pyenv (shims/ or versions/).
	if p := pyenvVersionsPath(); under(real, p) {
		return classification{ToolID: "pyenv", Installer: InstPyenv, InstallRoot: p}
	}
	if p := pyenvShimsPath(); under(real, p) {
		return classification{ToolID: "pyenv", Installer: InstPyenv, InstallRoot: pyenvVersionsPath()}
	}

	// brew: /<prefix>/Cellar/<formula>/<ver>/...
	// 注意：brew 公式 node 保持原名（Cellar/node 已把 node/npm/npx/corepack
	// 合并为一条），改名会破坏 "brew uninstall node" 的公式名。
	if f, ver, ok := brewCellarMatch(real); ok {
		root := filepath.Join(brewPrefix(), "Cellar", f)
		return classification{ToolID: f, Installer: InstBrew, CurrentVersion: ver, InstallRoot: root}
	}

	// versioned installers: .../<base>/versions/<v>/<bin> (claude, mavis…).
	if base, v, ok := versionedMatch(real); ok {
		return classification{
			ToolID: filepath.Base(base), Installer: InstVersioned,
			CurrentVersion: v, InstallRoot: filepath.Join(base, "versions"),
		}
	}

	// npm global packages: .../node_modules/<pkg>/…
	if pkg, root, ok := npmPkgMatch(real); ok {
		// Node.js 运行时家族（nvm/fnm 布局里 node/npm/npx/corepack 是独立的
		// node_modules 包）→ 合并为一条 "nodejs"。
		if id, ok := nodejsFamilyRoot(pkg); ok {
			return classification{ToolID: id, Installer: InstNodejs, InstallRoot: root, Family: id}
		}
		return classification{ToolID: pkg, Installer: InstNpm, InstallRoot: root}
	}

	// pipx: ~/.local/pipx/venvs/<pkg>/
	if pkg, root, ok := pipxMatch(real); ok {
		return classification{ToolID: pkg, Installer: InstPipx, InstallRoot: root}
	}

	// go install: $GOPATH/bin or $GOBIN.
	if under(real, goBin()) {
		return classification{ToolID: entryName, Installer: InstGo}
	}

	// rustup 本体：~/.cargo/bin/rustup。单独识别为 InstRustup（而非
	// InstCargo）以推导工具链清理项（旧工具链 USER 展示），否则 rustup
	// 的工具链细分在扫描中完全缺失。
	if under(real, cargoBin()) && strings.EqualFold(filepath.Base(real), "rustup") {
		return classification{ToolID: "rustup", Installer: InstRustup}
	}

	// cargo: ~/.cargo/bin.
	if under(real, cargoBin()) {
		return classification{ToolID: entryName, Installer: InstCargo}
	}

	// Node.js 运行时家族兜底：Windows 官方安装器 / nvm-windows / Volta / scoop
	// 把 node.exe、npm.cmd、npx.cmd、corepack.cmd 放在同一目录，每个都是
	// InstOther 独立工具 → 合并为一条 "nodejs"。unix 上独立放置的 node/npm
	// 同理。brew/nvm 等更具体的来源已在上面命中，不会走到这里。
	if id, ok := nodejsFamilyRoot(entryName); ok {
		return classification{
			ToolID: id, Installer: InstNodejs, Family: id,
			InstallRoot: nodejsInstallRoot(filepath.Dir(real)),
		}
	}

	// npm 全局 shim（Windows）：npm -g 在 %APPDATA%\npm 根部生成 <name>.cmd
	// shim，包本体在 node_modules/<pkg>。shim 是真实工具的入口（pi →
	// @earendil-works/pi-coding-agent、opencode → opencode-ai），应归为对应
	// 的 npm 包工具而非 nodejs 的附属。仅 .cmd/.bat shim 形态触发（unix 上
	// npm 全局是符号链接，EvalSymlinks 后由 npmPkgMatch 处理）。
	lowName := strings.ToLower(entryName)
	if strings.HasSuffix(lowName, ".cmd") || strings.HasSuffix(lowName, ".bat") {
		if shim := strings.TrimSuffix(strings.TrimSuffix(lowName, ".cmd"), ".bat"); shim != "" {
			if pkgDir, ok := npmGlobalShim(filepath.Dir(real), shim); ok {
				return classification{ToolID: shim, Installer: InstNpm, InstallRoot: pkgDir}
			}
		}
	}

	// Git 家族兜底：Git for Windows 的 cmd/ 目录把 git.exe、git-lfs.exe、
	// scalar.exe、tig.exe、start-ssh-agent.cmd 等放在同一安装；各自独立成行
	// 会造成"git 被拆成多个"的噪音 → 合并为一条 "git"。brew 公式 git-lfs /
	// tig 等已在更早的 brew 分支保持独立公式身份；gitk/git-gui 是 GUI 应用
	// （被 IsConsoleExe 排除），不属于家族。
	if id, ok := gitFamilyRoot(entryName); ok {
		return classification{
			ToolID: id, Installer: InstOther, Family: id,
			InstallRoot: gitInstallRoot(filepath.Dir(real)),
		}
	}

	return classification{ToolID: entryName, Installer: InstOther}
}

// ---- Node.js runtime family ----

// nodejsFamily 是随 Node.js 运行时分发的命令。它们共享同一安装目录
// （Windows）或同属一个 node_modules 布局（unix），应归并为一条 "nodejs"。
// install_tools.bat / nodevars.bat 是官方安装器自带的辅助脚本（同在安装
// 目录内），独立成行是噪音，一并归并。yarn/pnpm 等独立分发的包管理器
// 不属于该家族，保持独立工具。
var nodejsFamily = map[string]bool{
	"node": true, "npm": true, "npx": true, "corepack": true, "node-gyp": true,
	"install_tools": true, "nodevars": true,
}

// normEntryName 把 PATH 入口名归一化用于家族匹配：Windows 入口带扩展名
// （node.exe / npm.cmd），剥掉后与家族表比较（大小写不敏感）。
func normEntryName(name string) string {
	n := strings.ToLower(name)
	for _, ext := range []string{".exe", ".cmd", ".bat", ".com"} {
		n = strings.TrimSuffix(n, ext)
	}
	return n
}

// nodejsFamilyRoot 报告 entryName 是否属于 Node.js 运行时家族；命中时返回
// 家族根名（"nodejs"）。
func nodejsFamilyRoot(entryName string) (string, bool) {
	if nodejsFamily[normEntryName(entryName)] {
		return "nodejs", true
	}
	return "", false
}

// nodejsInstallRoot 仅当目录本身就是 Node 运行时目录时返回该目录作为安装根
// （目录内存在 node / node.exe，且不含 node 家族以外的命令），避免把
// /usr/local/bin、~/.local/bin 等共享 bin 目录整体算作 nodejs 的安装占用
// （手动放置的 node 只是单个二进制，目录里其余命令不属于 node 运行时）。
// Windows 官方安装器 / nvm-windows / Volta / scoop 的目录都满足；unix 上
// 独立散放的 npm 脚本不满足（返回 ""，按文件计大小）。
func nodejsInstallRoot(dir string) string {
	if dir == "" {
		return ""
	}
	hasNode := false
	for _, cand := range []string{"node.exe", "node"} {
		if st, err := os.Stat(filepath.Join(dir, cand)); err == nil && !st.IsDir() {
			hasNode = true
			break
		}
	}
	if !hasNode {
		return ""
	}
	if !nodeOnlyDir(dir) {
		return ""
	}
	return dir
}

// nodeOnlyDir 报告目录内除 node 家族命令外无其他命令（专用运行时目录形态）。
// Windows 官方安装目录附带 npm.ps1/npx.ps1/corepack.ps1、node_etw_provider.man
// 等清单与文档文件，这些不是其他工具的命令，必须放行：unix 按执行位判定
// （无执行位的文档/数据文件放行），Windows 按"命令必须带可执行扩展名
// （.exe/.cmd/.bat/.com/.ps1）"判定（.man/.dll/无扩展名等一律放行）。
func nodeOnlyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch normEntryName(e.Name()) {
		case "", "node", "npm", "npx", "corepack", "node-gyp", "install_tools", "nodevars",
			"npm.ps1", "npx.ps1", "corepack.ps1", "node-gyp.ps1":
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if runtime.GOOS == "windows" {
			// Windows 命令必须带可执行扩展名：.exe/.cmd/.bat/.com/.ps1 之外的
			// 文件（.man/.dll/.dat/无扩展名等）是数据/文档——官方安装器目录里
			// 存在 node_etw_provider.man 等清单文件，把它们当"其他命令"会把
			// 真正的 Node 运行时目录误拒（安装根归因失效）。
			switch ext {
			case ".exe", ".cmd", ".bat", ".com", ".ps1":
				return false // 其他工具的命令（不在 node 家族白名单内）
			default:
				continue
			}
		} else if info.Mode()&0o111 == 0 {
			continue // 无执行位 = 文档/数据文件，不是命令
		}
		switch ext {
		case ".md", ".txt", ".html", ".json", ".xml", ".yml", ".yaml", ".cfg", ".ini":
			continue // 文档/配置文件不是命令
		}
		return false
	}
	return true
}

// ---- Git 家族 ----

// gitFamily 是随 Git 分发的命令：Git for Windows 的 cmd/ 目录（git.exe、
// git-lfs.exe、scalar.exe、tig.exe、git-receive-pack.exe、git-upload-pack.exe、
// start-ssh-agent.cmd、start-ssh-pageant.cmd 等）与 unix 安装中的同名命令。
// 它们共享同一安装，应归并为一条 "git"。brew 公式 git-lfs / tig 等是独立
// 产品，已在 classify 的 brew 分支保持独立（家族检查在其后）。gh / hub 是
// 独立 CLI 产品，不属于家族。gitk/git-gui 是随 Git 分发的 GUI 附属：unix 上
// 是可执行脚本（IsConsoleExe 恒真），不并入家族会各自成为独立工具行。
var gitFamily = map[string]bool{
	"git": true, "git-lfs": true, "scalar": true, "tig": true,
	"gitk": true, "git-gui": true,
	"git-receive-pack": true, "git-upload-pack": true, "git-upload-archive": true,
	"start-ssh-agent": true, "start-ssh-pageant": true,
}

// gitFamilyRoot 报告 entryName 是否属于 Git 家族；命中时返回家族根名（"git"）。
func gitFamilyRoot(entryName string) (string, bool) {
	if gitFamily[normEntryName(entryName)] {
		return "git", true
	}
	return "", false
}

// gitInstallRoot 仅当目录本身就是 Git 命令目录（含 git.exe / git，且不含
// Git 家族以外的命令）时返回该目录作为安装根（Git for Windows 的 cmd/
// 即此形态），避免把 /usr/local/bin 等共享 bin 目录整体算作 git 的安装
// 占用。unix 上 git 由 brew/系统提供（已在更早分支命中），独立散放的 git
// 脚本不满足（返回 ""，按文件计大小）。
func gitInstallRoot(dir string) string {
	if dir == "" {
		return ""
	}
	hasGit := false
	for _, cand := range []string{"git.exe", "git"} {
		if st, err := os.Stat(filepath.Join(dir, cand)); err == nil && !st.IsDir() {
			hasGit = true
			break
		}
	}
	if !hasGit {
		return ""
	}
	if !gitOnlyDir(dir) {
		return ""
	}
	return dir
}

// gitOnlyDir 报告目录内除 git 家族命令外无其他命令（专用命令目录形态）。
// gitk/git-gui/git-citool 等 GUI 附属与 git-bash/git-cmd 等启动器随 Git 官方
// 安装器分发（git-bash/git-cmd 不在 gitFamily，也不应并入家族——它们是
// 启动器而非 git 命令），目录内存在它们不算"其他工具"；文档/数据文件
// （README/LICENSE 等）同样放行（unix 按执行位、Windows 按扩展名）。
func gitOnlyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch normEntryName(e.Name()) {
		case "", "gitk", "git-gui", "git-citool", "git-web--browse", "git-mergetool", "git-difftool",
			"git-bash", "git-cmd":
			continue
		}
		if gitFamily[normEntryName(e.Name())] {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if runtime.GOOS == "windows" {
			// Windows 命令必须带可执行扩展名：其余（.dll/.md/无扩展名等）是
			// 数据/文档，不当"其他命令"，避免把真实 Git for Windows 的 cmd/
			// 目录误拒（安装根归因失效）。
			switch ext {
			case ".exe", ".cmd", ".bat", ".com", ".ps1":
				return false // 其他工具的命令（不在 git 家族白名单内）
			default:
				continue
			}
		} else if info.Mode()&0o111 == 0 {
			continue // 无执行位 = 文档/数据文件，不是命令
		}
		switch ext {
		case ".md", ".txt", ".html", ".json", ".xml", ".yml", ".yaml", ".cfg", ".ini":
			continue // 文档/配置文件不是命令
		}
		return false
	}
	return true
}

// probeOrder 把工具的主二进制排到 Binaries[0]：版本探测（--version）取
// Binaries[0]；nodejs 合并工具的主二进制是 node（而非 corepack/npm/npx），
// git 合并工具的主二进制是 git（而非 git-lfs/tig/start-ssh-agent）。
func probeOrder(tb *toolBuilder) {
	if len(tb.binaries) < 2 {
		return
	}
	preferred := ""
	switch tb.name {
	case "nodejs":
		preferred = "node"
	case "git":
		preferred = "git"
	}
	if preferred == "" {
		return
	}
	sort.SliceStable(tb.binaries, func(i, j int) bool {
		return normEntryName(tb.binaries[i].Name) == preferred &&
			normEntryName(tb.binaries[j].Name) != preferred
	})
}

// ---- install-root helpers ----

var brewPrefixVal string
var brewPrefixOnce sync.Once

func brewPrefix() string {
	brewPrefixOnce.Do(func() {
		if p := os.Getenv("HOMEBREW_PREFIX"); p != "" {
			brewPrefixVal = p
			return
		}
		for _, p := range []string{"/opt/homebrew", "/usr/local", "/home/linuxbrew/.linuxbrew"} {
			if st, err := os.Stat(filepath.Join(p, "Cellar")); err == nil && st.IsDir() {
				brewPrefixVal = p
				return
			}
		}
	})
	return brewPrefixVal
}

func brewCellarMatch(real string) (formula, ver string, ok bool) {
	pre := brewPrefix()
	if pre == "" {
		return "", "", false
	}
	marker := brewCellarPrefix(pre)
	if !strings.HasPrefix(real, marker) {
		return "", "", false
	}
	segs := strings.Split(strings.TrimPrefix(real, marker), "/")
	if len(segs) < 2 {
		return "", "", false
	}
	return segs[0], segs[1], true
}

// brewCellarPrefix 返回 Cellar 前缀（含尾分隔符）。
// 用 filepath.Join 规范化：用户设置 HOMEBREW_PREFIX 带尾斜杠时
// 手拼 pre+"/Cellar/" 会产生双斜杠，真实路径不匹配导致 brew 归因失效。
func brewCellarPrefix(pre string) string {
	return filepath.Join(pre, "Cellar") + string(filepath.Separator)
}

// versionedMatch finds .../<base>/versions/<v>/ in a real path.
// 分隔符无关：Windows 反斜杠路径与 unix 正斜杠路径同样识别（拆分后按
// 正斜杠重建，filepath.FromSlash 还原为本机分隔符）。
// versions 后的段必须是版本形态（数字或 v+数字开头）：nvm 的布局是
// ~/.nvm/versions/<家族名 node/iojs>/<版本>/bin/<cmd>，"versions" 后是
// 家族名而非版本号，误判会把 node/npm/npx 归为名为 ".nvm" 的工具并使
// nodejs 家族合并失效。
func versionedMatch(real string) (base, v string, ok bool) {
	segs := splitPath(real)
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] != "versions" {
			continue
		}
		base = filepath.FromSlash(strings.Join(segs[:i], "/"))
		if base == "" || base == "/" {
			continue
		}
		v = segs[i+1]
		if !isVersionDir(v) {
			continue // nvm 布局（versions 后是 node/iojs 等家族名）
		}
		return base, v, true
	}
	return "", "", false
}

// isVersionDir 报告目录名是否为版本目录形态（数字开头或 v+数字开头）。
func isVersionDir(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	if c >= '0' && c <= '9' {
		return true
	}
	return (c == 'v' || c == 'V') && len(name) > 1 && name[1] >= '0' && name[1] <= '9'
}

// npmPkgMatch returns the package name, its node_modules root and the pkg dir.
func npmPkgMatch(real string) (pkg, pkgDir string, ok bool) {
	segs := splitPath(real)
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] != "node_modules" {
			continue
		}
		root := filepath.FromSlash(strings.Join(segs[:i+1], "/"))
		if i+2 < len(segs) && strings.HasPrefix(segs[i+1], "@") {
			pkg = segs[i+1] + "/" + segs[i+2]
			return pkg, filepath.Join(root, pkg), true
		}
		if i+1 < len(segs) && segs[i+1] != "" {
			pkg = segs[i+1]
			return pkg, filepath.Join(root, pkg), true
		}
	}
	return "", "", false
}

// npmGlobalShim 解析 npm 全局前缀（Windows 的 %APPDATA%\npm 等）根部的
// <name>.cmd shim：shim 是 node_modules 中某个包的 bin 入口，包本体才是
// 真正的工具安装。先查普通包 node_modules/<name>，再扫描全部包（普通包与
// 作用域包 node_modules/@<scope>/<pkg>）按 package.json 的 bin 字段匹配
// shim 名（覆盖 bin 名 ≠ 包名的场景，如 opencode-ai 的 bin 名为 opencode）。
// 返回包目录（安装根）；无法解析时 ok=false（保持原 InstOther 行为）。
func npmGlobalShim(dir, shimName string) (pkgDir string, ok bool) {
	nm := filepath.Join(dir, "node_modules")
	if st, err := os.Stat(nm); err != nil || !st.IsDir() {
		return "", false
	}
	// 普通包同名：node_modules/<name>（最常见形态）
	if st, err := os.Stat(filepath.Join(nm, shimName)); err == nil && st.IsDir() {
		return filepath.Join(nm, shimName), true
	}
	// 其余形态：bin 名与 shim 名匹配（含作用域包）。要求包名（最后一段）
	// 以 shim 名为前缀——corepack 这类包会声明 yarn/pnpm/npx 等代理 bin，
	// 包名与 shim 名无关，不能据此把 shim 归到代理包。
	entries, err := os.ReadDir(nm)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "@") {
			scopes, err := os.ReadDir(filepath.Join(nm, e.Name()))
			if err != nil {
				continue
			}
			for _, p := range scopes {
				if !p.IsDir() {
					continue
				}
				pkgDir := filepath.Join(nm, e.Name(), p.Name())
				if shimOwnsPkg(pkgDir, shimName) {
					return pkgDir, true
				}
			}
			continue
		}
		pkgDir := filepath.Join(nm, e.Name())
		if shimOwnsPkg(pkgDir, shimName) {
			return pkgDir, true
		}
	}
	return "", false
}

// shimOwnsPkg 报告包目录是否为该 shim 名对应的真实工具：bin 字段命中
// 且包名最后一段以 shim 名为前缀（opencode-ai / pi-coding-agent 形态；
// corepack 的 yarn/pnpm 代理 bin 不满足前缀条件）。
func shimOwnsPkg(pkgDir, shimName string) bool {
	if !pkgHasBin(pkgDir, shimName) {
		return false
	}
	return strings.HasPrefix(strings.ToLower(filepath.Base(pkgDir)), strings.ToLower(shimName))
}

// pkgHasBin 报告包目录的 package.json 是否声明了名为 binName 的 bin。
// bin 为对象时查键；bin 为字符串时按 npm 约定 bin 名 = 包名最后一段。
func pkgHasBin(pkgDir, binName string) bool {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return false
	}
	var p struct {
		Bin json.RawMessage `json:"bin"`
	}
	if json.Unmarshal(data, &p) != nil {
		return false
	}
	var m map[string]string
	if json.Unmarshal(p.Bin, &m) == nil {
		for k := range m {
			if strings.EqualFold(k, binName) {
				return true
			}
		}
		return false
	}
	var s string
	if json.Unmarshal(p.Bin, &s) == nil && s != "" {
		return strings.EqualFold(filepath.Base(pkgDir), binName)
	}
	return false
}

// pipxMatch returns the venv package and its venv dir.
func pipxMatch(real string) (pkg, venvDir string, ok bool) {
	segs := splitPath(real)
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] == "pipx" && i+1 < len(segs) && segs[i+1] == "venvs" && i+2 < len(segs) {
			pkg = segs[i+2]
			return pkg, filepath.FromSlash(strings.Join(segs[:i+3], "/")), true
		}
	}
	return "", "", false
}

// splitPath 把路径拆成段：Windows 上反斜杠与正斜杠都是分隔符，unix 上
// 仅正斜杠（反斜杠是 unix 合法文件名字符，不能当分隔符切）。
func splitPath(p string) []string {
	if runtime.GOOS == "windows" {
		p = strings.ReplaceAll(p, "\\", "/")
	}
	return strings.Split(p, "/")
}

func goBin() string {
	if b := os.Getenv("GOBIN"); b != "" {
		return b
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, "go", "bin")
	}
	return filepath.Join("go", "bin")
}

var (
	goEnvOnce sync.Once
	goEnvVal  string
)

// goEnvGopath 返回真实 GOPATH：优先 $GOPATH，其次 `go env GOPATH`
// （用户可能在 ~/.config/go/env 里配置，Go 1.8+ 默认路径与 env 变量不同），
// 均不可用时回退 ~/go。go 命令首次调用后缓存。
func goEnvGopath() string {
	goEnvOnce.Do(func() {
		if g := os.Getenv("GOPATH"); g != "" {
			goEnvVal = filepath.Clean(g)
			return
		}
		if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
			if g := strings.TrimSpace(string(out)); g != "" {
				goEnvVal = filepath.Clean(g)
				return
			}
		}
		if h, err := os.UserHomeDir(); err == nil {
			goEnvVal = filepath.Join(h, "go")
		}
	})
	return goEnvVal
}

func cargoBin() string {
	// CARGO_HOME 可自定义（默认 ~/.cargo）
	if c := os.Getenv("CARGO_HOME"); c != "" {
		return filepath.Join(c, "bin")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".cargo", "bin")
	}
	return filepath.Join(".cargo", "bin")
}

func pyenvVersionsPath() string {
	// PYENV_ROOT 可自定义（默认 ~/.pyenv）
	if r := os.Getenv("PYENV_ROOT"); r != "" {
		return filepath.Join(r, "versions")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".pyenv", "versions")
	}
	return filepath.Join(".pyenv", "versions")
}

func pyenvShimsPath() string {
	if r := os.Getenv("PYENV_ROOT"); r != "" {
		return filepath.Join(r, "shims")
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".pyenv", "shims")
	}
	return filepath.Join(".pyenv", "shims")
}

// under reports whether real == dir or real is a strict descendant of dir.
func under(real, dir string) bool {
	if dir == "" {
		return false
	}
	if runtime.GOOS == "windows" {
		// Windows 文件系统大小写不敏感：GOPATH 等环境变量与 EvalSymlinks
		// 解析出的真实路径大小写可能不同，必须按大小写不敏感比较，
		// 否则 go/cargo/pyenv 归因失效（工具被归为 other）。
		return underFold(real, dir)
	}
	if real == dir {
		return true
	}
	return strings.HasPrefix(real, dir+string(filepath.Separator))
}

// underFold 是 Windows 专用的大小写不敏感路径包含判断（斜杠归一化后比较）。
func underFold(real, dir string) bool {
	if dir == "" {
		return false
	}
	rl := strings.ToLower(filepath.ToSlash(real))
	dl := strings.ToLower(filepath.ToSlash(dir))
	return rl == dl || strings.HasPrefix(rl, dl+"/")
}
