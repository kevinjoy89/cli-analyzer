package rules

import (
	"testing"

	"cli-analyzer/internal/platform"
)

// TestAICatalogEntries 校验第一批 AI 编目条目（add-ai-tool-catalog）：
// 条目存在、类别正确、桌面端宿主应用已登记。
func TestAICatalogEntries(t *testing.T) {
	tbl := Load()
	agents := []string{"zcode", "agy", "workbuddy", "u1s1", "copilot", "dsh", "pi", "kimi", "codex", "claude", "opencode", "mimocode"}
	for _, name := range agents {
		r := tbl.Lookup(name)
		if r == nil {
			t.Errorf("AI 编目缺少条目 %q", name)
			continue
		}
		if r.Category != CategoryAgent {
			t.Errorf("%q category = %q, want %q", name, r.Category, CategoryAgent)
		}
	}
	ides := []string{"code", "trae-cn", "marscode", "joycoder", "codebuddy"}
	for _, name := range ides {
		r := tbl.Lookup(name)
		if r == nil {
			t.Errorf("AI 编目缺少条目 %q", name)
			continue
		}
		if r.Category != CategoryAIIde {
			t.Errorf("%q category = %q, want %q", name, r.Category, CategoryAIIde)
		}
	}
	// 已卸载残留条目必须登记桌面端应用名（已卸载判定依据）
	for _, name := range []string{"trae-cn", "marscode", "zcode", "agy", "joycoder", "codebuddy"} {
		r := tbl.Lookup(name)
		if r != nil && len(r.Apps) == 0 {
			t.Errorf("%q 未登记 Apps（已卸载判定需要桌面端宿主名）", name)
		}
	}
}

// TestAIDataDirKinds 校验 AI 条目的 DataDir kind 都是已知集合内的值
// （拼错 kind 会静默展示英文原名）。
func TestAIDataDirKinds(t *testing.T) {
	known := map[string]bool{
		"config": true, "data": true, "cache": true, "install": true, "state": true,
		KindSession: true, KindSessionDB: true, KindIndex: true, KindRuntime: true,
		KindPlugins: true, KindArtifacts: true, KindLogs: true, KindTelemetry: true,
		KindCheckpoint: true, KindWorkspaceData: true,
	}
	tbl := Load()
	for _, name := range tbl.Names() {
		r := tbl.Lookup(name)
		for _, d := range r.DataDirs {
			if !known[d.Kind] {
				t.Errorf("%s: 未知 kind %q（%s）", name, d.Kind, d.Sub)
			}
		}
	}
}

// TestSessionDBReportOnly 校验会话数据库编目：zcode cli/db 与会话数据库
// 是仅报告目录（不生成可处置项）。
func TestSessionDBReportOnly(t *testing.T) {
	if !IsReportOnly(KindSessionDB) {
		t.Error("session-db 必须是仅报告 kind")
	}
	if IsReportOnly(KindSession) || IsReportOnly("cache") {
		t.Error("其他 kind 不应是仅报告")
	}
	tbl := Load()
	zc := tbl.Lookup("zcode")
	found := false
	for _, d := range zc.DataDirs {
		if d.Sub == ".zcode/cli/db" {
			found = true
			if d.Kind != KindSessionDB {
				t.Errorf("zcode cli/db kind = %q, want session-db", d.Kind)
			}
		}
	}
	if !found {
		t.Error("zcode 缺少 cli/db 会话数据库规则")
	}
}

// TestMarsCodeIndexTier 校验 MarsCode 残留的 .ckg 代码知识图谱索引按
// 可再生缓存（index / TierSafe）标注。
func TestMarsCodeIndexTier(t *testing.T) {
	r := Load().Lookup("marscode")
	if r == nil {
		t.Fatal("缺少 marscode 条目")
	}
	for _, d := range r.DataDirs {
		if d.Sub == ".marscode/.ckg" {
			if d.Kind != KindIndex || d.Tier != TierSafe {
				t.Errorf(".ckg kind=%q tier=%q, want index/safe", d.Kind, d.Tier)
			}
			return
		}
	}
	t.Error("marscode 缺少 .ckg 索引规则")
}

// TestClaudeInstallBaseIsInstall 校验 claude 的版本化安装基目录
// （~/.local/share/claude，含 versions/）标为 install 而非 data：
// 标 data 会把安装本体列为可处置项（误删风险）并吞掉安装占用列。
func TestClaudeInstallBaseIsInstall(t *testing.T) {
	r := Load().Lookup("claude")
	if r == nil {
		t.Fatal("缺少 claude 条目")
	}
	for _, d := range r.DataDirs {
		if d.Root == platform.XDGData && d.Sub == "claude" {
			if d.Kind != "install" {
				t.Errorf("XDGData/claude kind = %q, want install", d.Kind)
			}
			return
		}
	}
	t.Error("claude 缺少 XDGData/claude 规则")
}
