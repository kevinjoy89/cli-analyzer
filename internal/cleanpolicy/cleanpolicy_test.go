package cleanpolicy

import (
	"reflect"
	"testing"

	"cli-analyzer/internal/config"
)

// TestDefaultSet 校验未改配置时，可批量清理的 Kind 恰好是内置那 7 种。
func TestDefaultSet(t *testing.T) {
	got := BulkKinds(config.CleanConfig{})
	want := []string{"backup", "cache", "download", "index", "logs", "old-version", "telemetry"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("default bulk kinds = %v, want %v", got, want)
	}
}

// TestBulkEligibleDefaults 校验默认配置下的资格判定。
func TestBulkEligibleDefaults(t *testing.T) {
	cfg := config.CleanConfig{}
	for _, k := range []string{"cache", "logs", "old-version"} {
		if !BulkEligible(k, cfg) {
			t.Errorf("%q should be bulk-eligible by default", k)
		}
	}
	// runtime 是 safe 但刻意不在批量集合内——这是本包存在的核心理由
	for _, k := range []string{"runtime", "config", "data", "session", "plugins", "install", ""} {
		if BulkEligible(k, cfg) {
			t.Errorf("%q should NOT be bulk-eligible by default", k)
		}
	}
}

// TestBulkKindsAppend 校验配置可追加自定义 Kind。
func TestBulkKindsAppend(t *testing.T) {
	cfg := config.CleanConfig{BulkKinds: []string{"my-cache"}}
	if !BulkEligible("my-cache", cfg) {
		t.Error("appended kind should be eligible")
	}
	// 内置项仍然在
	if !BulkEligible("cache", cfg) {
		t.Error("builtin kinds must survive an append")
	}
	got := BulkKinds(cfg)
	if len(got) != 8 {
		t.Errorf("len = %d, want 8: %v", len(got), got)
	}
}

// TestBulkExclude 校验配置可剔除内置 Kind。
func TestBulkExclude(t *testing.T) {
	cfg := config.CleanConfig{BulkExclude: []string{"cache"}}
	if BulkEligible("cache", cfg) {
		t.Error("excluded kind must not be eligible")
	}
	if !BulkEligible("logs", cfg) {
		t.Error("other builtin kinds must survive an exclude")
	}
}

// TestExcludeBeatsAppend 校验同时出现在两个列表时，剔除优先。
func TestExcludeBeatsAppend(t *testing.T) {
	cfg := config.CleanConfig{
		BulkKinds:   []string{"my-cache", "cache"},
		BulkExclude: []string{"cache"},
	}
	if BulkEligible("cache", cfg) {
		t.Error("exclude must win over append")
	}
	if !BulkEligible("my-cache", cfg) {
		t.Error("my-cache should still be eligible")
	}
}

// TestWhitespaceTrimmed 校验配置里的空白被忽略。
func TestWhitespaceTrimmed(t *testing.T) {
	cfg := config.CleanConfig{BulkKinds: []string{"  my-cache\n"}}
	if !BulkEligible("my-cache", cfg) {
		t.Error("surrounding whitespace should be trimmed")
	}
}

// TestNewKindNotBulkByDefault 是本包的核心安全默认：
// 用户规则包声明的自定义 Kind，即使 tier=safe，也不会被 --all 一并清掉。
func TestNewKindNotBulkByDefault(t *testing.T) {
	// 模拟用户规则包声明的 kind，配置未声明
	if BulkEligible("user-defined-cache", config.CleanConfig{}) {
		t.Error("a user-declared kind must NOT be bulk-eligible until configured")
	}
}
