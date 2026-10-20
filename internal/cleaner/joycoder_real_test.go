package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestJoyCoderParentChildReal 复现实机 joycoder 的情形：父项
// ~/Library/Application Support/JoyCoder (22.4MB) 与子项 .../CachedData
// (10.1MB) 同时被选中。确认 GUI 弹窗显示的 22.4MB / 1 项是对的。
func TestJoyCoderParentChildReal(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "JoyCoder")
	child := filepath.Join(parent, "CachedData")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	res := &scanner.ScanResult{Tools: []scanner.Tool{{
		Name: "joycoder",
		Cleanables: []scanner.Cleanable{
			{ID: "joycoder|data|" + parent, Tool: "joycoder", Path: parent, Bytes: 23491424, Kind: "data"},
			{ID: "joycoder|cache|" + child, Tool: "joycoder", Path: child, Bytes: 10576864, Kind: "cache"},
		},
	}}}

	// 全选（GUI 勾了父 + 子）
	rep := Clean(res, []string{
		"joycoder|data|" + parent,
		"joycoder|cache|" + child,
	}, true)

	t.Logf("deleted=%v freed=%d skipped=%v", rep.Deleted, rep.Freed, rep.Skipped)
	if len(rep.Deleted) != 1 || rep.Deleted[0] != parent {
		t.Errorf("只应处置父项，得到 %v", rep.Deleted)
	}
	if rep.Freed != 23491424 {
		t.Errorf("freed = %d, want 23491424（父项单份，不重复计子项）", rep.Freed)
	}
	if len(rep.Skipped) != 1 {
		t.Errorf("子项应进 skipped，得到 %v", rep.Skipped)
	}
}
