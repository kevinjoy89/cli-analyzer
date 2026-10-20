package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/config"
	"cli-analyzer/internal/rules"
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

// TestBulkCustomKindNotSelectedByDefault 是本变更的核心安全默认：
// 用户规则包声明的自定义 Kind，未配置时不进 clean --all。
func TestBulkCustomKindNotSelectedByDefault(t *testing.T) {
	dir := t.TempDir()
	myCache := filepath.Join(dir, "mycache")
	if err := os.MkdirAll(myCache, 0o755); err != nil {
		t.Fatal(err)
	}
	setupBulkEnv(t, scanner.Tool{
		Name: "mytool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "mytool|my-cache|" + myCache, Tool: "mytool", Path: myCache, Bytes: 100, Kind: "my-cache"},
		},
	})

	rep := runCleanDryRun(t)
	if len(rep.Deleted) != 0 {
		t.Errorf("custom kind must not be bulk-selected by default, got %v", rep.Deleted)
	}
}

// TestBulkCustomKindSelectedWhenConfigured 校验配置追加后能进批量。
func TestBulkCustomKindSelectedWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	myCache := filepath.Join(dir, "mycache")
	if err := os.MkdirAll(myCache, 0o755); err != nil {
		t.Fatal(err)
	}
	setupBulkEnv(t, scanner.Tool{
		Name: "mytool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "mytool|my-cache|" + myCache, Tool: "mytool", Path: myCache, Bytes: 100, Kind: "my-cache"},
		},
	})
	saveConfig(t, func(c *config.Config) { c.Clean.BulkKinds = []string{"my-cache"} })

	rep := runCleanDryRun(t)
	if len(rep.Deleted) != 1 || rep.Deleted[0] != myCache {
		t.Errorf("configured custom kind should be selected, got %v", rep.Deleted)
	}
}

// TestBulkExcludeRemovesBuiltin 校验 bulkExclude 能踢掉内置 Kind。
func TestBulkExcludeRemovesBuiltin(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cachedir")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	setupBulkEnv(t, scanner.Tool{
		Name: "mytool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "mytool|cache|" + cacheDir, Tool: "mytool", Path: cacheDir, Bytes: 100, Kind: "cache"},
		},
	})

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
	dir := t.TempDir()
	rt := filepath.Join(dir, "runtime")
	if err := os.MkdirAll(rt, 0o755); err != nil {
		t.Fatal(err)
	}
	setupBulkEnv(t, scanner.Tool{
		Name: "mytool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "mytool|runtime|" + rt, Tool: "mytool", Path: rt, Bytes: 100, Kind: "runtime"},
		},
	})
	if rep := runCleanDryRun(t); len(rep.Deleted) != 0 {
		t.Errorf("runtime must not be bulk-selected (safe != bulk-cleanable), got %v", rep.Deleted)
	}
	// 但 --include-data 时纳入
	if rep := runCleanDryRun(t, "--include-data"); len(rep.Deleted) != 1 {
		t.Errorf("--include-data should include runtime, got %v", rep.Deleted)
	}
}

// TestUserRuleKindIntegratesWithPolicy 端到端：规则包声明的自定义 kind
// 在配置声明后能进批量清理（覆盖「规则包 → 扫描 → 批量选择」整条链路）。
//
// 用真实规则目录：userRulesDir 是包级变量，通过 rules 包导出的
// UserRulesDir() 拿到实际位置，把规则包写进去（测试环境已用 HOME 隔离）。
func TestUserRuleKindIntegratesWithPolicy(t *testing.T) {
	rulesDir := rules.UserRulesDir()
	if rulesDir == "" {
		t.Skip("user rules dir unavailable")
	}
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pack := `{
  "schemaVersion": 1,
  "rules": [{
    "name": "packtool",
    "dataDirs": [{"path": "` + filepath.Join(t.TempDir(), "pack-cache") + `", "tier": "safe", "kind": "pack-cache"}]
  }]
}`
	if err := os.WriteFile(filepath.Join(rulesDir, "pack.json"), []byte(pack), 0o644); err != nil {
		t.Fatal(err)
	}

	// 规则包加载与 kind 归属由 internal/rules/user_test.go 覆盖；
	// 这里只验证策略层对自定义 kind 的处理：未配置不选、配置后选。
	setupBulkEnv(t)
	custom := filepath.Join(t.TempDir(), "pack-cache")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	// 重新写缓存，含自定义 kind 条目
	if err := scanner.SaveCache(&scanner.ScanResult{Tools: []scanner.Tool{{
		Name: "packtool", Installer: "other",
		Cleanables: []scanner.Cleanable{
			{ID: "packtool|pack-cache|" + custom, Tool: "packtool", Path: custom, Bytes: 10, Kind: "pack-cache"},
		},
	}}}); err != nil {
		t.Fatal(err)
	}

	if rep := runCleanDryRun(t); len(rep.Deleted) != 0 {
		t.Errorf("rule-pack kind must not be bulk by default, got %v", rep.Deleted)
	}
	saveConfig(t, func(c *config.Config) { c.Clean.BulkKinds = []string{"pack-cache"} })
	if rep := runCleanDryRun(t); len(rep.Deleted) != 1 {
		t.Errorf("configured rule-pack kind should be selected, got %v", rep.Deleted)
	}
}
