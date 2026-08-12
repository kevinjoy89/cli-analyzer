package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
)

// setupUninstallEnv 隔离平台根并写入 mock 扫描缓存，供 uninstall 子命令测试。
func setupUninstallEnv(t *testing.T, tools []scanner.Tool) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "xdg-cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	res := &scanner.ScanResult{Tools: tools, Totals: scanner.Totals{}}
	b, _ := json.Marshal(res)
	p := filepath.Join(platform.CacheRoot(), "last-scan.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunUninstallBlocked(t *testing.T) {
	pinZhCN(t)
	setupUninstallEnv(t, nil)
	captureStdout(t)
	errBuf := captureStderr(t)
	if code := Run([]string{"uninstall", "python"}); code != 2 {
		t.Fatalf("blocked exit = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "拒绝卸载") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

func TestRunUninstallUnknown(t *testing.T) {
	pinZhCN(t)
	setupUninstallEnv(t, nil)
	captureStdout(t)
	captureStderr(t)
	if code := Run([]string{"uninstall", "nosuchtool"}); code != 2 {
		t.Fatalf("unknown exit = %d, want 2", code)
	}
}

func TestRunUninstallResidue(t *testing.T) {
	pinZhCN(t)
	cfgDir := filepath.Join(t.TempDir(), "xdg-config", "mytool")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	setupUninstallEnv(t, []scanner.Tool{
		{Name: "mytool", Installer: "other", Binaries: []scanner.Binary{{Name: "mytool", Path: "/x/mytool", Real: "/x/mytool"}}},
	})
	// 让平台根指向同一临时目录（setupUninstallEnv 的 XDG_CONFIG_HOME 与 cfgDir 需一致）
	// —— 上面 cfgDir 用了独立 TempDir，这里改为通过环境对齐：
	// 简化：直接以 HOME 下 .mytool 作为残留（generic 规则含 Home/.name）
	home := os.Getenv("HOME")
	if err := os.MkdirAll(filepath.Join(home, ".mytool"), 0o755); err != nil {
		t.Fatal(err)
	}
	buf := captureStdout(t)
	if code := Run([]string{"uninstall", "mytool", "--residue"}); code != 0 {
		t.Fatalf("residue exit = %d, want 0", code)
	}
	out := buf.String()
	if !strings.Contains(out, ".mytool") {
		t.Errorf("residue list missing path: %q", out)
	}
	if !strings.Contains(out, "含登录凭证") {
		t.Errorf("user residue should be marked with credential note: %q", out)
	}
}

func TestRunUninstallJSON(t *testing.T) {
	pinZhCN(t)
	setupUninstallEnv(t, []scanner.Tool{
		{Name: "gh", Installer: "brew", Binaries: []scanner.Binary{{Name: "gh", Path: "/x/gh", Real: "/x/gh"}}},
	})
	buf := captureStdout(t)
	if code := Run([]string{"uninstall", "gh", "--json"}); code != 0 {
		t.Fatalf("json exit = %d, want 0", code)
	}
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, buf.String())
	}
	if parsed["officialCommand"] != "brew uninstall gh" || parsed["runnable"] != true {
		t.Errorf("parsed = %v", parsed)
	}
}
