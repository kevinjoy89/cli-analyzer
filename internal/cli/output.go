package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

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
	fmt.Fprintln(w, "工具\t命令\t总占用\t可清理(SAFE)\t用户数据\t来源")
	for _, t := range res.Tools {
		cmds := fmt.Sprint(len(t.Binaries))
		if len(t.Binaries) == 0 {
			cmds = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			t.Name, cmds, humanBytes(t.Footprint), humanBytes(t.Cleanable), humanBytes(t.User), t.Installer)
	}
	fmt.Fprintf(w, "合计\t-\t%s\t%s\t%s\t-\n",
		humanBytes(res.Totals.Footprint), humanBytes(res.Totals.Cleanable), humanBytes(res.Totals.User))
	w.Flush()
	fmt.Fprintf(stdout(), "\n共 %d 个工具 · 扫描用时 %d ms · 遍历错误 %d\n", len(res.Tools), res.ScanTimeMS, res.Errors)
	if len(res.Unattributed) > 0 {
		fmt.Fprintf(stdout(), "另有 %d 个未归因目录（--full）\n", len(res.Unattributed))
	}
}

// printCleanables lists all SAFE cleanables across tools (for `clean --list`).
func printCleanables(res *scanner.ScanResult, onlyTools []string) {
	w := tabwriter.NewWriter(stdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "工具\t占用\t类型\t路径")
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
	fmt.Fprintf(stdout(), "\n共 %d 项可安全清理，合计 %s\n", shown, humanBytes(total))
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
