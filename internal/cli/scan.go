package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"

	"cli-analyzer/internal/history"
	"cli-analyzer/internal/scanner"
)

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	full := fs.Bool("full", false, "also measure unmatched dirs (slow)")
	refresh := fs.Bool("refresh", false, "ignore the cache and rescan")
	noCache := fs.Bool("no-cache", false, "do not write the cache")
	order := fs.String("order", "size", "sort order: size|name")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 1
	}
	filters := fs.Args()

	var res *scanner.ScanResult
	if !*refresh {
		if cached, err := scanner.LoadCache(); err == nil {
			res = cached
		}
	}
	if res == nil {
		var err error
		res, err = scanner.Scan(scanner.Options{Full: *full, NoCache: *noCache, ToolFilter: filters})
		if err != nil {
			fmt.Fprintf(stderr(), "scan failed: %v\n", err)
			return 1
		}
		// 只有真实扫描才追加历史；命中缓存时已有记录，不重复写入
		_ = history.Record(res)
	} else if len(filters) > 0 {
		res = filterResult(res, filters)
	}

	if *order == "name" {
		sort.Slice(res.Tools, func(i, j int) bool { return res.Tools[i].Name < res.Tools[j].Name })
	}
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
