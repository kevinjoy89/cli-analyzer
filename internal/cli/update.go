package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
	"cli-analyzer/internal/updater"
	"cli-analyzer/internal/upgrade"
)

// runUpdate 处理：
//   - `update check [--json]`            检查应用自身更新（向后兼容）
//   - `update check <工具> [--json]`     检查工具是否有新版本
//   - `update run <工具> [--yes] [--json]` 代跑工具官方升级命令
//
// 退出码约定：0 = 已是最新/成功、2 = 有更新、1 = 错误（脚本友好）。
func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON")
	yes := fs.Bool("yes", false, "skip interactive prompts")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
		return 1
	}
	switch fs.Arg(0) {
	case "check":
		if fs.NArg() >= 2 {
			return runToolCheck(fs.Arg(1), *jsonOut)
		}
		return runAppCheck(*jsonOut)
	case "run":
		if fs.NArg() != 2 {
			fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
			return 1
		}
		return runToolUpgrade(fs.Arg(1), *yes, *jsonOut)
	default:
		fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
		return 1
	}
}

// runAppCheck 检查应用自身更新（原 `update check` 行为，不动）。
func runAppCheck(jsonOut bool) int {
	res := updater.CheckForUpdates(context.Background(), true)
	if res.Error != "" {
		if jsonOut {
			b, _ := json.Marshal(res)
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.updateCheckFailed", map[string]any{"err": res.Error}))
		}
		return 1
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout(), string(b))
		return updateExitCode(res.UpdateAvailable)
	}
	if res.UpdateAvailable {
		fmt.Fprintln(stdout(), i18n.T("cli.updateFound", map[string]any{"latest": res.Latest, "current": res.Current}))
		if res.DownloadURL != "" {
			fmt.Fprintln(stdout(), i18n.T("cli.updateDownload", map[string]any{"url": res.DownloadURL}))
		} else {
			fmt.Fprintln(stdout(), i18n.T("cli.updateReleasePage", map[string]any{"url": res.ReleaseURL}))
		}
		return updateExitCode(true)
	}
	fmt.Fprintln(stdout(), i18n.T("cli.updateUpToDate", map[string]any{"version": res.Latest}))
	return 0
}

// runToolCheck 检测单个工具是否有新版本（`update check <工具>`）。
func runToolCheck(tool string, jsonOut bool) int {
	res, err := upgrade.CheckToolByName(tool)
	if err != nil {
		msg := i18n.T("cli.noScanResult")
		if errors.Is(err, upgrade.ErrToolNotFound) {
			msg = i18n.T("up.toolNotFound")
		}
		if jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": tool, "error": msg})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintln(stderr(), msg)
		}
		return 1
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout(), string(b))
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

// runToolUpgrade 代跑工具官方升级命令（`update run <工具>`）。
// 流程镜像 uninstall：展示命令 → 确认（--yes 跳过）→ 代跑。
func runToolUpgrade(name string, yes, jsonOut bool) int {
	res, err := scanner.LoadCache()
	if err != nil {
		if jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": name, "error": i18n.T("cli.noScanResult")})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintln(stderr(), i18n.T("cli.noScanResult"))
		}
		return 1
	}
	tool := findTool(res, name)
	if tool == nil {
		if jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": name, "error": i18n.T("up.toolNotFound")})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintln(stderr(), i18n.T("up.toolNotFound"))
		}
		return 2
	}
	cmd := upgrade.OfficialFor(*tool)
	if !cmd.Runnable {
		if jsonOut {
			b, _ := json.Marshal(map[string]any{"tool": tool.Name, "installer": tool.Installer, "command": cmd.Command, "runnable": false})
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintf(stdout(), "%s\n", i18n.T("up.notRunnable", map[string]any{"cmd": cmd.Command}))
		}
		return 2
	}
	if jsonOut {
		// --json 契约：只输出信息，不执行（镜像 uninstall）
		b, _ := json.Marshal(map[string]any{
			"tool": tool.Name, "installer": tool.Installer, "command": cmd.Command, "runnable": true,
		})
		fmt.Fprintln(stdout(), string(b))
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

func updateExitCode(hasUpdate bool) int {
	if hasUpdate {
		return 2
	}
	return 0
}
