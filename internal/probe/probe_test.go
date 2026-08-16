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
	if !ok || v != "0.4.10" {
		t.Errorf("ProbeVersion = (%q,%v), want (0.4.10, true)", v, ok)
	}
}

func TestProbeFallsBackToHelp(t *testing.T) {
	// --version/-V 失败，--help 输出含版本号 → 回退成功并提取版本
	bin := writeScript(t, `case "$1" in
  --help) echo "probe-tool v1.2.3 — usage: probe-tool [options]"; exit 0;;
  *) exit 1;;
esac
`)
	v, ok := ProbeVersion(bin)
	if !ok || v != "1.2.3" {
		t.Errorf("ProbeVersion = (%q,%v), want (1.2.3, true)", v, ok)
	}
}

func TestProbeHelpWithoutVersionFails(t *testing.T) {
	// --help 成功但输出没有版本号（kubectl 等）→ 不赋值（用户要求：不清楚就空）
	bin := writeScript(t, `case "$1" in
  --help) echo "kubectl controls the Kubernetes cluster manager."; exit 0;;
  *) exit 1;;
esac
`)
	if v, ok := ProbeVersion(bin); ok {
		t.Errorf("help without version must fail, got (%q, true)", v)
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
	v1, ok1 := CachedOrRun(bin, bin, st.Size())
	v2, ok2 := CachedOrRun(bin, bin, st.Size())
	if !ok1 || !ok2 || v1 != "9.9.9" || v2 != "9.9.9" {
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
	if got != "1.2.3" {
		t.Errorf("GBK decode = %q, want 1.2.3", got)
	}
}

func TestCleanOutputStripsANSI(t *testing.T) {
	in := "\x1b[32mgreen 1.0\x1b[0m\n"
	if got := extractVersion([]byte(in)); got != "1.0" {
		t.Errorf("ANSI strip = %q", got)
	}
}

func TestExtractVersionForms(t *testing.T) {
	cases := map[string]string{
		"uv 0.12.5 (210d1f678 2026-08-14 aarch64-apple-darwin)":         "0.12.5",
		"pip 20.2.3 from /Library/Frameworks/Python.framework (py 3.9)": "20.2.3",
		"Docker version 29.4.0, build 9d7ad9f":                          "29.4.0",
		"Docker Compose version v5.1.2":                                 "5.1.2",
		"go version go1.22.5 darwin/arm64":                              "1.22.5",
		"node: v22.14.0":                                                "22.14.0",
		"1.18.18":                                                       "1.18.18",
		"codex-cli 0.147.0":                                             "0.147.0",
		"kubectl controls the Kubernetes cluster manager.":              "",
		"Usage: probe-tool [options]":                                   "",
		"isort _":                                                       "",
		"gh version 2.5.0 (2024-01-01)":                                 "2.5.0",
	}
	for in, want := range cases {
		if got := extractVersion([]byte(in)); got != want {
			t.Errorf("extractVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
