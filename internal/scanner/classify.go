package scanner

import (
	"os"
	"path/filepath"
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

	return classification{ToolID: entryName, Installer: InstOther}
}

// ---- Node.js runtime family ----

// nodejsFamily 是随 Node.js 运行时分发的命令。它们共享同一安装目录
// （Windows）或同属一个 node_modules 布局（unix），应归并为一条 "nodejs"。
// yarn/pnpm 等独立分发的包管理器不属于该家族，保持独立工具。
var nodejsFamily = map[string]bool{
	"node": true, "npm": true, "npx": true, "corepack": true, "node-gyp": true,
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

// probeOrder 把工具的主二进制排到 Binaries[0]：版本探测（--version）取
// Binaries[0]，nodejs 合并工具的主二进制是 node（而非 corepack/npm/npx）。
func probeOrder(tb *toolBuilder) {
	if tb.name != "nodejs" || len(tb.binaries) < 2 {
		return
	}
	sort.SliceStable(tb.binaries, func(i, j int) bool {
		return normEntryName(tb.binaries[i].Name) == "node" &&
			normEntryName(tb.binaries[j].Name) != "node"
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
func versionedMatch(real string) (base, v string, ok bool) {
	segs := strings.Split(real, "/")
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] != "versions" {
			continue
		}
		base = strings.Join(segs[:i], "/")
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
	segs := strings.Split(real, "/")
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] != "node_modules" {
			continue
		}
		root := strings.Join(segs[:i+1], "/")
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

// pipxMatch returns the venv package and its venv dir.
func pipxMatch(real string) (pkg, venvDir string, ok bool) {
	segs := strings.Split(real, "/")
	for i := 0; i+1 < len(segs); i++ {
		if segs[i] == "pipx" && i+1 < len(segs) && segs[i+1] == "venvs" && i+2 < len(segs) {
			pkg = segs[i+2]
			return pkg, strings.Join(segs[:i+3], "/"), true
		}
	}
	return "", "", false
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
