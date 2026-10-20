package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/upgrade"
)

// runUpgrade 处理工具侧的版本检查与升级（CONTEXT.md：Upgrade = 工具升级）：
//   - `upgrade check <tool> [--json]`      检查工具是否有新版本
//   - `upgrade run <tool> [--yes] [--json]` 代跑工具官方升级命令
//
// 这两个动作原先是 `update check <tool>` / `update run <tool>`，与「Update =
// 应用自身更新」的术语定义冲突，故迁出；旧写法保留为别名（见 update.go）。
//
// 退出码：0 = 已是最新/成功、2 = 有更新或不可代跑、1 = 错误（脚本友好）。
func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
	yes := fs.Bool("yes", false, "skip interactive prompts")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if fs.NArg() < 2 {
		fmt.Fprintln(stderr(), i18n.T("cli.upgradeUsage"))
		return 1
	}
	switch fs.Arg(0) {
	case "check":
		return runToolCheck(fs.Arg(1), *jsonOut, *pretty)
	case "run":
		return runToolUpgrade(fs.Arg(1), *yes, *jsonOut, *pretty)
	default:
		fmt.Fprintln(stderr(), i18n.T("cli.upgradeUsage"))
		return 1
	}
}

// runToolCheck 检测单个工具是否有新版本。
func runToolCheck(tool string, jsonOut, pretty bool) int {
	res, err := upgrade.CheckToolByName(tool)
	if err != nil {
		msg := i18n.T("cli.noScanResult")
		if errors.Is(err, upgrade.ErrToolNotFound) {
			msg = i18n.T("up.toolNotFound")
		}
		if jsonOut {
			emitJSONError(KindToolCheck, map[string]any{"tool": tool, "message": msg}, pretty)
		} else {
			fmt.Fprintln(stderr(), msg)
		}
		return 1
	}
	if jsonOut {
		_ = emitJSON(KindToolCheck, res, pretty)
		return updateExitCode(res.HasUpdate)
	}
	if !res.Detected {
		fmt.Fprintln(stdout(), i18n.T("up.cantDetect", map[string]any{"cmd": res.Command}))
		return 0
	}
	if res.HasUpdate {
		fmt.Fprintln(stdout(), i18n.T("up.found", map[string]any{"tool": res.Name, "current": res.Current, "latest": res.Latest}))
		fmt.Fprintln(stdout(), i18n.T("up.command", map[string]any{"cmd": res.Command}))
		return updateExitCode(true)
	}
	fmt.Fprintln(stdout(), i18n.T("up.upToDate", map[string]any{"tool": res.Name}))
	return 0
}

// runToolUpgrade 代跑工具官方升级命令。
// 流程镜像 uninstall：展示命令 → 确认（--yes 跳过）→ 代跑。
func runToolUpgrade(name string, yes, jsonOut, pretty bool) int {
	res, err := scanner.LoadCache()
	if err != nil {
		if jsonOut {
			emitJSONError(KindToolUpgrade, map[string]any{"tool": name, "message": i18n.T("cli.noScanResult")}, pretty)
		} else {
			fmt.Fprintln(stderr(), i18n.T("cli.noScanResult"))
		}
		return 1
	}
	tool := findTool(res, name)
	if tool == nil {
		if jsonOut {
			emitJSONError(KindToolUpgrade, map[string]any{"tool": name, "message": i18n.T("up.toolNotFound")}, pretty)
		} else {
			fmt.Fprintln(stderr(), i18n.T("up.toolNotFound"))
		}
		return 2
	}
	cmd := upgrade.OfficialFor(*tool)
	if !cmd.Runnable {
		if jsonOut {
			_ = emitJSON(KindToolUpgrade, map[string]any{
				"tool": tool.Name, "installer": tool.Installer, "command": cmd.Command, "runnable": false,
			}, pretty)
		} else {
			fmt.Fprintf(stdout(), "%s\n", i18n.T("up.notRunnable", map[string]any{"cmd": cmd.Command}))
		}
		return 2
	}
	if jsonOut {
		// --json 契约：只输出信息，不执行（镜像 uninstall）
		_ = emitJSON(KindToolUpgrade, map[string]any{
			"tool": tool.Name, "installer": tool.Installer, "command": cmd.Command, "runnable": true,
		}, pretty)
		return 0
	}
	fmt.Fprintf(stdout(), "%s: %s\n", i18n.T("up.officialCmd"), cmd.Command)
	if !yes && !confirmPrompt(i18n.T("up.runPrompt")) {
		fmt.Fprintln(stdout(), i18n.T("up.skipRun"))
		return 0
	}
	if err := upgrade.RunOfficial(cmd, stdout(), stderr()); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintln(stderr(), i18n.T("up.runTimeout"))
		} else {
			fmt.Fprintf(stderr(), "%s: %v\n", i18n.T("up.runFailed"), err)
		}
		return 1
	}
	fmt.Fprintln(stdout(), i18n.T("up.runDone"))
	return 0
}
