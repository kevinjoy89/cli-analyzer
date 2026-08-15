package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/history"
)

// TestRunScanFilterDoesNotPolluteHistory 验证过滤扫描（`scan <filter> --refresh`）
// 不得把"过滤后的 totals"写入历史：历史快照是整体占用趋势的数据源，写入过滤值
// 会让趋势图/增量排行显示错误数据。红测试：修复前 runScan 无条件 history.Record
// 过滤结果，过滤扫描后历史最新记录的 footprint 变成只有单个工具的值。
func TestRunScanFilterDoesNotPolluteHistory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only PATH 注入用例")
	}
	// 隔离数据根：History 落在 DataRoot（darwin ~/Library/Application Support）
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())

	dir := t.TempDir()
	for _, name := range []string{"aa", "bb"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	// 全量扫描（刷新），历史记录全量 totals
	if code := runScan([]string{"--refresh", "--no-cache"}); code != 0 {
		t.Fatalf("full scan code = %d", code)
	}
	full, err := history.Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Points) == 0 {
		t.Fatal("无历史记录")
	}
	fullFP := full.Points[len(full.Points)-1].Footprint
	if fullFP <= 0 {
		t.Fatalf("全量 footprint 应为正，got %d", fullFP)
	}

	// 过滤扫描（刷新）：历史不得被过滤值覆盖
	if code := runScan([]string{"--refresh", "--no-cache", "aa"}); code != 0 {
		t.Fatalf("filtered scan code = %d", code)
	}
	after, err := history.Trends(30)
	if err != nil {
		t.Fatal(err)
	}
	lastFP := after.Points[len(after.Points)-1].Footprint
	if lastFP != fullFP {
		t.Errorf("过滤扫描污染历史: footprint %d != 全量 %d（过滤扫描不应写历史）", lastFP, fullFP)
	}
}
