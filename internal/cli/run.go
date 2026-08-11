package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var out io.Writer = os.Stdout
var errOut io.Writer = os.Stderr

func stdout() io.Writer { return out }
func stderr() io.Writer { return errOut }

// Version is the CLI/GUI version string.
const Version = "0.2.0"

// Run dispatches CLI subcommands. args excludes argv[0]; the root main.go
// routes scan/clean/cache/version here and everything else to the Wails GUI.
func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}
	switch args[0] {
	case "scan":
		return runScan(args[1:])
	case "clean":
		return runClean(args[1:])
	case "cache":
		return runCache(args[1:])
	case "trash":
		return runTrash(args[1:])
	case "trends":
		return runTrends(args[1:])
	case "version", "--version", "-v":
		fmt.Fprintf(stdout(), "cli-analyzer %s\n", Version)
		return 0
	case "help", "-h", "--help":
		usage()
		return 0
	}
	fmt.Fprintf(stderr(), "unknown command %q\n", args[0])
	usage()
	return 1
}

func usage() {
	fmt.Fprintln(stdout(), `cli-analyzer — 扫描 CLI 工具的磁盘占用并安全清理

用法:
  cli-analyzer scan [-j|--json] [--full] [--refresh] [--order=size|name] [工具...]
  cli-analyzer clean [-a|--all] [-n|--dry-run] [-y|--yes] [-j|--json] [--list] [--permanent] [工具...]
  cli-analyzer cache [--clear]
  cli-analyzer trash [list|restore <id>|empty]
  cli-analyzer trends [天数]   # 查看占用趋势与可清理增量 Top 5（默认最近 30 天）
  cli-analyzer version
  cli-analyzer gui        # 打开图形界面（无参数时默认）

安全模型: 只有 SAFE 级（缓存/旧版本/包管理器缓存）可被清理；
USER 级（配置/历史/venv）仅展示，任何情况下都不会被自动删除。`)
}

// reorderFlags moves "-" prefixed arguments before positional ones so that
// `scan pylint --json` behaves like `scan --json pylint` (Go's flag package
// stops at the first non-flag argument).
func reorderFlags(args []string) []string {
	var flags, pos []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
		} else {
			pos = append(pos, a)
		}
	}
	return append(flags, pos...)
}
