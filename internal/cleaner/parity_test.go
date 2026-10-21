package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
)

// 这些用例与 frontend/src/lib/clean-parity.test.ts **一一对应**：CLI 的
// cleaner 与 GUI 的 keepMaximalPaths 必须对同一组输入给出同一个合计。
// 任何一边改了去重口径，至少有一边会红——这正是「弹窗说 7.9G、执行只
// 回收 3.9G」这类分叉的防线。

// mkClean 构造一个最小可处置项（路径由调用方给出，父目录自动创建）。
func mkClean(t *testing.T, dir, name string, bytes int64) scanner.Cleanable {
	t.Helper()
	return scanner.Cleanable{ID: name, Tool: "x", Path: dir, Bytes: bytes, Kind: "cache"}
}

// dryRunFreed 返回全选这些项时 cleaner 报告的释放字节（dry-run 不落盘）。
func dryRunFreed(t *testing.T, items []scanner.Cleanable) int64 {
	t.Helper()
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	res := &scanner.ScanResult{Tools: []scanner.Tool{{Name: "x", Cleanables: items}}}
	return Clean(res, ids, true).Freed
}

// TestMaximalPathTotals 与前端 clean-parity.test.ts 的用例表对应。
func TestMaximalPathTotals(t *testing.T) {
	base := t.TempDir()
	join := func(p ...string) string { return filepath.Join(append([]string{base}, p...)...) }

	cases := []struct {
		name  string
		paths []string
		bytes []int64
		want  int64
	}{
		{
			name:  "互不相交：全额相加",
			paths: []string{join("a"), join("b")},
			bytes: []int64{100, 200},
			want:  300,
		},
		{
			name:  "父子并存：只计父项",
			paths: []string{join("a"), join("a", "child")},
			bytes: []int64{100, 40},
			want:  100,
		},
		{
			name:  "三层嵌套：只计最外层",
			paths: []string{join("a"), join("a", "b"), join("a", "b", "c")},
			bytes: []int64{100, 60, 30},
			want:  100,
		},
		{
			name:  "前缀相似但不是父子：都计入",
			paths: []string{join("a"), join("ab")},
			bytes: []int64{100, 200},
			want:  300,
		},
		{
			name:  "子项先选、父项后选：仍只计父项",
			paths: []string{join("a", "b"), join("a")},
			bytes: []int64{40, 100},
			want:  100,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items := make([]scanner.Cleanable, 0, len(c.paths))
			for i, p := range c.paths {
				items = append(items, mkClean(t, p, p, c.bytes[i]))
			}
			if got := dryRunFreed(t, items); got != c.want {
				t.Errorf("freed = %d, want %d", got, c.want)
			}
		})
	}
}

// TestMaximalPathTotalsWindowsSeparators 对应前端同名用例：Windows 反斜杠
// 路径下 filepath.Rel 仍能识别父子关系。
func TestMaximalPathTotalsWindowsSeparators(t *testing.T) {
	// 必须用真实临时目录做基：Windows 上 "/x" 不是绝对路径（缺盘符），
	// guard 会以 not-absolute 拒绝，测不到去重逻辑。
	base := t.TempDir()
	parent := filepath.Join(base, "x")
	child := filepath.Join(parent, "y")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	items := []scanner.Cleanable{
		mkClean(t, parent, "p", 100),
		mkClean(t, child, "c", 50),
	}
	if got := dryRunFreed(t, items); got != 100 {
		t.Errorf("freed = %d, want 100 (child subsumed)", got)
	}
}
