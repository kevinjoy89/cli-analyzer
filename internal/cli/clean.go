package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"cli-analyzer/internal/cleaner"
	"cli-analyzer/internal/cleanpolicy"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

// includeDataKinds 是 --include-data 纳入批量的「数据类」Kind。
// 与 cleanpolicy 的内置白名单正交：前者是用户显式要求纳入，后者是默认范围。
var includeDataKinds = map[string]bool{
	"config": true, "data": true, "state": true, "toolchain": true,
	"runtime": true, "session": true, "plugins": true, "artifacts": true,
	"checkpoint": true, "workspace-data": true,
}

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
	pretty := fs.Bool("pretty", false, "indent JSON output")
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
		if *jsonOut {
			// --list --json：列出可处置项（不处置）
			_ = emitJSON(KindClean, map[string]any{"dryRun": true, "list": items}, *pretty)
			return 0
		}
		printCleanables(res, filters)
		return 0
	}
	if len(items) == 0 {
		// --json 下也必须输出 JSON（契约：stdout 只含一份 JSON 文档）。
		// 这是第二条纯文本分支，早先同样违反了契约。
		if *jsonOut {
			_ = emitJSON(KindClean, scanner.CleanReport{
				DryRun:  *dryRun,
				Deleted: []string{},
				Skipped: []scanner.SkippedItem{},
				Errors:  []string{},
			}, *pretty)
			return 0
		}
		fmt.Fprintln(stdout(), i18n.T("cli.noCleanable"))
		return 0
	}

	// --all 的保护性默认：只批量选择「可批量清理」的 Kind。判定交给
	// cleanpolicy（CLI 与 GUI 共用，见 ADR 0005）——Tier 不能当批量资格用：
	// runtime 是 safe 但删了会立即破坏工具可用性。
	//
	// --include-data 仍保留：它把 user 类也纳入批量（config/data/state/
	// toolchain/runtime/session/plugins/artifacts/checkpoint/workspace-data），
	// 与 Kind 白名单是两条正交的途径。
	cfg := config.Load()
	bulkOK := func(kind string) bool {
		if cleanpolicy.BulkEligible(kind, cfg.Clean) {
			return true
		}
		return *includeData && includeDataKinds[kind]
	}

	var chosen []string
	if *all {
		for _, it := range items {
			if bulkOK(it.Kind) {
				chosen = append(chosen, it.ID)
			}
		}
		if len(chosen) == 0 {
			// 批量选择未命中任何可批量清理项（数据类需 --include-data）。
			// --json 下必须仍输出 JSON：契约是「stdout 只含一份 JSON 文档」，
			// 这里早先直接打纯文本，导致脚本拿到非 JSON 而解析失败。
			if *jsonOut {
				_ = emitJSON(KindClean, scanner.CleanReport{
					DryRun:  *dryRun,
					Deleted: []string{},
					Skipped: []scanner.SkippedItem{},
					Errors:  []string{},
				}, *pretty)
				return 0
			}
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
		_ = emitJSON(KindClean, report, *pretty)
	} else {
		modeWord := i18n.T("cli.delete")
		if !*permanent && config.Load().Trash.UseTrash {
			modeWord = i18n.T("cli.moveToTrash")
		}
		fmt.Fprint(stdout(), i18n.T("cli.cleanSummary", map[string]any{"action": modeWord, "n": len(report.Deleted), "size": humanBytes(report.Freed)}), "\n")
		for _, s := range report.Skipped {
			fmt.Fprintf(stdout(), "%s\n", i18n.T("cli.skipped", map[string]any{"path": s.Path, "reason": s.Message}))
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
