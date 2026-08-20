//go:build windows

package cmdexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveCommandExtension 验证 Windows 上裸命令名补全 PATHEXT 扩展名：
// PATH 目录里只有 fakext.exe，传 "fakext" 也能解析到（静态表/代跑裸名场景）。
func TestResolveCommandExtension(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "fakext.exe")
	if err := os.WriteFile(exe, []byte("MZ fake exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	resolved, err := ResolveCommand("fakext")
	if err != nil {
		t.Fatalf("ResolveCommand(fakext): %v", err)
	}
	// Windows 文件系统大小写不敏感：解析结果扩展名按 PATHEXT 枚举的 case
	// 返回（runner 上 .EXE 大写），与磁盘存储的 .exe 仅 case 不同。
	if !strings.EqualFold(resolved, exe) {
		t.Errorf("resolved = %q, want %q", resolved, exe)
	}
	if _, err := ResolveCommand("no-such-fakext-xyz"); err == nil {
		t.Error("missing command should error")
	}
}
