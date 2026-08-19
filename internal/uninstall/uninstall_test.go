package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// fakeRoots 用环境变量把平台数据根重定向到临时目录，使残留检测可确定性测试。
func fakeRoots(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	for k, v := range map[string]string{
		"HOME":            home,
		"USERPROFILE":     home, // Windows：os.UserHomeDir 优先 USERPROFILE
		"XDG_DATA_HOME":   filepath.Join(home, "xdg-data"),
		"XDG_CONFIG_HOME": filepath.Join(home, "xdg-config"),
		"XDG_CACHE_HOME":  filepath.Join(home, "xdg-cache"),
		"XDG_STATE_HOME":  filepath.Join(home, "xdg-state"),
		"LOCALAPPDATA":    filepath.Join(home, "local-appdata"),
		"APPDATA":         filepath.Join(home, "appdata"),
	} {
		t.Setenv(k, v)
	}
}

// configRoot 返回当前平台生效的 config 数据根：unix 用 XDG config，
// Windows 用 %APPDATA%（generic 规则在 Windows 上映射到 AppData）。
func configRoot() string {
	if r := platform.Root(platform.XDGConfig); r != "" {
		return r
	}
	return platform.Root(platform.AppData)
}

func mkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestIsBlocked(t *testing.T) {
	for _, name := range []string{"python", "python3", "node", "npm", "git", "docker", "go", "brew", "bash", "zsh", "cli-analyzer", "CLI-Analyzer"} {
		if !IsBlocked(name) {
			t.Errorf("IsBlocked(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"claude", "kimi", "uv", "gh", "opencode", "pipx"} {
		if IsBlocked(name) {
			t.Errorf("IsBlocked(%q) = true, want false", name)
		}
	}
	// Windows 扩展名变体同样命中黑名单
	for _, ex := range []string{"ssh.exe", "SSH.EXE", "cmd.exe", "powershell.exe", "wsl.exe"} {
		if !IsBlocked(ex) {
			t.Errorf("IsBlocked(%q) = false, want true (extension variant)", ex)
		}
	}
}

func TestOfficialCommand(t *testing.T) {
	cases := []struct {
		installer scanner.Installer
		name      string
		wantCmd   string
		runnable  bool
	}{
		{scanner.InstBrew, "gh", "brew uninstall gh", true},
		{scanner.InstNpm, "opencode", "npm uninstall -g opencode", true},
		{scanner.InstNpm, "pi", "npm uninstall -g @earendil-works/pi-coding-agent", true}, // npmToolID 映射短名 → 真实包名
		{scanner.InstPipx, "uv", "pipx uninstall uv", true},
		{scanner.InstCargo, "sd", "cargo uninstall sd", true},
		{scanner.InstGo, "x", "", false}, // go 来源命令包含 bin 名，单独断言
		{scanner.InstVersioned, "claude", "", false},
		{scanner.InstLocalBin, "uv", "", true}, // 命令含 bin 目录，单独断言
		{scanner.InstOther, "kimi", "", false},
	}
	for _, c := range cases {
		off := OfficialCommand(c.installer, c.name, "binx")
		if c.installer == scanner.InstGo {
			if !strings.Contains(off.Command, "binx") || off.Runnable {
				t.Errorf("go: command=%q runnable=%v", off.Command, off.Runnable)
			}
			continue
		}
		if c.installer == scanner.InstLocalBin {
			// unix：可代跑 rm；Windows 无 ~/.local/bin（LocalBinDir 为空）→
			// 回退为提示命令（不可代跑）
			if platform.LocalBinDir() != "" {
				if !strings.Contains(off.Command, "binx") || !off.Runnable || off.Bin != "rm" {
					t.Errorf("local-bin: command=%q bin=%q runnable=%v", off.Command, off.Bin, off.Runnable)
				}
			} else if !strings.Contains(off.Command, "binx") || off.Runnable {
				t.Errorf("local-bin (windows): command=%q runnable=%v", off.Command, off.Runnable)
			}
			continue
		}
		if off.Command != c.wantCmd || off.Runnable != c.runnable {
			t.Errorf("%s %s: command=%q runnable=%v, want %q/%v", c.installer, c.name, off.Command, off.Runnable, c.wantCmd, c.runnable)
		}
	}
}

func TestResiduesRulesSource(t *testing.T) {
	fakeRoots(t)
	cfgDir := filepath.Join(configRoot(), "mytool")
	dotDir := filepath.Join(platform.HomeDir(), ".mytool")
	mkdir(t, cfgDir) // 存在 → 应检出
	mkdir(t, dotDir) // 存在 → 应检出

	res := Residues("mytool", nil) // 无快照：纯规则表源
	paths := map[string]bool{}
	for _, r := range res {
		paths[r.Path] = true
	}
	if !paths[cfgDir] || !paths[dotDir] {
		t.Errorf("residues missing expected dirs, got %v", paths)
	}
	// tier 标注：config 源 → user
	for _, r := range res {
		if r.Path == cfgDir && r.Tier != "user" {
			t.Errorf("config residue tier = %q, want user", r.Tier)
		}
	}
}

func TestResiduesExcludesMissing(t *testing.T) {
	fakeRoots(t)
	mkdir(t, filepath.Join(configRoot(), "mytool"))
	// 其他源目录不存在 → 不应出现在残留中
	res := Residues("mytool", nil)
	for _, r := range res {
		if _, err := os.Stat(r.Path); err != nil {
			t.Errorf("residue %q does not exist", r.Path)
		}
	}
}

func TestResiduesSnapshotSource(t *testing.T) {
	fakeRoots(t)
	// 快照归因目录（规则表没有的自定义位置）
	extra := filepath.Join(os.Getenv("HOME"), "custom", "mytool-data")
	mkdir(t, extra)
	snap := &scanner.ScanResult{Tools: []scanner.Tool{
		{Name: "mytool", DataDirs: []scanner.DataDir{{Path: extra, Tier: scanner.TierUser, Kind: "data"}}},
	}}
	res := Residues("mytool", snap)
	found := false
	for _, r := range res {
		if r.Path == extra {
			found = true
		}
	}
	if !found {
		t.Errorf("snapshot dir %q not detected as residue", extra)
	}
}

func TestTrashResiduesMovesToTrash(t *testing.T) {
	fakeRoots(t)
	// 回收站根与待删目录同文件系统（都在临时区）
	cfgDir := filepath.Join(configRoot(), "mytool")
	mkdir(t, cfgDir)
	res := Residues("mytool", nil)
	if len(res) == 0 {
		t.Fatal("no residue detected")
	}
	deleted, errs := TrashResidues(res, "mytool")
	if len(errs) > 0 {
		t.Fatalf("TrashResidues errors: %v", errs)
	}
	if len(deleted) != len(res) {
		t.Errorf("deleted %d, want %d", len(deleted), len(res))
	}
	for _, p := range deleted {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("residue %q still exists after trash", p)
		}
	}
	// 回收站里有对应项目（可恢复）
	items, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Error("trash is empty after TrashResidues")
	}
}

// TestRemoveResiduesPermanently 验证永久删除变体直接删除且不经过内置回收站：
// 残留处置默认走 TrashResidues（可恢复），永久删除是用户的显式选择。
func TestRemoveResiduesPermanently(t *testing.T) {
	fakeRoots(t)
	cfgDir := filepath.Join(configRoot(), "mytool")
	mkdir(t, cfgDir)
	res := Residues("mytool", nil)
	if len(res) == 0 {
		t.Fatal("no residue detected")
	}
	deleted, errs := RemoveResidues(res, "mytool")
	if len(errs) > 0 {
		t.Fatalf("RemoveResidues errors: %v", errs)
	}
	if len(deleted) != len(res) {
		t.Errorf("deleted %d, want %d", len(deleted), len(res))
	}
	for _, p := range deleted {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("residue %q still exists after permanent delete", p)
		}
	}
	items, err := trash.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("trash should be empty after permanent delete, got %d items", len(items))
	}
}

// 空结果必须是非 nil 切片：JSON 序列化为 [] 而非 null（前端 null.length 会崩）。
func TestResiduesEmptyIsNotNil(t *testing.T) {
	fakeRoots(t)
	res := Residues("no-such-tool-xyz", nil)
	if res == nil {
		t.Fatal("Residues() returned nil slice for empty result")
	}
}

// TestOfficialCommandGoNoBinName 验证 go 来源工具无 PATH 二进制（缓存种子）
// 时卸载提示不是残缺命令（rm $(go env GOPATH)/bin/ 无文件名）。
func TestOfficialCommandGoNoBinName(t *testing.T) {
	off := OfficialCommand(scanner.InstGo, "mytool", "")
	if off.Command == "rm $(go env GOPATH)/bin/" {
		t.Errorf("binName 为空时命令残缺: %q", off.Command)
	}
	if !strings.Contains(off.Command, "<命令名>") {
		t.Errorf("应给通用占位提示，got %q", off.Command)
	}
}

// TestOfficialForNpmPackage 验证 OfficialFor 按 Tool.Package（真实包名）寻址：
// 映射短名（pi）卸载到 scoped 包；撞名真实短名包（真包 "pi"）卸载到原包。
// 这是 code review #3 修复的碰撞回归。
func TestOfficialForNpmPackage(t *testing.T) {
	// 映射短名 + 扫描记录的真实包名
	off := OfficialFor(scanner.Tool{Name: "pi", Installer: string(scanner.InstNpm), Package: "@earendil-works/pi-coding-agent"})
	if off.Command != "npm uninstall -g @earendil-works/pi-coding-agent" || !off.Runnable {
		t.Errorf("mapped: %+v", off)
	}
	// 撞名真实短名包：Package 确切为 "pi"，不得被逆映射改写
	off2 := OfficialFor(scanner.Tool{Name: "pi", Installer: string(scanner.InstNpm), Package: "pi"})
	if off2.Command != "npm uninstall -g pi" {
		t.Errorf("real short pkg: %+v", off2)
	}
}
