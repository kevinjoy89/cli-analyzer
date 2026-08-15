package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cli-analyzer/internal/uninstall"
)

// TestRunOfficialFindsCommandInAugmentedPath 验证代跑卸载命令在最小 PATH
// 环境下也能通过增强目录找到命令（与 GUI 行为一致）：HOME 下的 ~/.local/bin
// 不在 PATH 中，裸 exec "fake-uv" 会报 executable file not found。
func TestRunOfficialFindsCommandInAugmentedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only augment 目录场景")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(binDir, "fake-uv")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho FAKE_UNINSTALL_OK\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "/usr/bin:/bin") // 最小 PATH：不含 ~/.local/bin

	buf := captureStdout(t)
	off := uninstall.Official{Bin: "fake-uv", Runnable: true}
	if err := runOfficial(off); err != nil {
		t.Fatalf("runOfficial 失败（增强 PATH 未生效）: %v", err)
	}
	if !strings.Contains(buf.String(), "FAKE_UNINSTALL_OK") {
		t.Errorf("脚本输出缺失: %q", buf.String())
	}
}
