package cli

import (
	"math"
	"testing"
)

// TestHumanBytesLarge 验证超大字节数不 panic：
// humanBytes 用 "KMGT"[exp] 选单位后缀，n ≥ 1024^5（1 PiB）时 exp=4
// 越界 panic——int64 范围内真实可触发（1<<50 即 1 PiB）。
func TestHumanBytesLarge(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{1 << 40, "1.0 TB"}, // 1 TiB → T
		{1 << 50, "1.0 PB"}, // 1 PiB → 应显示 PB，且不 panic
		{1<<60 - 1, ""},     // 超大值只要求不 panic
		{math.MaxInt64, ""}, // 极端边界只要求不 panic
	}
	for _, c := range cases {
		got := humanBytes(c.n)
		if c.want != "" && got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

// TestHumanBytesNormal 回归：常规大小格式化不变。
func TestHumanBytesNormal(t *testing.T) {
	cases := map[int64]string{
		0:           "0 B",
		1023:        "1023 B",
		1024:        "1.0 KB",
		1 << 20:     "1.0 MB",
		1 << 30:     "1.0 GB",
		5 * 1 << 30: "5.0 GB",
	}
	for n, want := range cases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
