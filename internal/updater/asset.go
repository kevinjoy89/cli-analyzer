package updater

import (
	"fmt"
	"strings"
)

// AssetName 按发布流程的命名约定（见 design D4 / release.yml）构造产物名：
//
//	darwin/arm64 → CLI-Analyzer-<v>-macos-arm64.dmg
//	darwin/amd64 → CLI-Analyzer-<v>-macos-amd64.dmg
//	windows/installer → CLI-Analyzer-<v>-windows-amd64-installer.exe
//	windows/portable  → CLI-Analyzer-<v>-windows-amd64-portable.zip
//	linux/deb         → CLI-Analyzer-<v>-linux-amd64.deb
//	linux/tarball     → CLI-Analyzer-<v>-linux-amd64.tar.gz
//
// version 需为 "0.3.0" 形式（无 v 前缀）。
func AssetName(version, goos, goarch, installSource string) string {
	v := strings.TrimPrefix(version, "v")
	base := "CLI-Analyzer-" + v
	switch goos {
	case "darwin":
		return base + "-macos-" + goarch + ".dmg"
	case "windows":
		switch installSource {
		case "installer":
			return base + "-windows-" + goarch + "-installer.exe"
		case "portable":
			return base + "-windows-" + goarch + "-portable.zip"
		}
	case "linux":
		switch installSource {
		case "deb":
			return base + "-linux-" + goarch + ".deb"
		case "tarball":
			return base + "-linux-" + goarch + ".tar.gz"
		}
	}
	return ""
}

// SelectAsset 在 release 的 assets 中按命名约定匹配当前平台与安装来源的产物。
// 匹配不到时返回错误，调用方应按 design D6 兜底打开 Release 页面。
func SelectAsset(r *Release, goos, goarch, installSource string) (*ReleaseAsset, error) {
	want := AssetName(r.TagName, goos, goarch, installSource)
	if want == "" {
		return nil, fmt.Errorf("no asset naming rule for %s/%s install-source %q", goos, goarch, installSource)
	}
	for i := range r.Assets {
		if r.Assets[i].Name == want {
			return &r.Assets[i], nil
		}
	}
	return nil, fmt.Errorf("release %s has no asset %q (install source %q)", r.TagName, want, installSource)
}
