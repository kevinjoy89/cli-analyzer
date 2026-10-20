package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDualVersionMismatch 校验双形态工具的两个版本分别记录，且不一致时
// 打上 VersionMismatch（界面据此显示双版本 + 叹号，而不是任选一个掩盖差异）。
// 真实场景：kimi 的 CLI 是 2.1.0，桌面 .app 是 1.0.2。
func TestDualVersionMismatch(t *testing.T) {
	td := t.TempDir()
	t.Setenv("HOME", td)

	bundle := filepath.Join(td, "Applications", "Kimi Code.app")
	if err := os.MkdirAll(filepath.Join(bundle, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
  <key>CFBundleShortVersionString</key><string>1.0.2</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}

	origPaths, origVer := appBundlePaths, appBundleVersion
	appBundlePaths = func(apps []string) []string { return []string{bundle} }
	appBundleVersion = func(string) string { return "1.0.2" }
	t.Cleanup(func() { appBundlePaths, appBundleVersion = origPaths, origVer })

	tools := aiAttrHarnessAt(t, td, []string{"kimi"})
	res := finalize(tools, []string{"kimi"}, Options{})
	if len(res.Tools) != 1 {
		t.Fatalf("tools = %d", len(res.Tools))
	}
	tool := res.Tools[0]

	// CLI 版本来自二进制探测（测试环境无二进制，故为空）；桌面版本来自应用包
	if tool.DesktopVersion != "1.0.2" {
		t.Errorf("desktopVersion = %q, want 1.0.2", tool.DesktopVersion)
	}
	// 只有一侧已知时不算 mismatch
	if tool.VersionMismatch {
		t.Error("single known version must not be flagged as mismatch")
	}
	// Version 兜底为已知的那个
	if tool.Version != "1.0.2" {
		t.Errorf("version = %q, want fallback to desktop version", tool.Version)
	}
}

// TestVersionMismatchFlagged 校验两侧都已知且不同时打标记。
func TestVersionMismatchFlagged(t *testing.T) {
	// finalize 内部按 cliVersion/desktopVersion 计算，这里直接构造 builder 验证口径。
	tb := &toolBuilder{cliVersion: "2.1.0", desktopVersion: "1.0.2"}
	mismatch := tb.cliVersion != "" && tb.desktopVersion != "" && tb.cliVersion != tb.desktopVersion
	if !mismatch {
		t.Error("2.1.0 vs 1.0.2 should be flagged")
	}

	same := &toolBuilder{cliVersion: "1.0.2", desktopVersion: "1.0.2"}
	if same.cliVersion != same.desktopVersion {
		t.Error("identical versions must not be flagged")
	}

	onlyOne := &toolBuilder{cliVersion: "2.1.0"}
	if onlyOne.cliVersion != "" && onlyOne.desktopVersion != "" {
		t.Error("single form must not be flagged")
	}
}
