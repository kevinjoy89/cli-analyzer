package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"cli-analyzer/internal/scanner"
)

// 契约结构快照：冻结每份 JSON 报告的**结构**（字段名 + JSON 类型 + 是否可选），
// 不冻结 desc / keep / sub 等内容型字段的**值**。
//
// 为什么只冻结结构：契约测试要防的是「字段被删 / 改名 / 类型变了」，而不是
// 「文案改了」。全量值快照会随规则表与 i18n 天天变红，最后没人再看它。
// 见 docs/adr/0002-cleanable-actionable-and-contract-compat.md。
//
// 更新方式（有意变更契约时）：
//
//	UPDATE_CONTRACT=1 go test ./internal/cli/ -run TestContractStructure
const contractSnapshotDir = "testdata/contract"

// contractCase 是一份需要冻结结构的报告。
type contractCase struct {
	name    string // 快照文件名（不含 .json）
	kind    string // 期望的顶层 kind
	payload any    // 报告载荷（零值即可，只看结构）
}

func contractCases() []contractCase {
	return []contractCase{
		{"scan", KindScan, scanContractSample()},
		{"clean", KindClean, cleanContractSample()},
		{"cache", KindCache, CacheReport{}},
		{"trash-list", KindTrashList, TrashReport{}},
		{"trash-restore", KindTrashRestore, map[string]any{"id": "", "restored": ""}},
		{"trash-empty", KindTrashEmpty, map[string]any{"deleted": []string{}, "errors": []string{}}},
		{"trends", KindTrends, TrendsReport{}},
		{"uninstall", KindUninstall, UninstallReport{}},
		{"baseline-diff", KindBaseline, BaselineDiff{}},
	}
}

// scanContractSample 是一份带一个工具、一个可处置项（含子项）的扫描结果。
// 零值不够用：空数组与缺失字段在结构快照里长得不一样（前者记录元素类型）。
func scanContractSample() *scanner.ScanResult {
	return &scanner.ScanResult{
		Tools: []scanner.Tool{{
			Name:        "x",
			DisplayName: "X",
			Aliases:     []string{"x"},
			Installer:   "brew",
			Family:      "fam",
			Package:     "pkg",
			Category:    "agent",
			Uninstalled: true,
			Binaries:    []scanner.Binary{{Name: "x", Path: "/x", Real: "/x"}},
			DataDirs:    []scanner.DataDir{{Path: "/d", Kind: "cache", Tier: scanner.TierSafe, Bytes: 1, Root: "home", Form: "desktop"}},
			Cleanables: []scanner.Cleanable{{
				ID: "x|cache|/d", Tool: "x", Path: "/d", Bytes: 1, Tier: scanner.TierSafe, Kind: "cache",
				Sub: []scanner.SubEntry{{Path: "/d/s", Bytes: 1, ID: "x|cache|/d::/d/s", Kind: "cache"}},
			}},
		}},
		Totals:       scanner.Totals{Footprint: 1, Cleanable: 1},
		Roots:        map[string][]string{"home": {"/d"}},
		Unattributed: []scanner.DataDir{{Path: "/u", Bytes: 1, Root: "home", Form: "cli"}},
		RuleErrors:   []string{"mine.json: boom"},
		OmittedTools: 2,
	}
}

// cleanContractSample 是一份含处置、跳过与错误的清理报告。
func cleanContractSample() *scanner.CleanReport {
	return &scanner.CleanReport{
		DryRun:  true,
		Deleted: []string{"/d"},
		Freed:   1,
		Skipped: []string{"/s (subsumed)"},
		Errors:  []string{},
	}
}

// TestContractStructure 比对每份报告的结构快照。
func TestContractStructure(t *testing.T) {
	update := os.Getenv("UPDATE_CONTRACT") == "1"
	for _, c := range contractCases() {
		t.Run(c.name, func(t *testing.T) {
			shape, err := shapeOf(c.kind, c.payload)
			if err != nil {
				t.Fatalf("shape: %v", err)
			}
			got := strings.Join(shape, "\n") + "\n"
			path := filepath.Join(contractSnapshotDir, c.name+".txt")
			if update {
				if err := os.MkdirAll(contractSnapshotDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated %s", path)
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing snapshot %s (run with UPDATE_CONTRACT=1 to create): %v", path, err)
			}
			if strings.TrimSpace(string(want)) != strings.TrimSpace(got) {
				t.Errorf("contract structure changed for %s.\n--- snapshot ---\n%s\n--- got ---\n%s\n"+
					"If this change is intentional, bump SchemaVersion when breaking and re-run with UPDATE_CONTRACT=1.",
					c.name, want, got)
			}
		})
	}
}

// TestContractFieldsPresent 断言每份报告都带 schemaVersion 与 kind —— 这是
// 消费方判别类型的唯一依据，缺了它契约就不成立。
func TestContractFieldsPresent(t *testing.T) {
	for _, c := range contractCases() {
		t.Run(c.name, func(t *testing.T) {
			doc, err := documentOf(c.kind, c.payload, false)
			if err != nil {
				t.Fatal(err)
			}
			// JSON 数字解出来是 float64，与 int 直接比会永远不等。
			if v, ok := doc["schemaVersion"].(float64); !ok || int(v) != SchemaVersion {
				t.Errorf("schemaVersion = %v, want %d", doc["schemaVersion"], SchemaVersion)
			}
			if doc["kind"] != c.kind {
				t.Errorf("kind = %v, want %q", doc["kind"], c.kind)
			}
		})
	}
}

// TestContractCompactIsSingleLine 断言紧凑输出是一行（管道友好），
// 且 --pretty 时是多行。
func TestContractCompactIsSingleLine(t *testing.T) {
	for _, c := range contractCases() {
		t.Run(c.name, func(t *testing.T) {
			var sb strings.Builder
			old := out
			out = &sb
			defer func() { out = old }()
			if err := emitJSON(c.kind, c.payload, false); err != nil {
				t.Fatal(err)
			}
			if n := strings.Count(strings.TrimRight(sb.String(), "\n"), "\n"); n != 0 {
				t.Errorf("compact JSON spans %d lines, want 1:\n%s", n+1, sb.String())
			}
		})
	}
}

// shapeOf renders the report as sorted "path: type" lines.
func shapeOf(kind string, payload any) ([]string, error) {
	doc, err := documentOf(kind, payload, false)
	if err != nil {
		return nil, err
	}
	var out []string
	flattenShape("", doc, &out)
	sort.Strings(out)
	return out, nil
}

// documentOf builds the emitted document (payload + contract fields) as a map.
func documentOf(kind string, payload any, pretty bool) (map[string]any, error) {
	var sb strings.Builder
	old := out
	out = &sb
	defer func() { out = old }()
	if err := emitJSON(kind, payload, pretty); err != nil {
		return nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(sb.String())), &doc); err != nil {
		return nil, fmt.Errorf("emitJSON output is not a JSON object: %w", err)
	}
	return doc, nil
}

// flattenShape walks the value and records "<path>: <type>" for every field.
// Arrays record their element type once ("tools[]: object") rather than each
// index, so the snapshot stays stable when counts change.
func flattenShape(prefix string, v any, out *[]string) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			flattenShape(p, t[k], out)
		}
	case []any:
		if len(t) == 0 {
			*out = append(*out, prefix+"[]: empty")
			return
		}
		// 元素类型取首个非空元素；对象元素递归记录其字段。
		switch elem := t[0].(type) {
		case map[string]any:
			flattenShape(prefix+"[]", elem, out)
		default:
			*out = append(*out, fmt.Sprintf("%s[]: %s", prefix, jsonTypeOf(elem)))
		}
	default:
		*out = append(*out, fmt.Sprintf("%s: %s", prefix, jsonTypeOf(t)))
	}
}

func jsonTypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return reflect.TypeOf(v).String()
}
