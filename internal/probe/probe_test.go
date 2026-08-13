package probe

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// writeScript 生成一个 /bin/sh 脚本（非 Windows），按参数分派行为。
func writeScript(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script probe tests skip on windows")
	}
	p := filepath.Join(t.TempDir(), "probe-tool")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbeVersionOrder(t *testing.T) {
	// 只有 --version 成功
	bin := writeScript(t, `case "$1" in
  --version) echo "uv 0.4.10"; exit 0;;
  *) exit 1;;
esac
`)
	v, ok := ProbeVersion(bin)
	if !ok || v != "uv 0.4.10" {
		t.Errorf("ProbeVersion = (%q,%v), want (uv 0.4.10, true)", v, ok)
	}
}

func TestProbeFallsBackToHelp(t *testing.T) {
	// --version/-V 失败，--help 成功
	bin := writeScript(t, `case "$1" in
  --help) echo "Usage: probe-tool [options]"; exit 0;;
  *) exit 1;;
esac
`)
	v, ok := ProbeVersion(bin)
	if !ok || v != "Usage: probe-tool [options]" {
		t.Errorf("ProbeVersion = (%q,%v), want help line", v, ok)
	}
}

func TestProbeTimeout(t *testing.T) {
	bin := writeScript(t, `sleep 10; echo never`)
	start := time.Now()
	v, ok := ProbeVersion(bin)
	elapsed := time.Since(start)
	if ok {
		t.Errorf("expected timeout failure, got (%q, true)", v)
	}
	if elapsed > 5*time.Second {
		t.Errorf("timeout too slow: %v", elapsed)
	}
}

func TestProbeFailureDegrades(t *testing.T) {
	bin := writeScript(t, `exit 2`)
	if v, ok := ProbeVersion(bin); ok {
		t.Errorf("expected failure, got (%q, true)", v)
	}
}

func TestCachedOrRun(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "runs")
	bin := writeScript(t, `echo "app 9.9.9" >> `+counter+`; echo "app 9.9.9"; exit 0`)
	st, _ := os.Stat(bin)
	v1, ok1 := CachedOrRun(bin, st.Size())
	v2, ok2 := CachedOrRun(bin, st.Size())
	if !ok1 || !ok2 || v1 != "app 9.9.9" || v2 != "app 9.9.9" {
		t.Errorf("CachedOrRun = (%q,%v) / (%q,%v)", v1, ok1, v2, ok2)
	}
	b, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Split(strings.TrimSpace(string(b)), "\n")); n != 1 {
		t.Errorf("expected 1 execution (cache hit on 2nd call), got %d", n)
	}
}

func TestCleanOutputGBK(t *testing.T) {
	// "版本 1.2.3" 的 GBK 字节
	gbk := []byte{0xB0, 0xE6, 0xB1, 0xBE, 0x20, 0x31, 0x2E, 0x32, 0x2E, 0x33, '\n'}
	got := extractVersion(gbk)
	if got != "版本 1.2.3" {
		t.Errorf("GBK decode = %q, want 版本 1.2.3", got)
	}
}

func TestCleanOutputStripsANSI(t *testing.T) {
	in := "\x1b[32mgreen 1.0\x1b[0m\n"
	if got := extractVersion([]byte(in)); got != "green 1.0" {
		t.Errorf("ANSI strip = %q", got)
	}
}
