package scanner

import (
	"testing"
)

// TestClearCacheIdempotent 验证无缓存时 clear 视为成功（幂等）：
// 首次运行/已清过的机器上"清除失败"是误导错误。
func TestClearCacheIdempotent(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := ClearCache(); err != nil {
		t.Fatalf("无缓存时 ClearCache 应成功: %v", err)
	}
}
