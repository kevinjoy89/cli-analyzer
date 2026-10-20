package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"cli-analyzer/internal/i18n"
	"cli-analyzer/internal/scanner"
)

// SchemaVersion is the version of the CLI's machine-readable output contract.
// It is bumped only for breaking changes; additive fields do not bump it.
// See docs/adr/0002-cleanable-actionable-and-contract-compat.md.
const SchemaVersion = 1

// Report kinds. Each JSON document carries one of these in its top-level
// `kind` field so consumers can tell report types apart without guessing.
const (
	KindScan         = "scan"
	KindClean        = "clean"
	KindCache        = "cache"
	KindTrashList    = "trash-list"
	KindTrashRestore = "trash-restore"
	KindTrashEmpty   = "trash-empty"
	KindTrends       = "trends"
	KindUninstall    = "uninstall"
	KindAppUpdate    = "app-update"
	KindToolCheck    = "tool-check"
	KindToolUpgrade  = "tool-upgrade"
	KindBaseline     = "baseline-diff"
	KindResidue      = "residue"
)

// emitJSON writes payload to stdout as a single JSON document carrying
// `schemaVersion` and `kind` alongside the payload's own fields (flat, not
// nested — existing consumers keep reading the same top-level keys).
//
// Compact by default (one line, pipe-friendly); pretty=true indents it for
// human reading. Array payloads are wrapped as {"...", "items": [...]}
// because a bare array has nowhere to put the contract fields.
func emitJSON(kind string, payload any, pretty bool) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		// payload 不是 JSON 对象（数组或标量）：用 items 包起来。
		doc = map[string]any{"items": json.RawMessage(b)}
	}
	doc["schemaVersion"] = SchemaVersion
	doc["kind"] = kind

	var out []byte
	if pretty {
		out, err = json.MarshalIndent(doc, "", "  ")
	} else {
		out, err = json.Marshal(doc)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout(), string(out))
	return nil
}

// emitJSONError writes a machine-readable error document. Kept separate so
// error reports still carry the contract fields and remain distinguishable
// from a successful report of the same kind.
func emitJSONError(kind string, fields map[string]any, pretty bool) {
	doc := map[string]any{"error": true}
	for k, v := range fields {
		doc[k] = v
	}
	_ = emitJSON(kind, doc, pretty)
}

// parseSize parses a human byte size ("500MB", "1.5GiB", "1024").
// Used by --fail-on-grow so thresholds can be written naturally in CI.
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	mult := int64(1)
	units := []struct {
		suffix string
		mult   int64
	}{
		{"KIB", 1 << 10}, {"MIB", 1 << 20}, {"GIB", 1 << 30}, {"TIB", 1 << 40},
		{"KB", 1000}, {"MB", 1000 * 1000}, {"GB", 1000 * 1000 * 1000}, {"TB", 1000 * 1000 * 1000 * 1000},
		{"K", 1 << 10}, {"M", 1 << 20}, {"G", 1 << 30}, {"T", 1 << 40},
		{"B", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			mult = u.mult
			s = strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			break
		}
	}
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	if f < 0 {
		return 0, fmt.Errorf("negative size %q", s)
	}
	return int64(f * float64(mult)), nil
}

// humanBytes renders a byte count compactly (e.g. "1.2 GB").
// 后缀表覆盖 int64 全范围：最大 9.2 EiB 对应 exp=5，故需 6 个后缀
// （K/M/G/T/P/E），此前 4 个后缀在 n ≥ 1 PiB 时 "KMGT"[4] 越界 panic。
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
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
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
		name := t.Name
		if t.DisplayName != "" {
			name = t.DisplayName
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			name, cmds, humanBytes(t.Footprint), humanBytes(t.Cleanable), humanBytes(t.User), t.Installer)
	}
	fmt.Fprint(w, i18n.T("cli.totalsRow", map[string]any{
		"footprint": humanBytes(res.Totals.Footprint),
		"cleanable": humanBytes(res.Totals.Cleanable),
		"user":      humanBytes(res.Totals.User),
	}))
	w.Flush()
	fmt.Fprint(stdout(), i18n.T("cli.statsLine", map[string]any{"n": len(res.Tools), "ms": res.ScanTimeMS, "errors": res.Errors}))
	if len(res.Unattributed) > 0 {
		var total int64
		for _, u := range res.Unattributed {
			total += u.Bytes
		}
		fmt.Fprint(stdout(), i18n.T("cli.unattributed", map[string]any{"n": len(res.Unattributed), "size": humanBytes(total)}))
	}
}

// printCleanables lists all actionable items across tools (for `clean --list`).
// Tier is a label, not a gate — everything attributed is listed.
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
