package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// classification is the outcome of resolving one PATH executable.
type classification struct {
	ToolID         string // canonical identity (formula/pkg/binary name)
	Installer      Installer
	CurrentVersion string // for versioned / brew: version of this binary
	InstallRoot    string // install dir to size (Cellar/<f>, versions/, node_modules/<pkg>…)
}

// classify maps a resolved real path to a tool identity. Match order matters
// (most specific first); entryName is the base name as seen on PATH.
func classify(real, entryName string) classification {
	if real == "" {
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

	return classification{ToolID: entryName, Installer: InstOther}
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
