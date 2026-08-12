package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"cli-analyzer/internal/updater"
)

// runUpdate 处理 `update check [--json]`。
// 退出码约定：0 = 已是最新、2 = 有更新、1 = 错误（脚本友好）。
// 手动检查不受 24h 缓存限制；CLI 不做下载/安装（那是 GUI 交互场景，design D8）。
func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "output JSON")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		return 1
	}
	if fs.NArg() == 0 || fs.Arg(0) != "check" {
		fmt.Fprintln(stderr(), "用法: cli-analyzer update check [--json]")
		return 1
	}

	res := updater.CheckForUpdates(context.Background(), true)
	if res.Error != "" {
		if *jsonOut {
			b, _ := json.Marshal(res)
			fmt.Fprintln(stdout(), string(b))
		} else {
			fmt.Fprintf(stderr(), "检查更新失败: %s\n", res.Error)
		}
		return 1
	}

	if *jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(stdout(), string(b))
		return updateExitCode(res.UpdateAvailable)
	}

	if res.UpdateAvailable {
		fmt.Fprintf(stdout(), "发现新版本: v%s（当前 v%s）\n", res.Latest, res.Current)
		if res.DownloadURL != "" {
			fmt.Fprintf(stdout(), "下载: %s\n", res.DownloadURL)
		} else {
			fmt.Fprintf(stdout(), "请访问 Release 页面获取安装包: %s\n", res.ReleaseURL)
		}
		return updateExitCode(true)
	}
	fmt.Fprintf(stdout(), "已是最新版本 v%s\n", res.Latest)
	return 0
}

func updateExitCode(hasUpdate bool) int {
	if hasUpdate {
		return 2
	}
	return 0
}
