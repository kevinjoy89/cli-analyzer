package cmdexec

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestWithPathCaseInsensitive 验证 WithPath 对 PATH 环境变量的替换是
// 大小写不敏感的：Windows 上系统环境变量名为 "Path"（而非 "PATH"），
// 只匹配 "PATH=" 前缀会保留旧变量并追加新变量，子进程拿到重复 PATH。
// GUI 与 CLI 代跑命令共用此函数（行为一致）。
func TestWithPathCaseInsensitive(t *testing.T) {
	env := WithPath([]string{"Path=/old/one", "HOME=/x"}, "/new/path")
	pathCount := 0
	var last string
	for _, e := range env {
		upper := strings.ToUpper(e)
		if strings.HasPrefix(upper, "PATH=") {
			pathCount++
			last = e
		}
	}
	if pathCount != 1 {
		t.Errorf("PATH-like entries = %d, want exactly 1 (case-insensitive replacement); env=%v", pathCount, env)
	}
	if last != "PATH=/new/path" {
		t.Errorf("last PATH entry = %q, want %q", last, "PATH=/new/path")
	}
	foundHome := false
	for _, e := range env {
		if e == "HOME=/x" {
			foundHome = true
		}
	}
	if !foundHome {
		t.Errorf("HOME=/x lost after WithPath: %v", env)
	}
}

// TestResolveCommandViaAugmentedPath 验证 ResolveCommand 在（增强的）PATH
// 目录中找到命令：模拟 GUI 最小 PATH，命令只存在于 ~/.local/bin（增强目录）。
// 仅 unix：Windows 的 PATH 增强是无操作（path_augment_windows.go）。
func TestResolveCommandViaAugmentedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("user-dir PATH augmentation is unix-only")
	}
	home := t.TempDir()
	localBin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(localBin, 0o755); err != nil {
		t.Fatal(err)
	}
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

// TestWrapShim 验证 .cmd/.bat 包装为 cmd.exe /c、其余原样（Windows 代跑
// 与 -h 探测共用）。
func TestWrapShim(t *testing.T) {
	if bin, args := WrapShim("npm.cmd", []string{"-g", "pkg"}); bin != "cmd.exe" ||
		len(args) != 4 || args[0] != "/c" || args[1] != "npm.cmd" || args[3] != "pkg" {
		t.Errorf("WrapShim(npm.cmd) = %s %v", bin, args)
	}
	if bin, args := WrapShim("tool.bat", []string{"x"}); bin != "cmd.exe" || len(args) != 3 {
		t.Errorf("WrapShim(tool.bat) = %s %v", bin, args)
	}
	if bin, args := WrapShim("claude.exe", []string{"update"}); bin != "claude.exe" || len(args) != 1 {
		t.Errorf("WrapShim(exe) = %s %v, want unchanged", bin, args)
	}
	if bin, args := WrapShim("claude", []string{"update"}); bin != "claude" || len(args) != 1 {
		t.Errorf("WrapShim(bare) = %s %v, want unchanged", bin, args)
	}
}
