package cli

import (
	"errors"
	"flag"
	"fmt"
	"text/tabwriter"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/uninstall"
)

// ResidueReport is the machine-readable contract of `residue`.
type ResidueReport struct {
	MinDays int                      `json:"minDays"`
	Pending []uninstall.PendingEvent `json:"pending"`
}

// runResidue 处理 `residue [--days N] [--forget <tool>] [--json]`：
// 列出「卸载已满 N 天、残留仍在」的工具，或收尾某个工具的跟踪记录。
//
// 这是残留闭环的出口：uninstall 落盘事件，这里提醒，用户处置完用 --forget
// 收尾。不做定时任务（CLI 不该常驻）。
func runResidue(args []string) int {
	fs := flag.NewFlagSet("residue", flag.ContinueOnError)
	days := fs.Int("days", 7, "only report uninstalls older than this many days")
	forget := fs.String("forget", "", "stop tracking this tool")
	jsonOut := fs.Bool("j", false, "output JSON")
	fs.BoolVar(jsonOut, "json", false, "output JSON")
	pretty := fs.Bool("pretty", false, "indent JSON output")
	fs.SetOutput(stderr())
	if err := fs.Parse(reorderFlagsWithValue(args, map[string]bool{"days": true, "forget": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	if *forget != "" {
		if err := uninstall.Forget(*forget); err != nil {
			if *jsonOut {
				emitJSONError(KindResidue, map[string]any{"tool": *forget, "message": err.Error()}, *pretty)
			} else {
				fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": err}))
			}
			return 1
		}
		if *jsonOut {
			_ = emitJSON(KindResidue, map[string]any{"forgotten": *forget}, *pretty)
		} else {
			fmt.Fprintln(stdout(), i18n.T("cli.residueForgot", map[string]any{"tool": *forget}))
		}
		return 0
	}

	pending, err := uninstall.Pending(*days)
	if err != nil {
		if *jsonOut {
			emitJSONError(KindResidue, map[string]any{"message": err.Error()}, *pretty)
		} else {
			fmt.Fprintf(stderr(), "%s\n", i18n.T("cli.errors", map[string]any{"msg": err}))
		}
		return 1
	}
	if *jsonOut {
		if pending == nil {
			pending = []uninstall.PendingEvent{}
		}
		_ = emitJSON(KindResidue, ResidueReport{MinDays: *days, Pending: pending}, *pretty)
		return 0
	}
	if len(pending) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.residueNone", map[string]any{"days": *days}))
		return 0
	}
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cli.residueHeader"))
	for _, p := range pending {
		fmt.Fprintf(w, "%s\t%d\t%s\n", p.Tool, p.DaysAgo, humanBytes(p.Bytes))
		for _, r := range p.Residue {
			fmt.Fprintf(w, "\t\t%s\n", r.Path)
		}
	}
	w.Flush()
	return 0
}
