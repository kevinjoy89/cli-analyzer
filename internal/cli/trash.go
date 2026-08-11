package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

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
		return trashEmpty()
	}
	fmt.Fprintf(stderr(), "未知的 trash 子命令 %q\n", args[0])
	trashUsage()
	return 1
}

// trashUsage 打印 trash 子命令的用法
func trashUsage() {
	fmt.Fprintln(stdout(), `cli-analyzer trash — 管理内置回收站

用法:
  cli-analyzer trash list              # 列出回收站项目
  cli-analyzer trash restore <id>      # 恢复一个项目到原路径
  cli-analyzer trash empty             # 清空回收站（彻底删除）`)
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
		fmt.Fprintf(stderr(), "读取回收站失败: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), "回收站为空")
		return 0
	}
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\t类型\t大小\t移入时间\t到期时间\t原路径")
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
		fmt.Fprintf(stderr(), "恢复失败: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout(), "已恢复到 %s\n", restored)
	return 0
}

// trashEmpty 清空回收站（彻底删除全部项目）
func trashEmpty() int {
	items, err := trash.List()
	if err != nil {
		fmt.Fprintf(stderr(), "读取回收站失败: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout(), "回收站为空")
		return 0
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	deleted, errs := trash.Purge(ids)
	fmt.Fprintf(stdout(), "已清空 %d 项\n", len(deleted))
	for _, e := range errs {
		fmt.Fprintf(stderr(), "错误: %s\n", e)
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}
