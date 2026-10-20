package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAppBundleVersionReadsPlist 校验从 .app 包读出版本号。
func TestAppBundleVersionReadsPlist(t *testing.T) {
	if _, err := os.Stat("/usr/bin/plutil"); err != nil {
		t.Skip("macOS only")
	}
	app := filepath.Join(t.TempDir(), "Fake.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleShortVersionString</key>
  <string>5.7.7</string>
  <key>CFBundleVersion</key>
  <string>57000700</string>
</dict>
</plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AppBundleVersion(app); got != "5.7.7" {
		t.Errorf("version = %q, want 5.7.7 (CFBundleShortVersionString wins)", got)
	}
}

// TestAppBundleVersionFallsBack 校验缺短版本号时回退 CFBundleVersion。
func TestAppBundleVersionFallsBack(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Fake.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
  <key>CFBundleVersion</key><string>1.0.2</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := AppBundleVersion(app); got != "1.0.2" {
		t.Errorf("version = %q, want 1.0.2", got)
	}
}

// TestAppBundleVersionMissing 校验读不到时返回空而不是报错。
func TestAppBundleVersionMissing(t *testing.T) {
	if got := AppBundleVersion(filepath.Join(t.TempDir(), "Nope.app")); got != "" {
		t.Errorf("version = %q, want empty", got)
	}
}
