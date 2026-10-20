package cli

import (
	"errors"
	"flag"
	"fmt"
	"text/tabwriter"

	"cli-analyzer/internal/history"
	"cli-analyzer/internal/i18n"
)

// TrendsReport is the machine-readable contract of `trends`.
// Points and TopGrowers come straight from history.TrendsResult; OmittedGrowers
// is set when --top truncates the grower list.
type TrendsReport struct {
	Days           int              `json:"days"`
	Points         []history.Point  `json:"points"`
	TopGrowers     []history.Grower `json:"topGrowers"`
	OmittedGrowers int              `json:"omittedGrowers,omitempty"`
}

// runTrends 打印最近 days 天的占用趋势与 cleanable 增量排行；可带一个天数参数
func runTrends(args []string) int {
	fs := flag.NewFlagSet("trends", flag.ContinueOnError)
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
	top := fs.Int("top", 0, "only output the top N growers (0 = default 5)")
	fs.SetOutput(stderr())
	// trends 的位置参数是天数（`trends 90`），reorderFlags 让 `trends 90 --json`
	// 与 `trends --json 90` 等价。
	if err := fs.Parse(reorderFlagsWithValue(args, map[string]bool{"top": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(stdout(), i18n.T("cli.trendsHelp"))
			return 0
		}
		return 1
	}
	days := 30
	if fs.NArg() >= 1 {
		var d int
		if _, err := fmt.Sscanf(fs.Arg(0), "%d", &d); err == nil && d > 0 {
			days = d
		}
	}

	tr, err := history.Trends(days)
	if err != nil {
		if *jsonOut {
			emitJSONError(KindTrends, map[string]any{"message": err.Error()}, *pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.trendsFailed", map[string]any{"err": err}))
		}
		return 1
	}

	report := TrendsReport{Days: days, Points: tr.Points, TopGrowers: tr.TopGrowers}
	if report.Points == nil {
		report.Points = []history.Point{}
	}
	if report.TopGrowers == nil {
		report.TopGrowers = []history.Grower{}
	}
	limit := 5
	if *top > 0 {
		limit = *top
	}
	if len(report.TopGrowers) > limit {
		report.OmittedGrowers = len(report.TopGrowers) - limit
		report.TopGrowers = report.TopGrowers[:limit]
	}

	if *jsonOut {
		_ = emitJSON(KindTrends, report, *pretty)
		return 0
	}
	if len(report.Points) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.trendsEmpty"))
		return 0
	}
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.trendsHeader"))
	for _, p := range report.Points {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Date, humanBytes(p.Footprint), humanBytes(p.Cleanable), humanBytes(p.User))
	}
	w.Flush()

	if len(report.TopGrowers) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.growersNone"))
	} else {
		fmt.Fprintln(stdout(), i18n.T("cli.growersTitle"))
		for i, g := range report.TopGrowers {
			fmt.Fprintf(stdout(), "  %d. %s  +%s\n", i+1, g.Tool, humanBytes(g.DeltaBytes))
		}
		if report.OmittedGrowers > 0 {
			fmt.Fprint(stdout(), i18n.T("cli.omittedGrowers", map[string]any{"n": report.OmittedGrowers}))
		}
	}
	return 0
}
