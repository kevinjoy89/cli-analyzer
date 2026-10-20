package cli

import (
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"strings"

	"cli-analyzer/internal/buildinfo"
	"cli-analyzer/internal/config"
	"cli-analyzer/internal/i18n"
)

var out io.Writer = os.Stdout
var errOut io.Writer = os.Stderr

// stdin 供交互确认读取；变量形式便于测试注入（空输入 = 拒绝）
var stdin io.Reader = os.Stdin

func stdout() io.Writer { return out }
func stderr() io.Writer { return errOut }

// Version is the CLI/GUI version string, sourced from buildinfo (single source).
var Version = buildinfo.Version

// Run dispatches CLI subcommands. args excludes argv[0]; the root main.go
// routes scan/clean/cache/version here and everything else to the Wails GUI.
func Run(args []string) int {
	// CLI 无 WebView：语言由显式配置 + 系统探测解析（设计 D2）
	i18n.SetLocale(i18n.Resolve(config.Load().Language))
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
	case "update":
		return runUpdate(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "uninstall":
		return runUninstall(args[1:])
	case "residue":
		return runResidue(args[1:])
	case "version", "--version", "-v":
		fmt.Fprintf(stdout(), "cli-analyzer %s (%s, %s)\n", Version, goruntime.GOOS, buildinfo.InstallSource)
		return 0
	case "help", "-h", "--help":
		usage()
		return 0
	}
	fmt.Fprintf(stderr(), "%s %q\n", i18n.T("cli.unknownCommand"), args[0])
	usage()
	return 1
}

func usage() {
	fmt.Fprintln(stdout(), i18n.T("cli.usage"))
}

// Subcommands 返回 cli 认识的子命令名（不含 help/version 这类元命令）。
// 供 main.go 的白名单测试守住「新增子命令忘了加进分发」这类漏发。
func Subcommands() []string {
	return []string{"scan", "clean", "cache", "trash", "trends", "update", "upgrade", "uninstall", "residue"}
}

// UsageMentions 报告 usage 文本里是否提到了某个子命令。
// 用于测试：新增子命令时忘了写进用法会红。
func UsageMentions(name string) bool {
	return strings.Contains(i18n.T("cli.usage"), "cli-analyzer "+name)
}

// reorderFlags moves "-" prefixed arguments before positional ones so that
// `scan pylint --json` behaves like `scan --json pylint` (Go's flag package
// stops at the first non-flag argument).
func reorderFlags(args []string) []string {
	return reorderFlagsWithValue(args, nil)
}

// reorderFlagsWithValue is reorderFlags that knows which flags take a value.
// Without that knowledge a value like `--baseline base.json` gets split: the
// flag moves up, `base.json` stays positional, and the flag then swallows the
// next flag as its value (`--baseline --json`). valueFlags holds the flag
// names (without dashes) that consume the following token.
func reorderFlagsWithValue(args []string, valueFlags map[string]bool) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		// `--flag=value` 自带值；`--flag value` 需要把下一个 token 一起带上。
		name := strings.TrimLeft(a, "-")
		if !strings.Contains(name, "=") && valueFlags[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, pos...)
}
