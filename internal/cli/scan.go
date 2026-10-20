package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"cli-analyzer/internal/history"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/probe"
	"cli-analyzer/internal/rules"
	"cli-analyzer/internal/scanner"
)

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
	refresh := fs.Bool("refresh", false, "ignore the cache and rescan")
	noCache := fs.Bool("no-cache", false, "do not write the cache")
	order := fs.String("order", "size", "sort order: size|cleanable|name")
	top := fs.Int("top", 0, "only output the top N tools (0 = all)")
	baselinePath := fs.String("baseline", "", "compare against a baseline `scan --json` file")
	failOnGrow := fs.String("fail-on-grow", "", "exit non-zero when any tool's cleanable grows by more than this size (needs --baseline)")
	fs.SetOutput(stderr())
	// 带值的 flag 必须告诉 reorderFlags，否则 `scan --baseline b.json --json`
	// 会被拆成 `--baseline --json b.json`（基线参数吞掉下一个 flag）。
	valueFlags := map[string]bool{"order": true, "top": true, "baseline": true, "fail-on-grow": true}
	if err := fs.Parse(reorderFlagsWithValue(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	filters := fs.Args()

	// --fail-on-grow 只能与 --baseline 搭配：没有基线就无所谓「增长」。
	if *failOnGrow != "" && *baselinePath == "" {
		fmt.Fprintln(stderr(), i18n.T("cli.failOnGrowNeedsBaseline"))
		return 1
	}
	var threshold int64
	if *failOnGrow != "" {
		v, err := parseSize(*failOnGrow)
		if err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.badSize", map[string]any{"value": *failOnGrow}))
			return 1
		}
		threshold = v
	}

	res, code := scanNow(*refresh, *noCache, filters)
	if code != 0 {
		return code
	}
	if len(filters) > 0 {
		res = filterResult(res, filters)
	}

	sortTools(res, *order)
	// 健康探测：缓存优先，总预算 2s（挂起工具按 3s 单条超时中断，不会拖垮输出）
	probe.FillVersions(res.Tools, 2*time.Second)

	// --baseline：输出增量而不是整份扫描结果。基线文件就是一份 `scan --json`。
	if *baselinePath != "" {
		return reportBaseline(res, *baselinePath, threshold, *jsonOut, *pretty)
	}

	if *top > 0 && *top < len(res.Tools) {
		res.OmittedTools = len(res.Tools) - *top
		res.Tools = res.Tools[:*top]
	}

	if *jsonOut {
		if err := emitJSON(KindScan, res, *pretty); err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.scanFailed", map[string]any{"err": err}))
			return 1
		}
		return 0
	}
	printTable(res)
	if res.OmittedTools > 0 {
		fmt.Fprint(stdout(), i18n.T("cli.omittedTools", map[string]any{"n": res.OmittedTools}))
	}
	if len(res.RuleErrors) > 0 {
		fmt.Fprint(stdout(), i18n.T("cli.ruleErrors", map[string]any{"n": len(res.RuleErrors), "dir": rules.UserRulesDir()}))
		for _, e := range res.RuleErrors {
			fmt.Fprintf(stdout(), "  %s\n", e)
		}
	}
	return 0
}

// scanNow runs the scan honouring the cache semantics and appends a history
// snapshot for real (unfiltered) scans. Returns a non-zero code on failure.
func scanNow(refresh, noCache bool, filters []string) (*scanner.ScanResult, int) {
	if refresh {
		res, err := scanner.Scan(scanner.Options{NoCache: noCache, ToolFilter: filters})
		if err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.scanFailed", map[string]any{"err": err}))
			return nil, 1
		}
		// 只有真实扫描才追加历史；命中缓存时已有记录，不重复写入
		// 过滤扫描（`scan <filter> --refresh`）返回的是过滤后的 totals，
		// 写入历史会污染整体趋势/增量排行数据（此前无条件 Record）
		if len(filters) == 0 {
			_ = history.Record(res)
		}
		return res, 0
	}
	// 非 --refresh：指纹未变化直接返回缓存（秒回）；变化则自动全量。
	// --no-cache 时跳过指纹捷径（调用方明确不要缓存语义），视为真实扫描。
	opts := scanner.Options{NoCache: noCache, ToolFilter: filters}
	scanned := true
	var res *scanner.ScanResult
	var err error
	if noCache {
		res, err = scanner.Scan(opts)
	} else {
		res, scanned, err = scanner.ScanIfUnchanged(opts)
	}
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.scanFailed", map[string]any{"err": err}))
		return nil, 1
	}
	// 真实扫描（含指纹变化触发的自动重扫与 --no-cache）才追加历史，
	// 与 --refresh 路径一致；过滤扫描不写（避免污染整体趋势/增量排行）。
	if scanned && len(filters) == 0 {
		_ = history.Record(res)
	}
	return res, 0
}

// sortTools orders tools by the requested key. Unknown keys fall back to size.
func sortTools(res *scanner.ScanResult, order string) {
	switch order {
	case "name":
		sort.Slice(res.Tools, func(i, j int) bool { return res.Tools[i].Name < res.Tools[j].Name })
	case "cleanable":
		sort.SliceStable(res.Tools, func(i, j int) bool { return res.Tools[i].Cleanable > res.Tools[j].Cleanable })
	default:
		sort.SliceStable(res.Tools, func(i, j int) bool { return res.Tools[i].Footprint > res.Tools[j].Footprint })
	}
}

// reportBaseline prints the diff between this scan and a baseline file, and
// returns a non-zero exit code when --fail-on-grow is exceeded.
func reportBaseline(res *scanner.ScanResult, path string, threshold int64, jsonOut, pretty bool) int {
	base, err := loadBaseline(path)
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.baselineFailed", map[string]any{"path": path, "err": err}))
		return 1
	}
	diff := diffAgainst(res, base, threshold)
	if jsonOut {
		_ = emitJSON(KindBaseline, diff, pretty)
	} else {
		printBaselineDiff(diff)
	}
	if threshold > 0 && len(diff.Exceeded) > 0 {
		return 1
	}
	return 0
}

// loadBaseline reads a previously saved `scan --json` document.
func loadBaseline(path string) (*scanner.ScanResult, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var res scanner.ScanResult
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, err
	}
	// 一份合法的基线至少有 tools 键（空数组也算）；完全不含该键说明这不是
	// scan 的输出（或版本不兼容），静默当作空基线会给出看似正常的增量。
	if res.Tools == nil {
		return nil, fmt.Errorf("not a scan document")
	}
	return &res, nil
}

// filterResult keeps only tools matching the filters and recomputes totals.
func filterResult(res *scanner.ScanResult, filters []string) *scanner.ScanResult {
	var out []scanner.Tool
	totals := scanner.Totals{}
	for _, t := range res.Tools {
		if !matchAny(t.Name, t.Aliases, filters) {
			continue
		}
		out = append(out, t)
		totals.Footprint += t.Footprint
		totals.Cleanable += t.Cleanable
		totals.User += t.User
	}
	res.Tools = out
	res.Totals = totals
	return res
}
