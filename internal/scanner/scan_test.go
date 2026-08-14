package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestScanNodejsFamilyEndToEnd 用受控 PATH 验证 Node.js 家族合并全链路：
// 同一目录的 node / npm.cmd / npx.cmd / corepack.cmd → 单一 nodejs 工具，
// 别名归一化（剥 .cmd 等扩展名）、family 字段、主二进制 node 排首位
// （probeOrder，版本探测取运行时版本）。
func TestScanNodejsFamilyEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		// 该用例以 unix 可执行位与无扩展名 node 为前置（Windows 需 .exe +
		// PATHEXT 语义，由 classify 单测覆盖同目录布局合并）
		t.Skip("unix-only PATH 注入用例")
	}
	dir := t.TempDir()
	for _, name := range []string{"node", "npm.cmd", "npx.cmd", "corepack.cmd"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	res, err := Scan(Options{NoCache: true, ToolFilter: []string{"nodejs"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != 1 {
		names := make([]string, 0, len(res.Tools))
		for _, tb := range res.Tools {
			bins := make([]string, 0, len(tb.Binaries))
			for _, b := range tb.Binaries {
				bins = append(bins, b.Path)
			}
			names = append(names, tb.Name+"(aliases="+strings.Join(tb.Aliases, ",")+";bins="+strings.Join(bins, ",")+")")
		}
		t.Fatalf("PATH=%q; want exactly the nodejs tool, got %d tools: %v", os.Getenv("PATH"), len(res.Tools), names)
	}
	tb := res.Tools[0]
	if tb.Name != "nodejs" || tb.Family != "nodejs" {
		t.Fatalf("tool = %q (family %q), want nodejs/nodejs", tb.Name, tb.Family)
	}
	if len(tb.Binaries) == 0 || tb.Binaries[0].Name != "node" {
		t.Errorf("binaries[0] = %+v, want node first (probe order)", tb.Binaries)
	}
	wantAliases := map[string]bool{"node": true, "npm": true, "npx": true, "corepack": true}
	for _, a := range tb.Aliases {
		if strings.ContainsAny(a, ".") {
			t.Errorf("alias %q not normalized (extension should be stripped)", a)
		}
		delete(wantAliases, a)
	}
	if len(wantAliases) != 0 {
		t.Errorf("missing aliases %v (got %v)", wantAliases, tb.Aliases)
	}
}
