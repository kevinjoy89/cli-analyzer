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
		// Windows 结构目录仅 LocalAppData
		{LocalAppData, "Programs", true},
		{LocalAppData, "Temp", true},
		{LocalAppData, "CrashDumps", true},
		{LocalAppData, "D3DSCache", true},
		{LocalAppData, "lxss", true},
		{LocalAppData, "Comms", true},
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
