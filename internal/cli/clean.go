package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

func runClean(args []string) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	all := fs.Bool("a", false, "all actionable items")
	fs.BoolVar(all, "all", false, "all actionable items")
	dryRun := fs.Bool("n", false, "dry run (print plan, delete nothing)")
	fs.BoolVar(dryRun, "dry-run", false, "dry run")
	yes := fs.Bool("y", false, "assume yes to every item")
	fs.BoolVar(yes, "yes", false, "assume yes")
	jsonOut := fs.Bool("j", false, "output JSON report")
	fs.BoolVar(jsonOut, "json", false, "output JSON report")
	list := fs.Bool("list", false, "list actionable items and exit")
	permanent := fs.Bool("permanent", false, "immediately delete, skip the built-in trash")
	includeData := fs.Bool("include-data", false, "include config/data/state/toolchain items in --all")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	filters := fs.Args()

	res, err := scanner.LoadCache()
	if err != nil {
		fmt.Fprintln(stderr(), i18n.T("cli.noScanResult"))
		return 1
	}

	// Collect actionable items (every attributed dir, any tier), optionally
	// filtered by tool. Tier is an informational label, not a gate.
	var items []scanner.Cleanable
	for _, t := range res.Tools {
		if len(filters) > 0 && !matchAny(t.Name, t.Aliases, filters) {
			continue
		}
		items = append(items, t.Cleanables...)
	}
	if *list {
		printCleanables(res, filters)
		return 0
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.noCleanable"))
		return 0
	}

	// --all 的保护性默认：只批量选择"清理类"（缓存/旧版本/备份/下载），
	// config/data/state/toolchain 等"数据类"需 --include-data 或逐项指定 id。
	// 这是防误操作的默认值，不是门槛——用户随时可以显式覆盖。
	cleanKinds := map[string]bool{"cache": true, "old-version": true, "backup": true, "download": true}
	if *includeData {
		for _, k := range []string{"config", "data", "state", "toolchain", "logs"} {
			cleanKinds[k] = true
		}
	}

	var chosen []string
	if *all {
		for _, it := range items {
			if cleanKinds[it.Kind] {
				chosen = append(chosen, it.ID)
			}
		}
		if len(chosen) == 0 {
			// 批量选择未命中任何"清理类"项（数据类需 --include-data），
			// 明确提示而非静默清 0 项。
			fmt.Fprintln(stdout(), i18n.T("cli.cleanAllEmpty"))
			return 0
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
		modeWord := i18n.T("cli.delete")
		if !*permanent && config.Load().Trash.UseTrash {
			modeWord = i18n.T("cli.moveToTrash")
		}
		fmt.Fprint(stdout(), i18n.T("cli.cleanSummary", map[string]any{"action": modeWord, "n": len(report.Deleted), "size": humanBytes(report.Freed)}), "\n")
		for _, s := range report.Skipped {
			fmt.Fprintf(stdout(), "%s\n", i18n.T("cli.skipped", map[string]any{"path": s}))
		}
		for _, e := range report.Errors {
			fmt.Fprintf(stdout(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
		}
		if *dryRun {
			fmt.Fprintln(stdout(), i18n.T("cli.dryRun"))
		}
	}
	return 0
}
