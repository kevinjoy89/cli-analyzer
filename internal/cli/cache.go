package cli

import (
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
)

func runCache(args []string) int {
	fs := flag.NewFlagSet("cache", flag.ContinueOnError)
	clear := fs.Bool("clear", false, "clear the scan cache")
	fs.SetOutput(stderr())
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if *clear {
		if err := scanner.ClearCache(); err != nil {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.cacheClearFailed", map[string]any{"err": err}))
			return 1
		}
		fmt.Fprintln(stdout(), i18n.T("cli.cacheCleared"))
		return 0
	}
	if ts, ok := scanner.CacheInfo(); ok {
		fmt.Fprint(stdout(), i18n.T("cli.cacheExists", map[string]any{"time": ts, "path": platform.CacheRoot()}), "\n")
	} else {
		fmt.Fprint(stdout(), i18n.T("cli.cacheNone", map[string]any{"path": platform.CacheRoot()}), "\n")
	}
	return 0
}
