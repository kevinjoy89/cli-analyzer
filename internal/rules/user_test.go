package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRulePack 在临时目录写一个规则包并把它设为当前用户规则目录。
func writeRulePack(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := userRulesDir
	userRulesDir = func() string { return dir }
	t.Cleanup(func() { userRulesDir = orig })
	return dir
}

// TestUserRuleAddsNewTool 验证用户规则包能追加新工具，并走同一套归因管线。
func TestUserRuleAddsNewTool(t *testing.T) {
	writeRulePack(t, "mine.json", `{
  "schemaVersion": 1,
  "rules": [{
    "name": "mytool",
    "displayName": "My Tool",
    "installer": "other",
    "dataDirs": [{"root": "home", "sub": ".mytool", "tier": "user", "kind": "data"}],
    "cleanables": [{"root": "home", "sub": ".mytool/cache", "tier": "safe", "kind": "cache"}]
  }]
}`)
	tbl := Load()
	r := tbl.Lookup("mytool")
	if r == nil {
		t.Fatal("user rule not loaded")
	}
	if r.DisplayName != "My Tool" || len(r.DataDirs) != 1 || len(r.Cleanables) != 1 {
		t.Errorf("rule = %+v", r)
	}
	if got := r.DataDirs[0].Resolve(); got == "" {
		t.Error("data dir should resolve on this platform")
	}
	if errs := tbl.UserErrors(); len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// TestUserRuleReplacesBuiltin 验证同名条目整条替换内置条目（而非合并）。
func TestUserRuleReplacesBuiltin(t *testing.T) {
	writeRulePack(t, "gh.json", `{
  "rules": [{"name": "gh", "displayName": "My gh", "installer": "brew"}]
}`)
	tbl := Load()
	r := tbl.Lookup("gh")
	if r == nil {
		t.Fatal("gh missing")
	}
	if r.DisplayName != "My gh" {
		t.Errorf("displayName = %q, want override", r.DisplayName)
	}
	// 整条替换：内置的 DataDirs 不应残留
	if len(r.DataDirs) != 0 {
		t.Errorf("builtin dataDirs leaked into override: %+v", r.DataDirs)
	}
}

// TestUserRuleInvalidEntrySkipped 验证单条不合法只跳过该条，内置表照常可用。
func TestUserRuleInvalidEntrySkipped(t *testing.T) {
	writeRulePack(t, "bad.json", `{
  "rules": [
    {"name": "", "dataDirs": [{"root": "home"}]},
    {"name": "reltool", "dataDirs": [{"path": "relative/path"}]},
    {"name": "badroot", "dataDirs": [{"root": "nope"}]},
    {"name": "goodtool", "dataDirs": [{"root": "home", "sub": ".good"}]}
  ]
}`)
	tbl := Load()
	if tbl.Lookup("goodtool") == nil {
		t.Error("valid entry in the same file should still load")
	}
	if tbl.Lookup("reltool") != nil || tbl.Lookup("badroot") != nil {
		t.Error("invalid entries must be skipped")
	}
	if tbl.Lookup("claude") == nil {
		t.Error("builtin rules must survive a broken user pack")
	}
	if got := len(tbl.UserErrors()); got != 3 {
		t.Errorf("errors = %d, want 3: %v", got, tbl.UserErrors())
	}
}

// TestUserRuleBrokenJSONDoesNotFailLoad 验证整份 JSON 坏掉时扫描仍可用。
func TestUserRuleBrokenJSONDoesNotFailLoad(t *testing.T) {
	writeRulePack(t, "broken.json", `{ not json`)
	tbl := Load()
	if tbl.Lookup("claude") == nil {
		t.Fatal("builtin rules must survive a broken file")
	}
	if len(tbl.UserErrors()) != 1 {
		t.Errorf("errors = %v, want 1", tbl.UserErrors())
	}
}

// TestUserRuleUnsupportedSchema 验证未知 schemaVersion 被拒绝而不是猜着解析。
func TestUserRuleUnsupportedSchema(t *testing.T) {
	writeRulePack(t, "future.json", `{"schemaVersion": 99, "rules": [{"name": "x"}]}`)
	tbl := Load()
	if tbl.Lookup("x") != nil {
		t.Error("unsupported schemaVersion must not be applied")
	}
	if len(tbl.UserErrors()) != 1 {
		t.Errorf("errors = %v", tbl.UserErrors())
	}
}
