package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipIfNotDarwin 跳过非 macOS 平台：AppBundleVersion 的 .app/Info.plist
// 解析是 darwin 专有实现，其他平台是返回空串的桩（见 appbundle_other.go）。
// 用 runtime.GOOS 而不是「plutil 是否存在」判断——后者在装了 plutil 的
// Linux 容器里会误判。
func skipIfNotDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("AppBundleVersion is macOS-only")
	}
}

// writeFakeApp 构造一个带 Info.plist 的假 .app 包。
func writeFakeApp(t *testing.T, plist string) string {
	t.Helper()
	app := filepath.Join(t.TempDir(), "Fake.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	return app
}

// TestAppBundleVersionReadsPlist 校验从 .app 包读出版本号。
func TestAppBundleVersionReadsPlist(t *testing.T) {
	skipIfNotDarwin(t)
	app := writeFakeApp(t, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleShortVersionString</key>
  <string>5.7.7</string>
  <key>CFBundleVersion</key>
  <string>57000700</string>
</dict>
</plist>`)
	if got := AppBundleVersion(app); got != "5.7.7" {
		t.Errorf("version = %q, want 5.7.7 (CFBundleShortVersionString wins)", got)
	}
}

// TestAppBundleVersionFallsBack 校验缺短版本号时回退 CFBundleVersion。
func TestAppBundleVersionFallsBack(t *testing.T) {
	skipIfNotDarwin(t)
	app := writeFakeApp(t, `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
  <key>CFBundleVersion</key><string>1.0.2</string>
</dict></plist>`)
	if got := AppBundleVersion(app); got != "1.0.2" {
		t.Errorf("version = %q, want 1.0.2", got)
	}
}

// TestAppBundleVersionMissing 校验读不到时返回空而不是报错。
// 这个用例跨平台成立：非 darwin 的桩实现同样返回空串。
func TestAppBundleVersionMissing(t *testing.T) {
	if got := AppBundleVersion(filepath.Join(t.TempDir(), "Nope.app")); got != "" {
		t.Errorf("version = %q, want empty", got)
	}
}
