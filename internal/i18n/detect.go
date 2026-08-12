package i18n

import "strings"

// Resolve 将显式配置的语言解析为生效语言：
//   - 显式语言（zh-CN / zh-TW / en）直接使用；
//   - "auto" 或空值走 DetectSystem() 平台探测；
//   - 探测不到或不受支持时回退 zh-CN（与历史行为一致）。
func Resolve(explicit string) string {
	switch explicit {
	case "zh-CN", "zh-TW", "en":
		return explicit
	case "auto", "":
		if d := DetectSystem(); d != "" {
			return d
		}
		return "zh-CN"
	default:
		return "zh-CN"
	}
}

// normalize 将任意系统语言标识映射为受支持语言；无法识别时返回空串。
// 支持形如 "zh-CN"、"zh-Hans-CN"、"zh_TW"、"en-US"、"en" 的常见形式。
func normalize(raw string) string {
	lower := strings.ToLower(strings.ReplaceAll(raw, "_", "-"))
	switch {
	case strings.HasPrefix(lower, "zh-hans"), strings.HasPrefix(lower, "zh-cn"),
		strings.HasPrefix(lower, "zh-sg"), lower == "zh":
		return "zh-CN"
	case strings.HasPrefix(lower, "zh-hant"), strings.HasPrefix(lower, "zh-tw"),
		strings.HasPrefix(lower, "zh-hk"), strings.HasPrefix(lower, "zh-mo"):
		return "zh-TW"
	case strings.HasPrefix(lower, "en"):
		return "en"
	}
	return ""
}
