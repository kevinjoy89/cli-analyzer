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
//
// 注意：PathDirs 会追加 augmentUserDirs 的固定目录（/usr/local/bin、
// ~/.local/bin 等，GUI 启动 PATH 补齐的产品设计），故不断言工具总数，
// 只断言 nodejs 工具的合并契约。
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
	var tb *Tool
	for i := range res.Tools {
		if res.Tools[i].Name == "nodejs" {
			tb = &res.Tools[i]
			break
		}
	}
	if tb == nil {
		names := make([]string, 0, len(res.Tools))
		for _, x := range res.Tools {
			names = append(names, x.Name)
		}
		t.Fatalf("nodejs tool not found; got: %v", names)
	}
	if tb.Family != "nodejs" {
		t.Fatalf("family = %q, want nodejs", tb.Family)
	}
	// 家族合并：PATH 注入的 tempdir 四个命令全部归入 nodejs
	wantBins := map[string]bool{}
	for _, name := range []string{"node", "npm.cmd", "npx.cmd", "corepack.cmd"} {
		wantBins[filepath.Join(dir, name)] = true
	}
	for _, b := range tb.Binaries {
		delete(wantBins, b.Path)
	}
	if len(wantBins) != 0 {
		t.Errorf("tempdir binaries missing from nodejs: %v (got %d bins)", wantBins, len(tb.Binaries))
	}
	// 主二进制 node 排首位（probeOrder）
	if len(tb.Binaries) == 0 || tb.Binaries[0].Name != "node" {
		t.Errorf("binaries[0] = %+v, want node first (probe order)", tb.Binaries)
	}
	// 别名归一化：家族别名剥扩展名
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
