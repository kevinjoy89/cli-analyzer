package cli

import (
	"flag"
	"fmt"

	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
)

func runCache(args []string) int {
	fs := flag.NewFlagSet("cache", flag.ContinueOnError)
	clear := fs.Bool("clear", false, "clear the scan cache")
	fs.SetOutput(stderr())
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *clear {
		if err := scanner.ClearCache(); err != nil {
			fmt.Fprintf(stderr(), "清除失败: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout(), "缓存已清除")
		return 0
	}
	if ts, ok := scanner.CacheInfo(); ok {
		fmt.Fprintf(stdout(), "缓存存在，写入时间 %s\n位置: %s\n", ts, platform.CacheRoot())
	} else {
		fmt.Fprintf(stdout(), "无缓存（位置: %s）\n", platform.CacheRoot())
	}
	return 0
}
