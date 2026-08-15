package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"text/tabwriter"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/trash"
)

// runTrash 分派回收站子命令：list / restore / empty
func runTrash(args []string) int {
	if len(args) == 0 {
		trashUsage()
		return 0
	}
	switch args[0] {
	case "list":
		return trashList()
	case "restore":
		return trashRestore(args[1:])
	case "empty":
		return trashEmpty(args[1:])
	case "help", "-h", "--help":
		trashUsage()
		return 0
	}
	fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashUnknown", map[string]any{"cmd": args[0]}))
	trashUsage()
	return 1
}

// trashUsage 打印 trash 子命令的用法
func trashUsage() {
	fmt.Fprintln(stdout(), `cli-analyzer trash — 管理内置回收站

用法:
  cli-analyzer trash list              # 列出回收站项目
  cli-analyzer trash restore <id>      # 恢复一个项目到原路径
  cli-analyzer trash empty [--yes]     # 清空回收站（永久删除，需确认；--yes 跳过确认）`)
}

// fmtTrashTime 将 RFC3339 时间压缩为 "2006-01-02 15:04" 便于终端阅读
func fmtTrashTime(ts string) string {
	if len(ts) < 16 {
		return ts
	}
	return strings.Replace(ts[:16], "T", " ", 1)
}

// trashList 打印回收站项目列表
func trashList() int {
	items, err := trash.List()
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashReadFailed", map[string]any{"err": err}))
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.trashEmpty"))
		return 0
	}
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.trashHeader"))
	for _, it := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			it.ID, it.Kind, humanBytes(it.Bytes), fmtTrashTime(it.TrashedAt), fmtTrashTime(it.ExpiresAt), it.Original)
	}
	w.Flush()
	return 0
}

// trashRestore 恢复一个回收站项目到原路径
func trashRestore(args []string) int {
	if len(args) != 1 {
		trashUsage()
		return 1
	}
	restored, err := trash.Restore(args[0])
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashRestoreFailed", map[string]any{"err": err}))
		return 1
	}
	fmt.Fprintln(stdout(), i18n.T("cli.trashRestored", map[string]any{"path": restored}))
	return 0
}

// trashEmpty 清空回收站（永久删除全部项目）。
// 破坏性操作需要用户确认（--yes 跳过确认，供脚本使用）；
// 永久删除语义与 README / GUI "Delete permanently" 契约一致
// （此前复用过期配置，默认只是转系统回收站，空间并未释放）。
func trashEmpty(args []string) int {
	fs := flag.NewFlagSet("empty", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation (permanent delete)")
	fs.SetOutput(stderr())
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if fs.NArg() > 0 {
		trashUsage()
		return 1
	}
	items, err := trash.List()
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashReadFailed", map[string]any{"err": err}))
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.trashEmpty"))
		return 0
	}
	if !*yes && !confirmPrompt(i18n.T("cli.trashEmptyConfirm", map[string]any{"n": len(items)})) {
		fmt.Fprintln(stdout(), i18n.T("cli.cancelled"))
		return 0
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	deleted, errs := trash.Purge(ids)
	fmt.Fprintln(stdout(), i18n.T("cli.trashEmptied", map[string]any{"n": len(deleted)}))
	for _, e := range errs {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}
