//go:build windows

package i18n

import "syscall"

// DetectSystem 读取 Windows 系统 UI 语言（GetUserDefaultUILanguage 的 LANGID）。
// 映射：primary 9 → en；primary 4 + sublang 4（简体）→ zh-CN；其余 primary 4 → zh-TW。
func DetectSystem() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")
	langID, _, _ := proc.Call()
	primary := uint16(langID) & 0x3ff
	sublang := uint16(langID) >> 10
	switch primary {
	case 9: // English
		return "en"
	case 4: // Chinese
		if sublang == 0x04 { // 简体
			return "zh-CN"
		}
		return "zh-TW"
	}
	return ""
}
