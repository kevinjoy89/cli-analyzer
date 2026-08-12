package i18n

import (
	"errors"
	"strings"
	"testing"
)

func TestSetLocaleAndActive(t *testing.T) {
	defer SetLocale("zh-CN")
	if got := SetLocale("en"); got != "en" || ActiveLocale() != "en" {
		t.Fatalf("SetLocale(en) = %q, active %q", got, ActiveLocale())
	}
	if got := SetLocale("bogus"); got != "zh-CN" || ActiveLocale() != "zh-CN" {
		t.Fatalf("SetLocale(bogus) should fall back zh-CN, got %q", got)
	}
}

func TestTInterpolation(t *testing.T) {
	defer SetLocale("zh-CN")
	SetLocale("en")
	if got := T("ui.confirmClean", map[string]any{"n": 3, "size": "1.5 GB"}); !strings.Contains(got, "3 items") || !strings.Contains(got, "1.5 GB") {
		t.Errorf("interpolation = %q", got)
	}
	// 缺失参数保留占位符
	if got := T("ui.confirmClean", map[string]any{"n": 3}); !strings.Contains(got, "{size}") {
		t.Errorf("missing arg should keep placeholder, got %q", got)
	}
}

func TestPluralEnglish(t *testing.T) {
	defer SetLocale("zh-CN")
	SetLocale("en")
	one := T("ui.selectedCount", map[string]any{"n": 1})
	if one != "1 selected" {
		t.Errorf("singular = %q, want \"1 selected\"", one)
	}
	many := T("ui.selectedCount", map[string]any{"n": 5})
	if many != "5 selected" {
		t.Errorf("plural = %q, want \"5 selected\"", many)
	}
}

func TestPluralChineseSameValue(t *testing.T) {
	defer SetLocale("zh-CN")
	// 中文句式不随数量变化：两个复数变体的原始值应相同
	one := dicts["zh-CN"]["ui.selectedCount_one"]
	many := dicts["zh-CN"]["ui.selectedCount_other"]
	if one != many {
		t.Errorf("zh plural variants differ: %q vs %q", one, many)
	}
	// 渲染值仅数量数字不同
	n1 := T("ui.selectedCount", map[string]any{"n": 1})
	n5 := T("ui.selectedCount", map[string]any{"n": 5})
	if n1 != "已选 1 项" || n5 != "已选 5 项" {
		t.Errorf("zh rendering = %q / %q", n1, n5)
	}
}

func TestMissingKeyReturnsKey(t *testing.T) {
	defer SetLocale("zh-CN")
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key = %q, want key itself", got)
	}
}

func TestResolve(t *testing.T) {
	if Resolve("zh-TW") != "zh-TW" {
		t.Error("explicit zh-TW")
	}
	if Resolve("en") != "en" {
		t.Error("explicit en")
	}
	if Resolve("bogus") != "zh-CN" {
		t.Error("invalid explicit falls back")
	}
	// auto 依赖系统探测：本机未知时回退 zh-CN；无论如何不得 panic
	got := Resolve("auto")
	switch got {
	case "zh-CN", "zh-TW", "en":
	default:
		t.Errorf("Resolve(auto) = %q", got)
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"zh-CN":      "zh-CN",
		"zh-Hans-CN": "zh-CN",
		"zh_SG":      "zh-CN",
		"zh-TW":      "zh-TW",
		"zh-Hant-TW": "zh-TW",
		"zh-HK":      "zh-TW",
		"en-US":      "en",
		"en":         "en",
		"ja-JP":      "",
		"de":         "",
	}
	for in, want := range cases {
		if got := normalize(in); got != want {
			t.Errorf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTranslatedError(t *testing.T) {
	defer SetLocale("zh-CN")
	// 哨兵身份保持：直接返回可被 errors.Is 匹配
	sentinel := NewError("err.trashCrossFS")
	if !errors.Is(sentinel, sentinel) {
		t.Fatal("sentinel must match itself")
	}
	// %w 包装后仍可匹配
	wrapped := errors.New("ctx")
	wrapped = &wrapError{msg: "outer", err: sentinel}
	if !errors.Is(wrapped, sentinel) {
		t.Fatal("wrapped sentinel must match")
	}
	// 消息按当前语言生成
	SetLocale("en")
	if !strings.Contains(sentinel.Error(), "different filesystems") {
		t.Errorf("en message = %q", sentinel.Error())
	}
	SetLocale("zh-CN")
	if sentinel.Error() != "目标与回收站不在同一文件系统，无法移入" {
		t.Errorf("zh message = %q", sentinel.Error())
	}
}

type wrapError struct {
	msg string
	err error
}

func (w *wrapError) Error() string { return w.msg }
func (w *wrapError) Unwrap() error { return w.err }

// parity：三个语言文件键集必须完全一致（无缺失/多余/空值）
func TestParityAllLocales(t *testing.T) {
	base := "zh-CN"
	for _, other := range Supported {
		if other == base {
			continue
		}
		if !KeysEqual(base, other) {
			missing, extra := keyDiff(base, other)
			t.Errorf("parity %s vs %s:\n  zh-CN 独有: %v\n  %s 独有: %v",
				base, other, missing, other, extra)
		}
	}
	// 空值检查
	for _, locale := range Supported {
		for k, v := range dicts[locale] {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: key %q is empty", locale, k)
			}
		}
	}
}

func keyDiff(a, b string) (missing, extra []string) {
	ka, kb := LocaleKeys(a), LocaleKeys(b)
	kbSet := map[string]bool{}
	for _, k := range kb {
		kbSet[k] = true
	}
	for _, k := range ka {
		if !kbSet[k] {
			missing = append(missing, k)
		}
	}
	kaSet := map[string]bool{}
	for _, k := range ka {
		kaSet[k] = true
	}
	for _, k := range kb {
		if !kaSet[k] {
			extra = append(extra, k)
		}
	}
	return
}
