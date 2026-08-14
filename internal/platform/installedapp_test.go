package platform

import "testing"

// TestMatchInstalledAppName 验证已安装应用交叉验证匹配规则（纯函数）：
// 精确相等 / 目录名前缀（≥5 字符）/ 应用名前缀且余段以 "-" 开头（≥5 字符）；
// 短名与普通复合名不误伤。
func TestMatchInstalledAppName(t *testing.T) {
	apps := []string{
		// 注册表卸载项 DisplayName（带版本/描述）与开始菜单快捷方式基名（裸名）
		"PeaZip 11.0.0 (WIN64)",
		"PeaZip",
		"LocalSend 版本 1.17.0",
		"LocalSend",
		"Termius 9.38.1",
		"Termius",
		"Tabby 1.0.233",
		"Tabby",
		"Notepad Next (current user)",
		"Notepad Next",
		"PixPin 版本 3.4.3.2",
		"PixPin",
		"IObit Uninstaller",
		"VoiceAccess",
		"腾讯电脑管家",
		"Git for Windows", // 真实 DisplayName（git 短名只作为前缀存在，不精确相等）
		"Visual Studio Code",
	}
	cases := []struct {
		name string
		want bool
	}{
		// 精确相等（大小写不敏感）
		{"VoiceAccess", true},
		{"voiceaccess", true},
		// 目录名是应用名前缀（应用名 = 目录名 + 版本/描述）
		{"PeaZip", true},
		{"peazip", true},
		{"LocalSend", true},
		{"Termius", true},
		{"Tabby", true},
		{"Notepad Next", true},
		{"PixPin", true},
		{"IObit", true}, // IObit ⊂ "IObit Uninstaller"
		// 应用名是目录名前缀且余段以 "-" 开头（<App>-updater 类 / 中文后缀）
		{"termius-updater", true},
		{"tabby-updater", true},
		{"腾讯电脑管家-全局信息", true},
		// 不误伤：短名不参与前缀匹配
		{"pc", false},
		{"git", false},
		{"code", false},
		{"go", false},
		// 不误伤：普通复合名（应用名虽为前缀，但余段不以 "-" 开头或长度不足）
		{"gitea", false},       // "git" 长度不足 + 余段 "ea" 无 "-"
		{"notepadnext", false}, // "notepad" 是前缀但余段 "next" 无 "-"
		{"opencode", false},    // "code" 长度不足
		{"dead-cli-tool", false},
		{"", false},
	}
	for _, c := range cases {
		if got := MatchInstalledAppName(c.name, apps); got != c.want {
			t.Errorf("MatchInstalledAppName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
