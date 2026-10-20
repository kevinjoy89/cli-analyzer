package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/updater"
)

// runUpdate 处理 cli-analyzer 自身的更新检查（`update check`）。
//
// 工具侧的版本检查与升级由 `upgrade` 子命令承载（见 upgrade.go）：CONTEXT.md
// 把 Update 定义为「应用自身更新」、Upgrade 定义为「工具升级」，一个动词干两件
// 事会持续误导。旧写法 `update check <tool>` / `update run <tool>` 仍可用，
// 但会向 stderr 输出 deprecation 提示。
//
// 退出码约定：0 = 已是最新/成功、2 = 有更新、1 = 错误（脚本友好）。
func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
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
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
		return 1
	}
	switch fs.Arg(0) {
	case "check":
		if fs.NArg() >= 2 {
			// 兼容别名：工具版本检查归 upgrade 管
			deprecation("update check <tool>", "upgrade check <tool>")
			return runToolCheck(fs.Arg(1), *jsonOut, *pretty)
		}
		return runAppCheck(*jsonOut, *pretty)
	case "run":
		if fs.NArg() != 2 {
			fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
			return 1
		}
		deprecation("update run <tool>", "upgrade run <tool>")
		return runToolUpgrade(fs.Arg(1), *yes, *jsonOut, *pretty)
	default:
		fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
		return 1
	}
}

// deprecation prints a one-line pointer to the replacement command on stderr
// (never stdout: stdout stays a single clean JSON document under --json).
func deprecation(old, new string) {
	fmt.Fprintln(stderr(), i18n.T("cli.deprecated", map[string]any{"old": old, "new": new}))
}

// runAppCheck 检查应用自身更新。
func runAppCheck(jsonOut, pretty bool) int {
	res := updater.CheckForUpdates(context.Background(), true)
	if res.Error != "" {
		if jsonOut {
			emitJSONError(KindAppUpdate, map[string]any{"message": res.Error}, pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.updateCheckFailed", map[string]any{"err": res.Error}))
		}
		return 1
	}
	if jsonOut {
		_ = emitJSON(KindAppUpdate, res, pretty)
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

func updateExitCode(hasUpdate bool) int {
	if hasUpdate {
		return 2
	}
	return 0
}
