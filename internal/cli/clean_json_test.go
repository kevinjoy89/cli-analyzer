package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestCleanJSONAlwaysEmitsJSON 守住 --json 契约：无论走哪条分支，
// stdout 都必须是一份合法 JSON 文档。
//
// 回归：`clean --all --json` 在批量选择为空时曾直接打纯文本
// 「没有可处置的项目」，脚本拿到非 JSON 直接解析失败。
func TestCleanJSONAlwaysEmitsJSON(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"all-empty", []string{"clean", "--all", "--dry-run", "--json"}},
		{"all-include-data", []string{"clean", "--all", "--include-data", "--dry-run", "--json"}},
		{"list", []string{"clean", "--list", "--json"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// 空扫描缓存：任何选择分支都命中「无可处置项」路径
			cacheEnv(t)
			if err := scanner.SaveCache(&scanner.ScanResult{Tools: []scanner.Tool{}}); err != nil {
				t.Fatal(err)
			}
			buf := captureStdout(t)
			_ = Run(c.args)
			out := bytes.TrimSpace(buf.Bytes())
			if len(out) == 0 {
				t.Fatal("no output at all")
			}
			var doc map[string]any
			if err := json.Unmarshal(out, &doc); err != nil {
				t.Fatalf("stdout is not JSON: %v\n%s", err, out)
			}
			if doc["kind"] != KindClean {
				t.Errorf("kind = %v, want %q", doc["kind"], KindClean)
			}
			if _, ok := doc["schemaVersion"]; !ok {
				t.Error("schemaVersion missing")
			}
		})
	}
}
