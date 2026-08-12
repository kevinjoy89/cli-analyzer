//go:build linux

package i18n

import "os"

// DetectSystem 读取 Linux 系统语言：LC_ALL / LC_MESSAGES / LANG 环境变量。
func DetectSystem() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" && v != "C" && v != "POSIX" {
			return normalize(v)
		}
	}
	return ""
}
