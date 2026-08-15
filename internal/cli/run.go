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
	case "uninstall":
		return runUninstall(args[1:])
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
