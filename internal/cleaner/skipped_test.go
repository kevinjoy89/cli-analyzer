package cleaner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestSkippedItemsAreStructured 校验 clean 报告的 skipped 是结构化条目：
// 稳定原因码 + 本地化文案分开给出，脚本与界面各取所需。
// 此前 skipped 是 "<path> (<reason>)" 字符串，脚本无法解析、界面无法本地化。
func TestSkippedItemsAreStructured(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "JoyCoder")
	child := filepath.Join(parent, "CachedData")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	res := &scanner.ScanResult{Tools: []scanner.Tool{{
		Name: "joycoder",
		Cleanables: []scanner.Cleanable{
			{ID: "p", Tool: "joycoder", Path: parent, Bytes: 100, Kind: "data"},
			{ID: "c", Tool: "joycoder", Path: child, Bytes: 40, Kind: "cache"},
		},
	}}}

	rep := Clean(res, []string{"p", "c"}, true)
	if len(rep.Skipped) != 1 {
		t.Fatalf("skipped = %+v, want 1 entry", rep.Skipped)
	}
	s := rep.Skipped[0]
	if s.Reason != SkipSubsumed {
		t.Errorf("reason = %q, want %q", s.Reason, SkipSubsumed)
	}
	if s.Path != child {
		t.Errorf("path = %q, want %q", s.Path, child)
	}
	if s.ID != "c" {
		t.Errorf("id = %q, want %q", s.ID, "c")
	}
	if s.Message == "" {
		t.Error("message should carry the localized explanation")
	}
}

// TestGuardReasonsHaveCodes 校验每个 guard 分支都给出稳定码，
// 而不是只返回一句人话。
func TestGuardReasonsHaveCodes(t *testing.T) {
	// 路径必须用平台原生的绝对形式构造：Windows 上 filepath.IsAbs("/a/../b")
	// 为 false，会先命中 not-absolute 而不是预期原因。
	base := t.TempDir()
	forbidden := "/usr/bin"
	if runtime.GOOS == "windows" {
		forbidden = `C:\Windows`
	}
	cases := []struct {
		name string
		c    scanner.Cleanable
		want string
	}{
		{"relative", scanner.Cleanable{Path: "relative/path"}, SkipNotAbsolute},
		// 不能用 filepath.Join：它会自动 Clean 掉 ".."，路径就不“脏”了。
		// 直接拼一个含冗余段但仍是绝对路径的串。
		{"dirty", scanner.Cleanable{Path: base + string(filepath.Separator) + "a" + string(filepath.Separator) + ".." + string(filepath.Separator) + "b"}, SkipDirtyPath},
		{"forbidden", scanner.Cleanable{Path: forbidden}, SkipForbiddenRoot},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := guard(&tc.c)
			if r.Code != tc.want {
				t.Errorf("code = %q, want %q (path=%q)", r.Code, tc.want, tc.c.Path)
			}
			if r.Message == "" {
				t.Error("message should be set alongside the code")
			}
		})
	}
}

// TestGuardSubNotInside 校验子项越出父目录时给 not-inside-parent。
func TestGuardSubNotInside(t *testing.T) {
	base := t.TempDir()
	r := guardSub(filepath.Join(base, "parent"), filepath.Join(base, "other"))
	if r.Code != SkipNotInside {
		t.Errorf("code = %q, want %q", r.Code, SkipNotInside)
	}
}

// TestUnknownIDReason 校验未知 id 也给结构化原因。
func TestUnknownIDReason(t *testing.T) {
	res := &scanner.ScanResult{Tools: []scanner.Tool{{Name: "x"}}}
	rep := Clean(res, []string{"no-such-id"}, true)
	if len(rep.Skipped) != 1 || rep.Skipped[0].Reason != SkipUnknownItem {
		t.Fatalf("skipped = %+v, want one unknown-item", rep.Skipped)
	}
}
