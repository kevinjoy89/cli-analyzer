package probe

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestFillVersionsProbesWhenOnlyDesktopVersionKnown 校验双形态工具在
// Version 已被桌面版本填上时，仍然会去探测 CLI 版本。
//
// 回归：此前跳过条件是 Version != ""，而双形态工具的 Version 先被桌面版本
// 兜底填上，于是永远不探测 CLI 版本（agy / kimi 就是这么丢了 CLI 版本）。
func TestFillVersionsProbesWhenOnlyDesktopVersionKnown(t *testing.T) {
	setupCache(t)

	bin := filepath.Join(t.TempDir(), "dual")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(bin)

	probed := 0
	origProbe := probeVersionFn
	probeVersionFn = func(string) (string, bool) { probed++; return "2.1.0", true }
	t.Cleanup(func() { probeVersionFn = origProbe })

	tools := []scanner.Tool{{
		Name: "kimi", Installer: string(scanner.InstBrew),
		Version:        "1.0.2", // 桌面版本已兜底
		DesktopVersion: "1.0.2",
		Binaries:       []scanner.Binary{{Name: "kimi", Path: bin, Real: bin, Size: st.Size()}},
	}}

	FillVersions(tools, 0)

	if probed == 0 {
		t.Fatal("should still probe: only the desktop version was known")
	}
	if tools[0].CLIVersion != "2.1.0" {
		t.Errorf("cliVersion = %q, want 2.1.0", tools[0].CLIVersion)
	}
	// Version 保留桌面版本（不覆盖已有值）
	if tools[0].Version != "1.0.2" {
		t.Errorf("version = %q, want desktop value kept", tools[0].Version)
	}
}

// TestFillVersionsSkipsWhenCLIVersionKnown 校验 CLI 版本已知时不再探测。
func TestFillVersionsSkipsWhenCLIVersionKnown(t *testing.T) {
	setupCache(t)
	bin := filepath.Join(t.TempDir(), "known")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(bin)

	probed := 0
	origProbe := probeVersionFn
	probeVersionFn = func(string) (string, bool) { probed++; return "", false }
	t.Cleanup(func() { probeVersionFn = origProbe })

	tools := []scanner.Tool{{
		Name: "x", Installer: string(scanner.InstBrew),
		Version: "1.0.0", CLIVersion: "1.0.0",
		Binaries: []scanner.Binary{{Name: "x", Path: bin, Real: bin, Size: st.Size()}},
	}}
	FillVersions(tools, 0)
	if probed != 0 {
		t.Errorf("should not probe when cliVersion is known, probed %d times", probed)
	}
}
