package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/config"
	"cli-analyzer/internal/scanner"
)

// setupBulkEnv 隔离配置与扫描缓存根，并写一份含指定工具的扫描缓存。
func setupBulkEnv(t *testing.T, tools ...scanner.Tool) {
	t.Helper()
	cacheEnv(t)
	restore := config.SetDataRoot(t.TempDir())
	t.Cleanup(restore)
	c := config.Default()
	c.Language = config.LangZhCN
	if err := config.Save(c); err != nil {
		t.Fatal(err)
	}
	if err := scanner.SaveCache(&scanner.ScanResult{Tools: tools}); err != nil {
		t.Fatal(err)
	}
}

// saveConfig 写入一份改过的配置（用于验证 bulkKinds / bulkExclude）。
func saveConfig(t *testing.T, mutate func(*config.Config)) {
	t.Helper()
	c := config.Load()
	mutate(c)
	if err := config.Save(c); err != nil {
		t.Fatal(err)
	}
}

// runCleanDryRun 执行 clean --all --dry-run --json 并返回解析后的报告。
func runCleanDryRun(t *testing.T, extra ...string) scanner.CleanReport {
	t.Helper()
	args := append([]string{"clean", "--all", "--dry-run", "--json"}, extra...)
	buf := captureStdout(t)
	if code := Run(args); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var rep scanner.CleanReport
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("parse: %v\n%s", err, buf.String())
	}
	return rep
}

// mkCustomKindTool 构造一个只含自定义 Kind 条目的扫描工具。
func mkCustomKindTool(t *testing.T, kind string) (scanner.Tool, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "custom")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return scanner.Tool{
		Name: "mytool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "mytool|" + kind + "|" + dir, Tool: "mytool", Path: dir, Bytes: 100, Kind: kind},
		},
	}, dir
}

// TestBulkCustomKindNotSelectedByDefault 是本变更的核心安全默认：
// 自定义 Kind 未配置时不进 clean --all。
func TestBulkCustomKindNotSelectedByDefault(t *testing.T) {
	tool, _ := mkCustomKindTool(t, "my-cache")
	setupBulkEnv(t, tool)

	rep := runCleanDryRun(t)
	if len(rep.Deleted) != 0 {
		t.Errorf("custom kind must not be bulk-selected by default, got %v", rep.Deleted)
	}
}

// TestBulkCustomKindSelectedWhenConfigured 校验配置追加后能进批量。
func TestBulkCustomKindSelectedWhenConfigured(t *testing.T) {
	tool, dir := mkCustomKindTool(t, "my-cache")
	setupBulkEnv(t, tool)
	saveConfig(t, func(c *config.Config) { c.Clean.BulkKinds = []string{"my-cache"} })

	rep := runCleanDryRun(t)
	if len(rep.Deleted) != 1 || rep.Deleted[0] != dir {
		t.Errorf("configured custom kind should be selected, got %v", rep.Deleted)
	}
}

// TestBulkExcludeRemovesBuiltin 校验 bulkExclude 能踢掉内置 Kind。
func TestBulkExcludeRemovesBuiltin(t *testing.T) {
	tool, _ := mkCustomKindTool(t, "cache")
	setupBulkEnv(t, tool)

	// 默认会选中
	if rep := runCleanDryRun(t); len(rep.Deleted) != 1 {
		t.Fatalf("cache should be selected by default, got %v", rep.Deleted)
	}

	// 剔除后不再选中
	saveConfig(t, func(c *config.Config) { c.Clean.BulkExclude = []string{"cache"} })
	if rep := runCleanDryRun(t); len(rep.Deleted) != 0 {
		t.Errorf("excluded kind must not be selected, got %v", rep.Deleted)
	}
}

// TestBulkRuntimeNotSelected 校验 runtime（TierSafe 但不可批量）不进批量。
func TestBulkRuntimeNotSelected(t *testing.T) {
	tool, _ := mkCustomKindTool(t, "runtime")
	setupBulkEnv(t, tool)

	if rep := runCleanDryRun(t); len(rep.Deleted) != 0 {
		t.Errorf("runtime must not be bulk-selected (safe != bulk-cleanable), got %v", rep.Deleted)
	}
	// 但 --include-data 时纳入
	if rep := runCleanDryRun(t, "--include-data"); len(rep.Deleted) != 1 {
		t.Errorf("--include-data should include runtime, got %v", rep.Deleted)
	}
}
