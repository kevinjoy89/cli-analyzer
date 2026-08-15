package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/updater"
)

// runUpdate 处理 `update check [--json]`。
// 退出码约定：0 = 已是最新、2 = 有更新、1 = 错误（脚本友好）。
// 手动检查不受 4h 缓存限制；CLI 不做下载/安装（那是 GUI 交互场景，design D8）。
func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if fs.NArg() == 0 || fs.Arg(0) != "check" {
		fmt.Fprintln(stderr(), i18n.T("cli.updateUsage"))
		return 1
	}

	res := updater.CheckForUpdates(context.Background(), true)
	if res.Error != "" {
		if *jsonOut {
			b, _ := json.Marshal(res)
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.updateCheckFailed", map[string]any{"err": res.Error}))
		}
		return 1
	}

	if *jsonOut {
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

func updateExitCode(hasUpdate bool) int {
	if hasUpdate {
		return 2
	}
	return 0
}
