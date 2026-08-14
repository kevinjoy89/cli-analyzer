package platform

import "testing"

// TestIsSystemDataDir 验证系统/共享结构目录判定：跨平台共享目录（node 生态、
// .NET/Mono、man 库）在各数据根均排除；Windows 结构目录仅 LocalAppData 下
// 排除；大小写不敏感；普通目录不受影响。
func TestIsSystemDataDir(t *testing.T) {
	cases := []struct {
		root RootKind
		name string
		want bool
	}{
		// 跨平台共享目录
		{XDGConfig, "configstore", true},
		{XDGData, "ConfigStore", true}, // 大小写不敏感
		{XDGCache, "simple-update-notifier", true},
		{XDGConfig, ".mono", true},
		{XDGData, "IsolatedStorage", true},
		{XDGData, ".isolated-storage", true},
		{XDGData, "man", true},
		{XDGConfig, "man", true},
		// Go 工具链构建缓存（跨平台共享基础设施）
		{XDGCache, "go-build", true},
		{LocalAppData, "go-build", true},
		{AppData, "go-build", true},
		// Windows 结构目录仅 LocalAppData
		{LocalAppData, "Programs", true},
		{LocalAppData, "Temp", true},
		{LocalAppData, "CrashDumps", true},
		{LocalAppData, "D3DSCache", true},
		{LocalAppData, "lxss", true},
		{LocalAppData, "Comms", true},
		{LocalAppData, "ConnectedDevicesPlatform", true},
		{LocalAppData, "PlaceholderTileLogoFolder", true},
		{LocalAppData, "SquirrelTemp", true},
		// Windows 语音访问组件：跨平台表，AppData（Roaming）根同样排除
		{AppData, "VoiceAccess", true},
		{LocalAppData, "voiceaccess", true},
		// 非 LocalAppData 下的同名目录不受影响
		{AppData, "Programs", false},
		{XDGData, "Temp", false},
		// 普通目录不受影响
		{XDGConfig, "dead-cli-tool", false},
		{LocalAppData, "npm-cache", false},
		{AppData, "gh", false},
	}
	for _, c := range cases {
		if got := IsSystemDataDir(c.root, c.name); got != c.want {
			t.Errorf("IsSystemDataDir(%s, %q) = %v, want %v", c.root, c.name, got, c.want)
		}
	}
}

// TestIsUpdaterDir 验证应用更新器目录结构规则：小写 "-updater" 后缀命中
// （Squirrel/electron-updater 约定 <App>-updater），普通目录不受影响。
func TestIsUpdaterDir(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"tabby-updater", true},
		{"termius-updater", true},
		{"qmlauncher-updater", true},
		{"Tabby-Updater", true}, // 大小写不敏感
		{"tabby", false},
		{"updater", false}, // 无 "-" 前缀
		{"dead-cli-tool", false},
		{"flutter_webview_windows", false},
	}
	for _, c := range cases {
		if got := IsUpdaterDir(c.name); got != c.want {
			t.Errorf("IsUpdaterDir(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
