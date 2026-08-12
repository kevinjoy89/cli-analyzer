//go:build darwin

package i18n

import (
	"os/exec"
	"regexp"
)

// DetectSystem 读取 macOS 系统 UI 语言：`defaults read -g AppleLanguages`
// 的首项（如 "zh-Hans-CN" / "zh-Hant-TW" / "en-US"）。
func DetectSystem() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`"([^"]+)"`)
	m := re.FindSubmatch(out)
	if m == nil {
		return ""
	}
	return normalize(string(m[1]))
}
