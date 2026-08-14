package scanner

import (
	"encoding/json"
	"os"
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
// （目录内存在 node / node.exe），避免把 ~/.local/bin 等共享 bin 目录整体
// 算作 nodejs 的安装占用。Windows 官方安装器 / nvm-windows / Volta / scoop
// 的目录都满足；unix 上独立散放的 npm 脚本不满足（返回 ""，按文件计大小）。
func nodejsInstallRoot(dir string) string {
	if dir == "" {
		return ""
	}
	for _, cand := range []string{"node.exe", "node"} {
		if st, err := os.Stat(filepath.Join(dir, cand)); err == nil && !st.IsDir() {
			return dir
		}
	}
	return ""
}

// ---- Git 家族 ----

// gitFamily 是随 Git 分发的命令：Git for Windows 的 cmd/ 目录（git.exe、
// git-lfs.exe、scalar.exe、tig.exe、git-receive-pack.exe、git-upload-pack.exe、
// start-ssh-agent.cmd、start-ssh-pageant.cmd 等）与 unix 安装中的同名命令。
// 它们共享同一安装，应归并为一条 "git"。brew 公式 git-lfs / tig 等是独立
// 产品，已在 classify 的 brew 分支保持独立（家族检查在其后）。gh / hub 是
// 独立 CLI 产品，不属于家族。
var gitFamily = map[string]bool{
	"git": true, "git-lfs": true, "scalar": true, "tig": true,
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

// gitInstallRoot 仅当目录本身就是 Git 命令目录（含 git.exe / git）时返回
// 该目录作为安装根（Git for Windows 的 cmd/ 即此形态），避免把共享 bin
// 目录整体算作 git 的安装占用。unix 上 git 由 brew/系统提供（已在更早分支
// 命中），独立散放的 git 脚本不满足（返回 ""，按文件计大小）。
func gitInstallRoot(dir string) string {
	if dir == "" {
		return ""
	}
	for _, cand := range []string{"git.exe", "git"} {
		if st, err := os.Stat(filepath.Join(dir, cand)); err == nil && !st.IsDir() {
			return dir
		}
	}
	return ""
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
	marker := pre + "/Cellar/"
	if !strings.HasPrefix(real, marker) {
		return "", "", false
	}
	segs := strings.Split(strings.TrimPrefix(real, marker), "/")
	if len(segs) < 2 {
		return "", "", false
	}
	return segs[0], segs[1], true
}

// versionedMatch finds .../<base>/versions/<v>/ in a real path.
// 分隔符无关：Windows 反斜杠路径与 unix 正斜杠路径同样识别（拆分后按
// 正斜杠重建，filepath.FromSlash 还原为本机分隔符）。
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
		if v == "" {
			continue
		}
		return base, v, true
	}
	return "", "", false
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
	// 其余形态：bin 名与 shim 名匹配（含作用域包）
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
				if pkgHasBin(pkgDir, shimName) {
					return pkgDir, true
				}
			}
			continue
		}
		pkgDir := filepath.Join(nm, e.Name())
		if pkgHasBin(pkgDir, shimName) {
			return pkgDir, true
		}
	}
	return "", false
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

func cargoBin() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".cargo", "bin")
	}
	return filepath.Join(".cargo", "bin")
}

func pyenvVersionsPath() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, ".pyenv", "versions")
	}
	return filepath.Join(".pyenv", "versions")
}

func pyenvShimsPath() string {
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
	if real == dir {
		return true
	}
	return strings.HasPrefix(real, dir+string(filepath.Separator))
}
