package cli

import (
	"fmt"
	"text/tabwriter"

	"cli-analyzer/internal/history"
	"cli-analyzer/internal/i18n"
)

// runTrends 打印最近 days 天的占用趋势与 cleanable 增量 Top 5；可带一个天数参数
func runTrends(args []string) int {
	days := 30
	if len(args) >= 1 {
		var d int
		if _, err := fmt.Sscanf(args[0], "%d", &d); err == nil && d > 0 {
			days = d
		}
	}
	tr, err := history.Trends(days)
	if err != nil {
		fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trendsFailed", map[string]any{"err": err}))
		return 1
	}
	if len(tr.Points) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.trendsEmpty"))
		return 0
	}
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.trendsHeader"))
	for _, p := range tr.Points {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Date, humanBytes(p.Footprint), humanBytes(p.Cleanable), humanBytes(p.User))
	}
	w.Flush()

	if len(tr.TopGrowers) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.growersNone"))
	} else {
		fmt.Fprintln(stdout(), i18n.T("cli.growersTitle"))
		for i, g := range tr.TopGrowers {
			fmt.Fprintf(stdout(), "  %d. %s  +%s\n", i+1, g.Tool, humanBytes(g.DeltaBytes))
		}
	}
	return 0
}
