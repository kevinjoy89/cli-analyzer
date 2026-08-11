package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/scanner"
)

func runClean(args []string) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	all := fs.Bool("a", false, "all cleanable items")
	fs.BoolVar(all, "all", false, "all cleanable items")
	dryRun := fs.Bool("n", false, "dry run (print plan, delete nothing)")
	fs.BoolVar(dryRun, "dry-run", false, "dry run")
	yes := fs.Bool("y", false, "assume yes to every item")
	fs.BoolVar(yes, "yes", false, "assume yes")
	jsonOut := fs.Bool("j", false, "output JSON report")
	fs.BoolVar(jsonOut, "json", false, "output JSON report")
	list := fs.Bool("list", false, "list cleanable items and exit")
	permanent := fs.Bool("permanent", false, "immediately delete, skip the built-in trash")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 1
	}
	filters := fs.Args()

	res, err := scanner.LoadCache()
	if err != nil {
		fmt.Fprintln(stderr(), "没有可用的扫描结果，请先运行 `cli-analyzer scan`")
		return 1
	}

	// Collect SAFE items, optionally filtered by tool.
	var items []scanner.Cleanable
	for _, t := range res.Tools {
		if len(filters) > 0 && !matchAny(t.Name, t.Aliases, filters) {
			continue
		}
		for _, c := range t.Cleanables {
			if c.Tier == scanner.TierSafe {
				items = append(items, c)
			}
		}
	}
	if *list {
		printCleanables(res, filters)
		return 0
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), "没有可安全清理的项目")
		return 0
	}

	var chosen []string
	if *all {
		for _, it := range items {
			chosen = append(chosen, it.ID)
		}
	} else {
		for _, it := range items {
			fmt.Fprintf(stdout(), "%s  %s  %s  %s — remove? [y/N] ",
				it.Tool, humanBytes(it.Bytes), it.Kind, it.Path)
			ok := *yes
			if !ok {
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				line = strings.ToLower(strings.TrimSpace(line))
				ok = line == "y" || line == "yes"
			}
			if ok {
				chosen = append(chosen, it.ID)
			}
		}
	}

	var report scanner.CleanReport
	if *permanent {
		report = cleaner.CleanPermanent(res, chosen, *dryRun)
	} else {
		report = cleaner.Clean(res, chosen, *dryRun)
	}
	if *jsonOut {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout(), string(b))
	} else {
		modeWord := "删除"
		if !*permanent && config.Load().Trash.UseTrash {
			modeWord = "移入回收站"
		}
		fmt.Fprintf(stdout(), "%s %d 项，释放 %s\n", modeWord, len(report.Deleted), humanBytes(report.Freed))
		for _, s := range report.Skipped {
			fmt.Fprintf(stdout(), "跳过: %s\n", s)
		}
		for _, e := range report.Errors {
			fmt.Fprintf(stdout(), "错误: %s\n", e)
		}
		if *dryRun {
			fmt.Fprintln(stdout(), "（dry-run：未实际删除）")
		}
	}
	return 0
}
