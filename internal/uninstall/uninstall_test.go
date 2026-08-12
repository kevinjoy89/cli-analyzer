package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// fakeRoots 用环境变量把平台数据根重定向到临时目录，使残留检测可确定性测试。
func fakeRoots(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	for k, v := range map[string]string{
		"HOME":            home,
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
		{scanner.InstPipx, "uv", "pipx uninstall uv", true},
		{scanner.InstCargo, "sd", "cargo uninstall sd", true},
		{scanner.InstGo, "x", "", false}, // go 来源命令包含 bin 名，单独断言
		{scanner.InstVersioned, "claude", "", false},
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
		if off.Command != c.wantCmd || off.Runnable != c.runnable {
			t.Errorf("%s %s: command=%q runnable=%v, want %q/%v", c.installer, c.name, off.Command, off.Runnable, c.wantCmd, c.runnable)
		}
	}
}

func TestResiduesRulesSource(t *testing.T) {
	fakeRoots(t)
	home := os.Getenv("HOME")
	cfgDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mytool")
	dotDir := filepath.Join(home, ".mytool")
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
	mkdir(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mytool"))
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
	cfgDir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "mytool")
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

// 空结果必须是非 nil 切片：JSON 序列化为 [] 而非 null（前端 null.length 会崩）。
func TestResiduesEmptyIsNotNil(t *testing.T) {
	fakeRoots(t)
	res := Residues("no-such-tool-xyz", nil)
	if res == nil {
		t.Fatal("Residues() returned nil slice for empty result")
	}
}

// ResolveCommand 应在（增强的）PATH 目录中找到命令：模拟 GUI 最小 PATH，
// 命令只存在于 ~/.local/bin（增强目录）。
func TestResolveCommandViaAugmentedPath(t *testing.T) {
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	mkdir(t, localBin)
	fake := filepath.Join(localBin, "faketool-un")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	resolved, err := ResolveCommand("faketool-un")
	if err != nil {
		t.Fatalf("ResolveCommand: %v", err)
	}
	if resolved != fake {
		t.Errorf("resolved = %q, want %q", resolved, fake)
	}
	if _, err := ResolveCommand("no-such-cmd-xyz"); err == nil {
		t.Error("missing command should error")
	}
}
