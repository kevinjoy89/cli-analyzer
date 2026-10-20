package cli

import (
	"errors"
	"flag"
	"fmt"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/scanner"
)

// CacheReport is the machine-readable contract of the `cache` subcommand.
type CacheReport struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
	// ScannedAt is the cached scan's timestamp (RFC3339); empty when absent.
	ScannedAt string `json:"scannedAt,omitempty"`
	// Cleared reports whether this run removed the cache file.
	Cleared bool `json:"cleared"`
}

func runCache(args []string) int {
	fs := flag.NewFlagSet("cache", flag.ContinueOnError)
	clear := fs.Bool("clear", false, "clear the scan cache")
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlags(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // 帮助请求不是错误
		}
		return 1
	}
	if *clear {
		if err := scanner.ClearCache(); err != nil {
			if *jsonOut {
				emitJSONError(KindCache, map[string]any{"message": err.Error()}, *pretty)
			} else {
				fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.cacheClearFailed", map[string]any{"err": err}))
			}
			return 1
		}
		if *jsonOut {
			_ = emitJSON(KindCache, CacheReport{Path: platform.CacheRoot(), Cleared: true}, *pretty)
		} else {
			fmt.Fprintln(stdout(), i18n.T("cli.cacheCleared"))
		}
		return 0
	}
	ts, ok := scanner.CacheInfo()
	if *jsonOut {
		_ = emitJSON(KindCache, CacheReport{Path: platform.CacheRoot(), Present: ok, ScannedAt: ts}, *pretty)
		return 0
	}
	if ok {
		fmt.Fprint(stdout(), i18n.T("cli.cacheExists", map[string]any{"time": ts, "path": platform.CacheRoot()}), "\n")
	} else {
		fmt.Fprint(stdout(), i18n.T("cli.cacheNone", map[string]any{"path": platform.CacheRoot()}), "\n")
	}
	return 0
}
