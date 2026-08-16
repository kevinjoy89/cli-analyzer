package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"sort"
	"time"

	"cli-analyzer/internal/history"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/probe"
	"cli-analyzer/internal/scanner"
)

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	refresh := fs.Bool("refresh", false, "ignore the cache and rescan")
	noCache := fs.Bool("no-cache", false, "do not write the cache")
	order := fs.String("order", "size", "sort order: size|name")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	filters := fs.Args()

	var res *scanner.ScanResult
	if *refresh {
		var err error
		res, err = scanner.Scan(scanner.Options{NoCache: *noCache, ToolFilter: filters})
		if err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.scanFailed", map[string]any{"err": err}))
			return 1
		}
		// 只有真实扫描才追加历史；命中缓存时已有记录，不重复写入
		// 过滤扫描（`scan <filter> --refresh`）返回的是过滤后的 totals，
		// 写入历史会污染整体趋势/增量排行数据（此前无条件 Record）
		if len(filters) == 0 {
			_ = history.Record(res)
		}
	} else {
		// 非 --refresh：指纹未变化直接返回缓存（秒回）；变化则自动全量。
		// --no-cache 时跳过指纹捷径（调用方明确不要缓存语义）。
		opts := scanner.Options{NoCache: *noCache, ToolFilter: filters}
		var err error
		if *noCache {
			res, err = scanner.Scan(opts)
		} else {
			res, err = scanner.ScanIfUnchanged(opts)
		}
		if err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.scanFailed", map[string]any{"err": err}))
			return 1
		}
	}
	if len(filters) > 0 {
		res = filterResult(res, filters)
	}

	if *order == "name" {
		sort.Slice(res.Tools, func(i, j int) bool { return res.Tools[i].Name < res.Tools[j].Name })
	}
	// 健康探测：缓存优先，总预算 2s（挂起工具按 3s 单条超时中断，不会拖垮输出）
	probe.FillVersions(res.Tools, 2*time.Second)
	if *jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout(), string(b))
		return 0
	}
	printTable(res)
	return 0
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
