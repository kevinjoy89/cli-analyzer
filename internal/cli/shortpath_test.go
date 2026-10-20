package cli

import (
	"testing"

	"cli-analyzer/internal/platform"
)

// TestShortPath 校验终端路径简写与前端 lib/format.ts 同口径：
// 只替换主目录本身或其子路径，前缀相似不算，home 未知时原样返回。
func TestShortPath(t *testing.T) {
	home := platform.HomeDir()
	if home == "" {
		t.Skip("no home directory in this environment")
	}

	cases := []struct{ name, in, want string }{
		{"主目录子路径", home + "/Library/Application Support/JoyCoder", "~/Library/Application Support/JoyCoder"},
		{"主目录本身", home, "~"},
		{"主目录外", "/Applications/WorkBuddy.app", "/Applications/WorkBuddy.app"},
		{"前缀相似不是子路径", home + "2/foo", home + "2/foo"},
		{"空路径", "", ""},
		{"末尾斜杠", home + "/foo/", "~/foo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shortPath(c.in); got != c.want {
				t.Errorf("shortPath(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
