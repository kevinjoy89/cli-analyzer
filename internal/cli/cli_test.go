package cli

import (
	"bytes"
	"strings"
	"testing"

	"cli-analyzer/internal/scanner"
)

// captureStdout substitutes the package-level out writer and restores it after
// the test, so we can assert on CLI output.
func captureStdout(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := out
	out = buf
	t.Cleanup(func() { out = old })
	return buf
}

func captureStderr(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := errOut
	errOut = buf
	t.Cleanup(func() { errOut = old })
	return buf
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{5 * 1024 * 1024 * 1024, "5.0 GB"},
		{10947133440, "10.2 GB"}, // 10*1GB + 200*1MB
	}
	for _, c := range cases {
		if got := humanBytes(c.in); got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMatchAny(t *testing.T) {
	if !matchAny("cli-analyzer", nil, []string{"cli"}) {
		t.Error("name substring should match")
	}
	if matchAny("npm", nil, []string{"xyz"}) {
		t.Error("non-matching filter should fail")
	}
	if !matchAny("gh", []string{"hub"}, []string{"hub"}) {
		t.Error("alias should match")
	}
	if matchAny("", nil, []string{"x"}) {
		t.Error("empty name should not match")
	}
}

func TestReorderFlags(t *testing.T) {
	got := reorderFlags([]string{"pylint", "--json", "-n", "kimi"})
	want := []string{"--json", "-n", "pylint", "kimi"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("reorderFlags = %v, want %v", got, want)
	}
}

func TestRunVersion(t *testing.T) {
	buf := captureStdout(t)
	if code := Run([]string{"version"}); code != 0 {
		t.Fatalf("version exit = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "cli-analyzer 0.1.0") {
		t.Errorf("version output = %q", buf.String())
	}
}

func TestRunVersionShortFlags(t *testing.T) {
	buf := captureStdout(t)
	for _, a := range []string{"--version", "-v"} {
		buf.Reset()
		if code := Run([]string{a}); code != 0 {
			t.Errorf("%s exit = %d, want 0", a, code)
		}
		if !strings.Contains(buf.String(), "cli-analyzer 0.1.0") {
			t.Errorf("%s output = %q", a, buf.String())
		}
	}
}

func TestRunHelp(t *testing.T) {
	buf := captureStdout(t)
	if code := Run([]string{"help"}); code != 0 {
		t.Fatalf("help exit = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "用法") {
		t.Errorf("help should print usage, got %q", buf.String())
	}
}

func TestRunNoArgsPrintsUsage(t *testing.T) {
	buf := captureStdout(t)
	if code := Run(nil); code != 0 {
		t.Fatalf("no-arg exit = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "scan") {
		t.Errorf("usage should mention scan, got %q", buf.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	captureStdout(t)
	errBuf := captureStderr(t)
	if code := Run([]string{"bogus"}); code != 1 {
		t.Fatalf("unknown command exit = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "unknown command") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

func TestRunFlagErrors(t *testing.T) {
	captureStdout(t)
	captureStderr(t)
	for _, args := range [][]string{
		{"scan", "--nope"},
		{"clean", "--nope"},
		{"cache", "--nope"},
	} {
		if code := Run(args); code != 1 {
			t.Errorf("Run(%v) exit = %d, want 1", args, code)
		}
	}
}

func TestFilterResult(t *testing.T) {
	res := &scanner.ScanResult{
		Tools: []scanner.Tool{
			{Name: "npm", Footprint: 100, Cleanable: 60, User: 40},
			{Name: "go", Footprint: 50, Cleanable: 0, User: 50},
		},
		Totals: scanner.Totals{Footprint: 150, Cleanable: 60, User: 90},
	}
	filtered := filterResult(res, []string{"npm"})
	if len(filtered.Tools) != 1 || filtered.Tools[0].Name != "npm" {
		t.Fatalf("filtered tools = %+v", filtered.Tools)
	}
	if filtered.Totals.Footprint != 100 || filtered.Totals.Cleanable != 60 || filtered.Totals.User != 40 {
		t.Errorf("filtered totals = %+v", filtered.Totals)
	}
}

func TestFilterResultNoMatch(t *testing.T) {
	res := &scanner.ScanResult{Tools: []scanner.Tool{{Name: "npm"}}}
	filtered := filterResult(res, []string{"zzz"})
	if len(filtered.Tools) != 0 {
		t.Fatalf("expected no tools, got %+v", filtered.Tools)
	}
	if filtered.Totals.Footprint != 0 || filtered.Totals.Cleanable != 0 || filtered.Totals.User != 0 {
		t.Errorf("empty totals = %+v", filtered.Totals)
	}
}

func TestPrintTable(t *testing.T) {
	res := &scanner.ScanResult{
		Tools: []scanner.Tool{
			{Name: "npm", Binaries: []scanner.Binary{{Name: "npm", Path: "/x/npm"}}, Footprint: 100, Cleanable: 60, User: 40, Installer: "npm"},
			{Name: "gh", Footprint: 10, Cleanable: 0, User: 10, Installer: "brew"},
		},
		Totals:     scanner.Totals{Footprint: 110, Cleanable: 60, User: 50},
		ScanTimeMS: 12,
	}
	buf := captureStdout(t)
	printTable(res)
	s := buf.String()
	for _, want := range []string{"工具", "npm", "gh", "合计", "共 2 个工具"} {
		if !strings.Contains(s, want) {
			t.Errorf("table missing %q in:\n%s", want, s)
		}
	}
}

func TestPrintCleanables(t *testing.T) {
	res := &scanner.ScanResult{
		Tools: []scanner.Tool{
			{Name: "npm", Cleanables: []scanner.Cleanable{
				{Tool: "npm", Path: "/cache/npm", Bytes: 1024 * 1024 * 1024, Kind: "cache", Tier: scanner.TierSafe},
			}},
		},
	}
	buf := captureStdout(t)
	printCleanables(res, nil)
	s := buf.String()
	for _, want := range []string{"工具", "npm", "1.0 GB", "共 1 项可安全清理"} {
		if !strings.Contains(s, want) {
			t.Errorf("cleanables output missing %q in:\n%s", want, s)
		}
	}
}
