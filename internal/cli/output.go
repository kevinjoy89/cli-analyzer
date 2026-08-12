package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

// humanBytes renders a byte count compactly (e.g. "1.2 GB").
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// printTable renders the scan result as a terminal table.
func printTable(res *scanner.ScanResult) {
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.tableHeader"))
	for _, t := range res.Tools {
		cmds := fmt.Sprint(len(t.Binaries))
		if len(t.Binaries) == 0 {
			cmds = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			t.Name, cmds, humanBytes(t.Footprint), humanBytes(t.Cleanable), humanBytes(t.User), t.Installer)
	}
	fmt.Fprintf(w, i18n.T("cli.totalsRow"),
		humanBytes(res.Totals.Footprint), humanBytes(res.Totals.Cleanable), humanBytes(res.Totals.User))
	w.Flush()
	fmt.Fprint(stdout(), i18n.T("cli.statsLine", map[string]any{"n": len(res.Tools), "ms": res.ScanTimeMS, "errors": res.Errors}))
	if len(res.Unattributed) > 0 {
		fmt.Fprint(stdout(), i18n.T("cli.unattributed", map[string]any{"n": len(res.Unattributed)}))
	}
}

// printCleanables lists all SAFE cleanables across tools (for `clean --list`).
func printCleanables(res *scanner.ScanResult, onlyTools []string) {
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.cleanHeader"))
	total := int64(0)
	shown := 0
	for _, t := range res.Tools {
		if len(onlyTools) > 0 && !matchAny(t.Name, t.Aliases, onlyTools) {
			continue
		}
		for _, c := range t.Cleanables {
			if c.Tier != scanner.TierSafe {
				continue
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Name, humanBytes(c.Bytes), c.Kind, c.Path)
			total += c.Bytes
			shown++
		}
	}
	w.Flush()
	fmt.Fprint(stdout(), i18n.T("cli.cleanTotal", map[string]any{"n": shown, "size": humanBytes(total)}))
}

func matchAny(name string, aliases []string, filters []string) bool {
	for _, f := range filters {
		if strings.Contains(name, f) {
			return true
		}
		for _, a := range aliases {
			if strings.Contains(a, f) {
				return true
			}
		}
	}
	return false
}
