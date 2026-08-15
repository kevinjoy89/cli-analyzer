//go:build linux

package i18n

import "os"

// DetectSystem 读取 Linux 系统语言：LC_ALL / LC_MESSAGES / LANG 环境变量。
// 依次取第一个可识别变量（C/POSIX 及 C.UTF-8 语义为英文界面）。
func DetectSystem() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			if n := normalize(v); n != "" {
				return n
			}
		}
	}
	return ""
}
