//go:build windows

package platform

import "testing"

// TestIsInstalledAppDataDir 用注入的假应用名单验证 Windows 集成入口
// （IsInstalledAppDataDir → installedAppNames → 匹配器）与测试注入钩子。
// 真实注册表/开始菜单读取由本机实机验证覆盖。
func TestIsInstalledAppDataDir(t *testing.T) {
	installedAppNamesOverride = []string{
		"PeaZip 11.0.0 (WIN64)",
		"Termius 9.38.1",
		"Termius", // 开始菜单快捷方式基名（裸名）形态
		"腾讯电脑管家",
	}
	resetInstalledAppNames()
	defer func() {
		installedAppNamesOverride = nil
		resetInstalledAppNames()
	}()

	hit := []string{"PeaZip", "termius-updater", "腾讯电脑管家-全局信息"}
	for _, n := range hit {
		if !IsInstalledAppDataDir(n) {
			t.Errorf("IsInstalledAppDataDir(%q): want hit", n)
		}
	}
	miss := []string{"pc", "dead-cli-tool", "clink", "gitea", "notepadnext"}
	for _, n := range miss {
		if IsInstalledAppDataDir(n) {
			t.Errorf("IsInstalledAppDataDir(%q): want miss", n)
		}
	}
}

// TestInstalledAppNamesLoad 冒烟：真实加载不 panic 且返回非空名单（本机
// Windows 一定有开始菜单快捷方式或卸载项）。
func TestInstalledAppNamesLoad(t *testing.T) {
	resetInstalledAppNames()
	defer resetInstalledAppNames()
	names := installedAppNames()
	if len(names) == 0 {
		t.Fatal("installedAppNames() returned empty list on Windows")
	}
}
