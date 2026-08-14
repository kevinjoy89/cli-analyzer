package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRootXDGEnv(t *testing.T) {
	// XDG 根是 unix 语义：Windows 上 rootFor(XDG*) 恒为空（数据根是
	// %APPDATA%/%LOCALAPPDATA%），该用例仅对 unix 有意义。
	if runtime.GOOS == "windows" {
		t.Skip("XDG roots do not apply on Windows")
	}
	// XDG env vars must take precedence over defaults.
	td := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(td, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(td, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(td, "config"))
	if got := Root(XDGCache); got != filepath.Join(td, "cache") {
		t.Errorf("XDGCache = %q", got)
	}
	if got := Root(XDGData); got != filepath.Join(td, "data") {
		t.Errorf("XDGData = %q", got)
	}
	if got := Root(XDGConfig); got != filepath.Join(td, "config") {
		t.Errorf("XDGConfig = %q", got)
	}
}

func TestHome(t *testing.T) {
	if HomeDir() == "" {
		t.Error("HomeDir() should not be empty in a test environment")
	}
}

func TestPathDirsDedup(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows：PATH 用 ; 分隔，系统目录按 %WINDIR% 判定
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		sys := filepath.Join(windir, "System32")
		t.Setenv("PATH", `D:\a;D:\a;D:\b;`+sys)
		dirs := PathDirs(true)
		if !contains(dirs, `D:\a`) || !contains(dirs, `D:\b`) || contains(dirs, sys) {
			t.Errorf("PathDirs = %v, want D:\\a and D:\\b present, System32 absent", dirs)
		}
		if !strings.Contains(strings.Join(PathDirs(false), ";"), sys) {
			t.Error("skipSystem=false should include System32")
		}
		return
	}
	t.Setenv("PATH", "/a:/a:/b:/usr/bin")
	dirs := PathDirs(true)
	// "/a" deduped, "/usr/bin" skipped, "/b" kept（另有用户目录增强项）。
	if !contains(dirs, "/a") || !contains(dirs, "/b") || contains(dirs, "/usr/bin") {
		t.Errorf("PathDirs = %v, want /a and /b present, /usr/bin absent", dirs)
	}
	if !strings.Contains(strings.Join(PathDirs(false), " "), "/usr/bin") {
		t.Error("skipSystem=false should include /usr/bin")
	}
}

func contains(dirs []string, want string) bool {
	for _, d := range dirs {
		if d == want {
			return true
		}
	}
	return false
}

func TestRootsNoEmpty(t *testing.T) {
	roots := Roots()
	if _, ok := roots[Home]; !ok {
		t.Error("Roots should always include Home")
	}
}

// GUI（最小 PATH）场景：增强应把已知用户二进制目录补进来。
// 仅 unix：Windows 的 augmentUserDirs 是无操作（PATH 已覆盖安装位置）。
func TestPathDirsAugmentUserDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("user-dir PATH augmentation is unix-only")
	}
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	dirs := PathDirs(true)
	if !contains(dirs, localBin) {
		t.Errorf("PathDirs missing augmented %q, got %v", localBin, dirs)
	}
	// 不存在的目录不应被加入
	if contains(dirs, filepath.Join(home, ".cargo", "bin")) {
		t.Error("non-existent dir should be skipped")
	}
}
