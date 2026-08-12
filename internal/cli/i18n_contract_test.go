package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

// TestUpdateCheckJSONLocaleIndependent 断言 `update check --json` 输出与语言无关：
// 机器契约（键名与结构）在任何语言下字节一致。
func TestUpdateCheckJSONLocaleIndependent(t *testing.T) {
	setupUpdateTest(t, "0.2.3", []mockRelease{{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"}})
	var outputs = map[string][]byte{}
	for _, loc := range i18n.Supported {
		i18n.SetLocale(loc)
		buf := captureStdout(t)
		if code := Run([]string{"update", "check", "--json"}); code != 2 {
			t.Fatalf("%s: exit = %d, want 2", loc, code)
		}
		outputs[loc] = bytes.Clone(buf.Bytes())
	}
	base := outputs["zh-CN"]
	for loc, b := range outputs {
		if !bytes.Equal(base, b) {
			t.Errorf("JSON contract broken for %s:\nzh-CN: %s\n%s: %s", loc, base, loc, b)
		}
	}
}

// TestScanResultJSONLocaleIndependent 断言 ScanResult 的 JSON 序列化与语言无关
// （runScan 的 --json 分支为纯 json.Marshal，此处直接验证其数据不含语言化内容）。
func TestScanResultJSONLocaleIndependent(t *testing.T) {
	res := &scanner.ScanResult{
		Tools: []scanner.Tool{
			{Name: "npm", Footprint: 100, Cleanable: 60, User: 40,
				Cleanables: []scanner.Cleanable{{Path: "/cache/npm", Bytes: 60, Kind: "cache", Tier: scanner.TierSafe}}},
		},
		Totals: scanner.Totals{Footprint: 100, Cleanable: 60, User: 40},
	}
	var outputs = map[string][]byte{}
	for _, loc := range i18n.Supported {
		i18n.SetLocale(loc)
		b, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		outputs[loc] = b
	}
	for loc, b := range outputs {
		if !bytes.Equal(outputs["zh-CN"], b) {
			t.Errorf("ScanResult JSON differs under %s", loc)
		}
	}
}
