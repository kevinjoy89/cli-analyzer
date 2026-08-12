// Package i18n 提供应用的语言解析与翻译能力。
//
// 语言文件位于本包目录下的 locales/（internal/i18n/locales/*.json），
// 经 go:embed 嵌入二进制，是全部用户可见文案的唯一来源：
//   - Go 侧（CLI/菜单/后端错误）直接调用 T()；
//   - 前端经 GUI 绑定（GetTranslations）拿到同一份字典后自行渲染。
//
// 默认 locale 为 zh-CN（与历史行为逐字一致）；SetLocale 仅接受受支持
// 的语言，非法值回退 zh-CN。
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

// Supported 返回受支持的语言列表（顺序即首选项下拉顺序）。
var Supported = []string{"zh-CN", "zh-TW", "en"}

type dict map[string]string

var (
	mu     sync.RWMutex
	active = "zh-CN"
	dicts  = map[string]dict{}
)

func init() {
	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		panic("i18n: read locales: " + err.Error())
	}
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		b, err := localesFS.ReadFile("locales/" + e.Name())
		if err != nil {
			panic("i18n: read " + e.Name() + ": " + err.Error())
		}
		d := dict{}
		if err := json.Unmarshal(b, &d); err != nil {
			panic("i18n: parse " + e.Name() + ": " + err.Error())
		}
		dicts[name] = d
	}
}

// IsSupported 报告 locale 是否为受支持的语言。
func IsSupported(locale string) bool {
	for _, s := range Supported {
		if s == locale {
			return true
		}
	}
	return false
}

// SetLocale 设置当前语言；非法值回退 zh-CN。返回生效的语言。
func SetLocale(locale string) string {
	if !IsSupported(locale) {
		locale = "zh-CN"
	}
	mu.Lock()
	active = locale
	mu.Unlock()
	return locale
}

// ActiveLocale 返回当前生效的语言。
func ActiveLocale() string {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

// dictFor 返回当前语言的字典（只读使用，调用方持锁）。
func dictFor() dict {
	mu.RLock()
	d := dicts[active]
	mu.RUnlock()
	if d == nil {
		d = dicts["zh-CN"]
	}
	return d
}

// T 返回 key 对应的当前语言文案，并将 {name} 占位符替换为 args 中的值。
//
// 复数规则：当 key 存在 _other 变体且 args 含 "n" 时，英文（en）按 n==1
// 选 _one，其余情况选 _other；中文两种变体同值（见语言文件）。
// 缺失的键返回键名本身（便于发现漏翻），缺失的参数保留占位符原样。
func T(key string, args ...map[string]any) string {
	d := dictFor()
	var a map[string]any
	if len(args) > 0 {
		a = args[0]
	}
	if n, ok := numericArg(a, "n"); ok {
		if v, exists := d[key+"_other"]; exists {
			if active == "en" && n == 1 {
				if one, ok2 := d[key+"_one"]; ok2 {
					return interpolate(one, a)
				}
			}
			return interpolate(v, a)
		}
	}
	v, ok := d[key]
	if !ok {
		return key
	}
	return interpolate(v, a)
}

// Dict 返回某语言的完整字典（只读副本）；非法语言回退 zh-CN。
func Dict(locale string) map[string]string {
	if !IsSupported(locale) {
		locale = "zh-CN"
	}
	d := dicts[locale]
	if d == nil {
		d = dicts["zh-CN"]
	}
	out := make(map[string]string, len(d))
	for k, v := range d {
		out[k] = v
	}
	return out
}

// LocaleKeys 返回某语言的完整键集（parity 测试用）。
func LocaleKeys(locale string) []string {
	d := dicts[locale]
	if d == nil {
		return nil
	}
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// KeysEqual 报告两个语言的键集是否完全一致（parity 测试用）。
func KeysEqual(a, b string) bool {
	ka, kb := LocaleKeys(a), LocaleKeys(b)
	if len(ka) != len(kb) {
		return false
	}
	for i := range ka {
		if ka[i] != kb[i] {
			return false
		}
	}
	return true
}

// numericArg 从 args 中取 "n" 的数值（int/int64/float64 等）。
func numericArg(args map[string]any, key string) (int64, bool) {
	if args == nil {
		return 0, false
	}
	switch v := args[key].(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	}
	return 0, false
}

// interpolate 替换 {name} 占位符；缺失参数保留原样。
func interpolate(s string, args map[string]any) string {
	if args == nil {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	for {
		start := strings.IndexByte(s, '{')
		if start < 0 {
			b.WriteString(s)
			break
		}
		end := strings.IndexByte(s[start:], '}')
		if end < 0 {
			b.WriteString(s)
			break
		}
		name := s[start+1 : start+end]
		b.WriteString(s[:start])
		if v, ok := args[name]; ok {
			b.WriteString(fmt.Sprint(v))
		} else {
			b.WriteString(s[start : start+end+1])
		}
		s = s[start+end+1:]
	}
	return b.String()
}

// Translated 是按当前语言延迟翻译的错误值，供哨兵错误保持 errors.Is 身份：
// 直接返回或 %w 包装均可被 errors.Is 匹配，消息在 Error() 时按当时语言生成。
type Translated struct {
	Key  string
	Args map[string]any
}

// NewError 构造一个延迟翻译的错误（返回指针以保持 errors.Is 的可比较性）。
func NewError(key string, args ...map[string]any) error {
	var a map[string]any
	if len(args) > 0 {
		a = args[0]
	}
	return &Translated{Key: key, Args: a}
}

func (e *Translated) Error() string { return T(e.Key, e.Args) }
