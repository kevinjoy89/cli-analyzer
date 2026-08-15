package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/trash"
)

// TestUninstallResidueErrorNoLiteralPlaceholder 验证残留清理失败时的错误输出
// 不含未替换的 {msg} 占位符（此前 i18n.T("cli.errors") 漏传 msg 参数，
// 字典值 "错误: {msg}" 的占位符原样输出到用户界面）。
func TestUninstallResidueErrorNoLiteralPlaceholder(t *testing.T) {
	pinZhCN(t)
	// 隔离扫描缓存与配置根（避免读写真实用户数据）
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	// 构造残留目录（规则表 GenericDataDirs 会命中 ~/.config/testtool）
	residueDir := filepath.Join(cfgHome, "testtool")
	if err := os.MkdirAll(residueDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 让 Trash 必然失败：回收站根指向只读目录
	ro := t.TempDir()
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	origRoot := trash.Root
	trash.Root = func() string { return filepath.Join(ro, "trash") }
	t.Cleanup(func() { trash.Root = origRoot })

	// 写扫描缓存：testtool 作为已知工具
	res := &scanner.ScanResult{
		ScannedAt: "2026-08-15T00:00:00+08:00",
		Tools: []scanner.Tool{{
			Name: "testtool", Installer: "other",
			DataDirs: []scanner.DataDir{{Path: residueDir, Tier: scanner.TierUser, Kind: "data"}},
		}},
	}
	if err := scanner.SaveCache(res); err != nil {
		t.Fatal(err)
	}
	oldStdin := stdin
	stdin = strings.NewReader("")
	t.Cleanup(func() { stdin = oldStdin })
	captureStdout(t)
	errBuf := captureStderr(t)

	code := runUninstall([]string{"testtool", "--yes"})
	if code == 0 {
		t.Log("code=0（残留可能被静默跳过），继续断言输出")
	}
	out := errBuf.String()
	if strings.Contains(out, "{msg}") {
		t.Errorf("错误输出包含未替换占位符 {msg}: %q", out)
	}
}
