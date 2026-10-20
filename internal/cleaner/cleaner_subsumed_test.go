package cleaner

import (
	"path/filepath"
	"strings"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestCleanDedupesSubsumedSelections 校验父子同选去重（fix-cleanable-sum-overlap）：
// 父目录与被其包含的子目录同时选中时，仅父项处置，子项以 cln.subsumed 跳过。
func TestCleanDedupesSubsumedSelections(t *testing.T) {
	td := t.TempDir()
	parent := filepath.Join(td, ".marscode")
	child := filepath.Join(td, ".marscode", ".ckg")
	res := &scanner.ScanResult{
		Tools: []scanner.Tool{{
			Name: "marscode",
			Cleanables: []scanner.Cleanable{
				{ID: "marscode|data|" + parent, Tool: "marscode", Path: parent, Bytes: 100, Kind: "data"},
				{ID: "marscode|index|" + child, Tool: "marscode", Path: child, Bytes: 90, Kind: "index"},
			},
		}},
	}
	// 子项在前、父项在后：去重与顺序无关
	rep := Clean(res, []string{"marscode|index|" + child, "marscode|data|" + parent}, true)
	if len(rep.Deleted) != 1 || rep.Deleted[0] != parent {
		t.Errorf("应仅处置父项 %s，得到 %v", parent, rep.Deleted)
	}
	if rep.Freed != 100 {
		t.Errorf("释放字节应按父项单份计 100，得到 %d", rep.Freed)
	}
	if len(rep.Skipped) != 1 || !strings.Contains(rep.Skipped[0].Path, child) {
		t.Errorf("子项应入 Skipped，得到 %v", rep.Skipped)
	}
	if rep.Skipped[0].Reason != SkipSubsumed {
		t.Errorf("跳过原因码 = %q, want %q", rep.Skipped[0].Reason, SkipSubsumed)
	}
	// 仅选子项时正常处置（不受去重影响）
	rep2 := Clean(res, []string{"marscode|index|" + child}, true)
	if len(rep2.Deleted) != 1 || rep2.Deleted[0] != child {
		t.Errorf("仅选子项应正常处置，得到 %v", rep2.Deleted)
	}
}
