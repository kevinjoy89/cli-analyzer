package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/disk"
)

// isolateXDGRoots 把所有 XDG 根指向 base 下的临时子目录并隔离 HOME。
// 不能只依赖 HOME 回退：runner/shell 可能预置 XDG_*_HOME 环境变量，
// xdgOr 优先 env，会导致测试遍历真实数据根。
// Windows 上 XDG 根不适用（rootFor 返回 ""），孤儿遍历改走
// %APPDATA%/%LOCALAPPDATA%——一并隔离，避免触达真实用户目录。
func isolateXDGRoots(t *testing.T, base string) (cacheRoot, dataRoot, cfgRoot string) {
	t.Helper()
	cacheRoot = filepath.Join(base, "cache")
	dataRoot = filepath.Join(base, "data")
	cfgRoot = filepath.Join(base, "config")
	for _, r := range []string{cacheRoot, dataRoot, cfgRoot} {
		if err := os.MkdirAll(r, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("XDG_DATA_HOME", dataRoot)
	t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	t.Setenv("HOME", base)
	t.Setenv("APPDATA", filepath.Join(base, "appdata"))
	t.Setenv("LOCALAPPDATA", filepath.Join(base, "localappdata"))
	for _, r := range []string{filepath.Join(base, "appdata"), filepath.Join(base, "localappdata")} {
		if err := os.MkdirAll(r, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return
}

// orphanRoot 返回当前平台被 findUnattributed 实际遍历、且已隔离到 base 的
// 数据根：unix 用 XDG config，Windows 用 %APPDATA%（并隔离 LOCALAPPDATA）。
func orphanRoot(t *testing.T, base string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		appdata := filepath.Join(base, "appdata")
		localappdata := filepath.Join(base, "localappdata")
		for _, r := range []string{appdata, localappdata} {
			if err := os.MkdirAll(r, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("APPDATA", appdata)
		t.Setenv("LOCALAPPDATA", localappdata)
		return appdata
	}
	_, _, cfg := isolateXDGRoots(t, base)
	return cfg
}

// skipOnWindows 标记 XDG 根语义用例：findUnattributed 的 XDG 根在 Windows
// 不适用（rootFor 返回 ""），这些用例验证的是 unix 布局；Windows 孤儿语义
// 由 platform 层测试（systemdirs/vendorexclusion/installedapp）覆盖。
func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("XDG 数据根在 Windows 不适用；Windows 孤儿语义由 platform 层测试覆盖")
	}
}

// TestFindUnattributedFilter verifies the non-CLI exclusion system inside
// findUnattributed: self dirs, structural GUI signals, vendor table.
func TestFindUnattributedFilter(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	cacheRoot := filepath.Join(base, "cache")
	dataRoot := filepath.Join(base, "data")
	cfgRoot := filepath.Join(base, "config")
	for _, r := range []string{cacheRoot, dataRoot, cfgRoot} {
		if err := os.MkdirAll(r, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("XDG_DATA_HOME", dataRoot)
	t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	// macOS root 走 HOME，隔离避免读到真实目录
	t.Setenv("HOME", base)

	mkdir := func(p string) {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "payload.bin"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 真孤儿：应保留
	mkdir(filepath.Join(dataRoot, "dead-cli-tool"))
	// 被工具认领：应排除
	mkdir(filepath.Join(cacheRoot, "claimed-tool"))
	// 本应用自身：应排除
	mkdir(filepath.Join(dataRoot, "cli-analyzer"))
	// 结构性 GUI 信号：应排除
	mkdir(filepath.Join(cfgRoot, "com.apple.Safari"))
	mkdir(filepath.Join(dataRoot, "Some.App_12345678"))
	mkdir(filepath.Join(cacheRoot, "packages"))
	// 厂商排除表：应排除
	mkdir(filepath.Join(dataRoot, "Netsarang Computer"))
	// macOS Application Support（GUI 主导，非孤儿来源）：应排除——即使 HOME
	// 下有该目录也不应被遍历
	mkdir(filepath.Join(base, "Library", "Application Support", "App Store"))
	mkdir(filepath.Join(base, "Library", "Application Support", "Safari"))

	tools := map[string]*toolBuilder{"claimed-tool": {name: "claimed-tool", aliases: map[string]bool{}}}
	order := []string{"claimed-tool"}

	got := findUnattributed(tools, order, &disk.Sizer{})
	if len(got) != 1 {
		t.Fatalf("expected 1 orphan dir, got %d: %+v", len(got), got)
	}
	if got[0].Path != filepath.Join(dataRoot, "dead-cli-tool") {
		t.Errorf("orphan path = %q, want dead-cli-tool", got[0].Path)
	}
	if got[0].Bytes <= 0 {
		t.Errorf("orphan bytes should be sized, got %d", got[0].Bytes)
	}
	if got[0].Root != "xdg-data" {
		t.Errorf("orphan root = %q, want xdg-data", got[0].Root)
	}
	if got[0].Tier != TierUser {
		t.Errorf("orphan tier = %q, want user", got[0].Tier)
	}
}

// TestFindUnattributedSkipsFilesAndSymlinks 验证非目录条目不列为孤儿：
// .DS_Store 等普通文件、指向文件的符号链接跳过；指向目录的符号链接
// 仍是合法孤儿候选。
func TestFindUnattributedSkipsFilesAndSymlinks(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	_, _, cfgRoot := isolateXDGRoots(t, base)

	// .DS_Store 文件（macOS 常见）
	if err := os.WriteFile(filepath.Join(cfgRoot, ".DS_Store"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	// 指向文件的符号链接
	targetFile := filepath.Join(base, "target-file")
	if err := os.WriteFile(targetFile, make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetFile, filepath.Join(cfgRoot, "link-to-file")); err != nil {
		t.Fatal(err)
	}
	// 指向目录的符号链接 + 真目录：均应为孤儿候选
	targetDir := filepath.Join(base, "target-dir")
	if err := os.MkdirAll(filepath.Join(targetDir, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "inner", "x"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, filepath.Join(cfgRoot, "link-to-dir")); err != nil {
		t.Fatal(err)
	}
	realDir := filepath.Join(cfgRoot, "dead-cli-tool")
	if err := os.MkdirAll(filepath.Join(realDir, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "inner", "x"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	got := findUnattributed(map[string]*toolBuilder{}, nil, &disk.Sizer{})
	if len(got) != 2 {
		t.Fatalf("expected 2 orphan dirs (real + symlink-to-dir), got %d: %+v", len(got), got)
	}
}

// TestFindUnattributedCaseInsensitiveClaim 验证认领匹配大小写不敏感：
// Windows 上 PATH 名（claude）与数据目录名（Claude）大小写不同时，
// 目录仍被工具认领，不列为孤儿。
func TestFindUnattributedCaseInsensitiveClaim(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	_, dataRoot, _ := isolateXDGRoots(t, base)
	if err := os.MkdirAll(filepath.Join(dataRoot, "Claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "Claude", "config.json"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	tools := map[string]*toolBuilder{"claude": {name: "claude", aliases: map[string]bool{}}}
	got := findUnattributed(tools, []string{"claude"}, &disk.Sizer{})
	if len(got) != 0 {
		t.Fatalf("claimed dir with different case must not be orphan: %+v", got)
	}
}

// TestFindUnattributedClaimsDataDirs 验证 dataDirs 顶层目录名与工具名不同
// 时（opencode 的 ~/.config/oh-my-opencode）仍被认领，不列为孤儿。
func TestFindUnattributedClaimsDataDirs(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	_, _, cfgRoot := isolateXDGRoots(t, base)
	if err := os.MkdirAll(filepath.Join(cfgRoot, "oh-my-opencode", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgRoot, "oh-my-opencode", "inner", "x"), make([]byte, 1024), 0o644); err != nil {
		t.Fatal(err)
	}

	tools := map[string]*toolBuilder{"opencode": {name: "opencode", aliases: map[string]bool{},
		dataDirs: []DataDir{{Path: filepath.Join(cfgRoot, "oh-my-opencode")}}}}
	got := findUnattributed(tools, []string{"opencode"}, &disk.Sizer{})
	if len(got) != 0 {
		t.Fatalf("dataDir with different top-level name must not be orphan: %+v", got)
	}
}

// TestFindUnattributedSystemAndVendorDirs 验证系统/共享结构目录与新扩充的
// 排除表条目不列为孤儿（.mono、configstore、man、iterm2、raycast 等）。
func TestFindUnattributedSystemAndVendorDirs(t *testing.T) {
	skipOnWindows(t)
	base := t.TempDir()
	_, dataRoot, cfgRoot := isolateXDGRoots(t, base)

	mk := func(p string) {
		if err := os.MkdirAll(filepath.Join(p, "inner"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "inner", "x"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 共享基础设施：应排除
	mk(filepath.Join(cfgRoot, ".mono"))
	mk(filepath.Join(cfgRoot, "configstore"))
	mk(filepath.Join(cfgRoot, "simple-update-notifier"))
	mk(filepath.Join(dataRoot, "IsolatedStorage"))
	mk(filepath.Join(dataRoot, "man"))
	// 新扩充的 GUI 产品：应排除
	mk(filepath.Join(cfgRoot, "iterm2"))
	mk(filepath.Join(cfgRoot, "raycast"))
	mk(filepath.Join(cfgRoot, "joplin-desktop"))
	mk(filepath.Join(dataRoot, "karabiner"))
	mk(filepath.Join(dataRoot, "Axure"))
	// 真孤儿：应保留
	mk(filepath.Join(cfgRoot, "dead-cli-tool"))

	got := findUnattributed(map[string]*toolBuilder{}, nil, &disk.Sizer{})
	if len(got) != 1 {
		t.Fatalf("expected 1 orphan dir, got %d: %+v", len(got), got)
	}
	if got[0].Path != filepath.Join(cfgRoot, "dead-cli-tool") {
		t.Errorf("orphan = %q, want dead-cli-tool", got[0].Path)
	}
}

// TestFindUnattributedUpdaterDirs 验证 <App>-updater 结构规则：GUI 应用
// 自动更新暂存目录（tabby-updater/termius-updater 等）不列为孤儿。
// 跨平台运行：unix 走 XDG config 根，Windows 走隔离的 %APPDATA%。
func TestFindUnattributedUpdaterDirs(t *testing.T) {
	base := t.TempDir()
	root := orphanRoot(t, base)
	mk := func(name string) {
		if err := os.MkdirAll(filepath.Join(root, name, "inner"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "inner", "x"), make([]byte, 1024), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("tabby-updater")
	mk("termius-updater")
	mk("dead-cli-tool")

	got := findUnattributed(map[string]*toolBuilder{}, nil, &disk.Sizer{})
	if len(got) != 1 || filepath.Base(got[0].Path) != "dead-cli-tool" {
		t.Fatalf("expected only dead-cli-tool orphan, got %d: %+v", len(got), got)
	}
}

// TestIsSelfDataDir 验证自身目录判定：cli-analyzer 恒为自身；运行中
// 可执行文件基名（含/不含扩展名）也视为自身（Wails 产品名 CLI Analyzer
// 的 exe 出现在数据根顶层时不算孤儿）。
func TestIsSelfDataDir(t *testing.T) {
	if !isSelfDataDir("cli-analyzer") {
		t.Error("cli-analyzer must be self data dir")
	}
	// 应用产品名形态（便携包 exe 名为 cli-analyzer.exe 时也命中）
	if !isSelfDataDir("CLI Analyzer") {
		t.Error("product name must be self data dir")
	}
	if !isSelfDataDir("CLI Analyzer.exe") {
		t.Error("product exe name must be self data dir")
	}
	if isSelfDataDir("dead-cli-tool") {
		t.Error("unrelated dir must not be self")
	}
	full, base := selfExeNames()
	if full == "" {
		return // os.Executable 不可用时跳过自身名断言
	}
	if !isSelfDataDir(full) {
		t.Errorf("running exe name %q must be self data dir", full)
	}
	if base != "" && !isSelfDataDir(base) {
		t.Errorf("running exe base %q must be self data dir", base)
	}
}
