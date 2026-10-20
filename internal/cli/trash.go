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
		return trashList(args[1:])
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
  cli-analyzer trash list [--json]           # 列出回收站项目
  cli-analyzer trash restore <id> [--json]   # 恢复一个项目到原路径
  cli-analyzer trash empty [--yes] [--json]  # 清空回收站（永久删除，需确认；--yes 跳过确认）

通用选项:
  --json / -j   输出单行 JSON（含 schemaVersion 与 kind）
  --pretty      JSON 缩进输出`)
}

// trashOpts 是 trash 三个子命令共用的输出选项。
type trashOpts struct {
	jsonOut bool
	pretty  bool
	yes     bool
}

// parseTrashFlags 解析共用选项，返回位置参数。
func parseTrashFlags(name string, args []string) (trashOpts, []string, int, bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	var o trashOpts
	fs.BoolVar(&o.jsonOut, "j", false, "output JSON")
	fs.BoolVar(&o.jsonOut, "json", false, "output JSON")
	fs.BoolVar(&o.pretty, "pretty", false, "indent JSON output")
	fs.BoolVar(&o.yes, "yes", false, "skip confirmation (permanent delete)")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return o, nil, 0, false // 帮助请求不是错误
		}
		return o, nil, 1, false
	}
	return o, fs.Args(), 0, true
}

// fmtTrashTime 将 RFC3339 时间压缩为 "2006-01-02 15:04" 便于终端阅读
func fmtTrashTime(ts string) string {
	if len(ts) < 16 {
		return ts
	}
	return strings.Replace(ts[:16], "T", " ", 1)
}

// TrashReport is the machine-readable contract of `trash list`.
type TrashReport struct {
	Items []trash.Item `json:"items"`
}

// trashList 打印回收站项目列表
func trashList(args []string) int {
	o, _, code, ok := parseTrashFlags("list", args)
	if !ok {
		return code
	}
	items, err := trash.List()
	if err != nil {
		if o.jsonOut {
			emitJSONError(KindTrashList, map[string]any{"message": err.Error()}, o.pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashReadFailed", map[string]any{"err": err}))
		}
		return 1
	}
	if o.jsonOut {
		if items == nil {
			items = []trash.Item{}
		}
		_ = emitJSON(KindTrashList, TrashReport{Items: items}, o.pretty)
		return 0
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
	o, rest, code, ok := parseTrashFlags("restore", args)
	if !ok {
		return code
	}
	if len(rest) != 1 {
		trashUsage()
		return 1
	}
	restored, err := trash.Restore(rest[0])
	if err != nil {
		if o.jsonOut {
			emitJSONError(KindTrashRestore, map[string]any{"id": rest[0], "message": err.Error()}, o.pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashRestoreFailed", map[string]any{"err": err}))
		}
		return 1
	}
	// 恢复后核对：清单在 Restore 之前读（项目目录随后会被删）。
	manifest, hasManifest := trash.ManifestOf(rest[0])
	verify := trash.VerifyItem(restored, manifest, hasManifest)
	if o.jsonOut {
		_ = emitJSON(KindTrashRestore, map[string]any{"id": rest[0], "restored": restored, "verify": verify}, o.pretty)
		return 0
	}
	fmt.Fprintln(stdout(), i18n.T("cli.trashRestored", map[string]any{"path": restored}))
	if !verify.Checked {
		fmt.Fprintln(stdout(), i18n.T("cli.trashRestoreUnverified"))
	} else if verify.OK {
		fmt.Fprintln(stdout(), i18n.T("cli.trashRestoreVerified", map[string]any{"n": verify.Files}))
	} else {
		fmt.Fprintln(stdout(), i18n.T("cli.trashRestoreMismatch", map[string]any{
			"got": verify.Files, "want": verify.ExpectedFiles,
		}))
		for _, m := range verify.Missing {
			fmt.Fprintln(stdout(), i18n.T("cli.trashRestoreMissing", map[string]any{"name": m}))
		}
	}
	return 0
}

// trashEmpty 清空回收站（永久删除全部项目）。
// 破坏性操作需要用户确认（--yes 跳过确认，供脚本使用）；
// 永久删除语义与 README / GUI "Delete permanently" 契约一致
// （此前复用过期配置，默认只是转系统回收站，空间并未释放）。
func trashEmpty(args []string) int {
	o, rest, code, ok := parseTrashFlags("empty", args)
	if !ok {
		return code
	}
	if len(rest) > 0 {
		trashUsage()
		return 1
	}
	items, err := trash.List()
	if err != nil {
		if o.jsonOut {
			emitJSONError(KindTrashEmpty, map[string]any{"message": err.Error()}, o.pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trashReadFailed", map[string]any{"err": err}))
		}
		return 1
	}
	if len(items) == 0 {
		if o.jsonOut {
			_ = emitJSON(KindTrashEmpty, map[string]any{"deleted": []string{}, "errors": []string{}}, o.pretty)
			return 0
		}
		fmt.Fprintln(stdout(), i18n.T("cli.trashEmpty"))
		return 0
	}
	if !o.yes {
		if o.jsonOut {
			// --json 不交互：未加 --yes 时拒绝执行并说明，而不是等 stdin
			emitJSONError(KindTrashEmpty, map[string]any{
				"message": i18n.T("cli.trashEmptyNeedsYes"), "pending": len(items),
			}, o.pretty)
			return 1
		}
		if !confirmPrompt(i18n.T("cli.trashEmptyConfirm", map[string]any{"n": len(items)})) {
			fmt.Fprintln(stdout(), i18n.T("cli.cancelled"))
			return 0
		}
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	deleted, errs := trash.Purge(ids)
	if o.jsonOut {
		if deleted == nil {
			deleted = []string{}
		}
		if errs == nil {
			errs = []string{}
		}
		_ = emitJSON(KindTrashEmpty, map[string]any{"deleted": deleted, "errors": errs}, o.pretty)
		if len(errs) > 0 {
			return 1
		}
		return 0
	}
	fmt.Fprintln(stdout(), i18n.T("cli.trashEmptied", map[string]any{"n": len(deleted)}))
	for _, e := range errs {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": e}))
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}
