package cli

import (
	"fmt"
	"sort"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

// BaselineDiff is the machine-readable contract of `scan --baseline`.
type BaselineDiff struct {
	Tools    []ToolDelta `json:"tools"`
	Added    []string    `json:"added"`
	Removed  []string    `json:"removed"`
	Exceeded []ToolDelta `json:"exceeded,omitempty"`
	// ThresholdBytes is the --fail-on-grow value (0 = report only).
	ThresholdBytes int64 `json:"thresholdBytes"`
}

// ToolDelta is one tool's change against the baseline.
type ToolDelta struct {
	Tool           string `json:"tool"`
	FootprintDelta int64  `json:"footprintDelta"`
	CleanableDelta int64  `json:"cleanableDelta"`
	UserDelta      int64  `json:"userDelta"`
}

// diffAgainst compares res to base. Tools whose bytes did not change are
// omitted (a baseline report is about change). A tool counts as "added" when
// it exists now but not in the baseline, "removed" for the reverse.
func diffAgainst(res, base *scanner.ScanResult, threshold int64) BaselineDiff {
	baseBy := make(map[string]scanner.Tool, len(base.Tools))
	for _, t := range base.Tools {
		baseBy[t.Name] = t
	}
	nowBy := make(map[string]bool, len(res.Tools))

	out := BaselineDiff{
		Tools:          []ToolDelta{},
		Added:          []string{},
		Removed:        []string{},
		ThresholdBytes: threshold,
	}
	for _, t := range res.Tools {
		nowBy[t.Name] = true
		prev, ok := baseBy[t.Name]
		if !ok {
			out.Added = append(out.Added, t.Name)
			continue
		}
		d := ToolDelta{
			Tool:           t.Name,
			FootprintDelta: t.Footprint - prev.Footprint,
			CleanableDelta: t.Cleanable - prev.Cleanable,
			UserDelta:      t.User - prev.User,
		}
		if d.FootprintDelta == 0 && d.CleanableDelta == 0 && d.UserDelta == 0 {
			continue
		}
		out.Tools = append(out.Tools, d)
		if threshold > 0 && d.CleanableDelta > threshold {
			out.Exceeded = append(out.Exceeded, d)
		}
	}
	for _, t := range base.Tools {
		if !nowBy[t.Name] {
			out.Removed = append(out.Removed, t.Name)
		}
	}
	sort.Strings(out.Added)
	sort.Strings(out.Removed)
	sort.SliceStable(out.Tools, func(i, j int) bool {
		return out.Tools[i].CleanableDelta > out.Tools[j].CleanableDelta
	})
	return out
}

// printBaselineDiff renders the diff for humans.
func printBaselineDiff(d BaselineDiff) {
	if len(d.Tools) == 0 && len(d.Added) == 0 && len(d.Removed) == 0 {
		fmt.Fprintln(stdout(), i18n.T("cli.baselineNoChange"))
		return
	}
	for _, t := range d.Tools {
		fmt.Fprintf(stdout(), "%s\t%s\t%s\t%s\n",
			t.Tool,
			signedBytes(t.FootprintDelta),
			signedBytes(t.CleanableDelta),
			signedBytes(t.UserDelta))
	}
	for _, n := range d.Added {
		fmt.Fprintln(stdout(), i18n.T("cli.baselineAdded", map[string]any{"tool": n}))
	}
	for _, n := range d.Removed {
		fmt.Fprintln(stdout(), i18n.T("cli.baselineRemoved", map[string]any{"tool": n}))
	}
	for _, t := range d.Exceeded {
		fmt.Fprintln(stdout(), i18n.T("cli.baselineExceeded", map[string]any{
			"tool": t.Tool, "delta": signedBytes(t.CleanableDelta),
		}))
	}
}

// signedBytes renders a byte delta with an explicit sign ("+1.2 GB").
func signedBytes(n int64) string {
	if n > 0 {
		return "+" + humanBytes(n)
	}
	if n < 0 {
		return "-" + humanBytes(-n)
	}
	return humanBytes(0)
}
