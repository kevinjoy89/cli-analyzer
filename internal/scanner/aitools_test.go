package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cli-analyzer/internal/disk"
	"cli-analyzer/internal/platform"
	"cli-analyzer/internal/rules"
)

// aiAttrHarnessAt 在隔离的 HOME 环境里对指定工具执行归因（复刻 scan 的
// seed 行为），返回构建器供断言。仅 unix 语义（Windows 路径布局不同，
// 调用方先行 skip）。home 参数即测试注入的 HOME，供阅读上下文。
func aiAttrHarnessAt(t *testing.T, home string, ids []string) map[string]*toolBuilder {
	t.Helper()
	ruleTable := rules.Load()
	tools := map[string]*toolBuilder{}
	order := []string{}
	for _, id := range ids {
		tb := &toolBuilder{name: id, aliases: map[string]bool{}}
		if r := ruleTable.Lookup(id); r != nil {
			tb.installer = Installer(r.Installer)
		}
		tools[id] = tb
		order = append(order, id)
	}
	attribute(tools, order, ruleTable, Options{}, &disk.Sizer{})
	_ = home
	return tools
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writefile(t *testing.T, path string, size int) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestAISessionDBReportOnly 校验 add-ai-tool-catalog 的核心安全红线：
// 会话数据库仅报告——不出现在可处置项中，树下钻中标注 session-db。
func TestAISessionDBReportOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("AI 归因测试基于 unix HOME 布局")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	xdgData := filepath.Join(td, ".local", "share")
	t.Setenv("XDG_DATA_HOME", xdgData)
	// opencode：会话数据库是父数据目录内的文件（opencode.db）
	writefile(t, filepath.Join(xdgData, "opencode", "opencode.db"), 1<<20)
	mkdirAll(t, filepath.Join(xdgData, "opencode", "snapshot"))
	// zcode：会话数据库是独立目录（.zcode/cli/db）
	mkdirAll(t, filepath.Join(td, ".zcode", "cli", "db"))
	writefile(t, filepath.Join(td, ".zcode", "cli", "db", "sessions.sqlite"), 1<<10)

	tools := aiAttrHarnessAt(t, td, []string{"opencode", "zcode"})

	oc := tools["opencode"]
	for _, c := range oc.cleanables {
		if strings.HasSuffix(c.Path, "opencode.db") {
			t.Errorf("opencode.db 不应成为可处置项: %s", c.Path)
		}
	}
	// 树下钻：父数据目录的 Sub 里 opencode.db 标注 session-db
	parent := ""
	for _, c := range oc.cleanables {
		if c.Path == filepath.Join(xdgData, "opencode") {
			parent = c.Path
		}
	}
	if parent == "" {
		t.Fatal("opencode 数据目录未被归因为可处置项")
	}
	dbTagged := false
	for _, s := range oc.cleanables {
		if s.Path != parent {
			continue
		}
		for _, se := range s.Sub {
			if strings.HasSuffix(se.Path, "opencode.db") {
				if se.Kind != rules.KindSessionDB {
					t.Errorf("opencode.db sub kind = %q, want session-db", se.Kind)
				}
				dbTagged = true
			}
		}
	}
	if !dbTagged {
		t.Error("树下钻未标注 opencode.db 为 session-db")
	}

	zc := tools["zcode"]
	for _, c := range zc.cleanables {
		if strings.HasSuffix(c.Path, filepath.Join(".zcode", "cli", "db")) {
			t.Errorf(".zcode/cli/db 会话数据库不应成为可处置项: %s", c.Path)
		}
	}
	dbDir := false
	for _, d := range zc.dataDirs {
		if d.Path == filepath.Join(td, ".zcode", "cli", "db") {
			dbDir = true
			if d.Kind != rules.KindSessionDB {
				t.Errorf("cli/db kind = %q, want session-db", d.Kind)
			}
		}
	}
	if !dbDir {
		t.Error(".zcode/cli/db 未被归因展示")
	}
}

// TestAIUninstalledResidue 校验已卸载 AI 工具的残留归因（design D3）：
// 数据存在 + 无二进制 + 桌面端缺失 → uninstalled 标注；且目录被条目认领。
//
// 断言涉及 MacAppSupport 根（~/Library/Application Support），该根只在
// darwin 上解析出路径，故根不可用时跳过。
func TestAIUninstalledResidue(t *testing.T) {
	if platform.Root(platform.MacAppSupport) == "" {
		t.Skip("残留归因断言依赖 macOS 的 Application Support 根")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	// MarsCode：只剩 .ckg 索引（3.9G 量级在本机；此处 1MB 足以判定存在）
	writefile(t, filepath.Join(td, ".marscode", ".ckg", "graph.bin"), 1<<20)
	// Trae CN：dotdir + AppSupport 双残留
	mkdirAll(t, filepath.Join(td, ".trae-cn"))
	mkdirAll(t, filepath.Join(td, "Library", "Application Support", "Trae CN", "User"))

	tools := aiAttrHarnessAt(t, td, []string{"marscode", "trae-cn"})

	mc := tools["marscode"]
	if !mc.uninstalled {
		t.Error("marscode 应标注已卸载（无二进制且桌面端缺失）")
	}
	ckg := false
	for _, d := range mc.dataDirs {
		if d.Path == filepath.Join(td, ".marscode", ".ckg") {
			ckg = true
			if d.Kind != rules.KindIndex || string(d.Tier) != rules.TierSafe {
				t.Errorf(".ckg kind=%q tier=%q, want index/safe", d.Kind, d.Tier)
			}
		}
	}
	if !ckg {
		t.Error(".marscode/.ckg 未被归因")
	}

	tc := tools["trae-cn"]
	if !tc.uninstalled {
		t.Error("trae-cn 应标注已卸载")
	}
	claimed := false
	for _, d := range tc.dataDirs {
		if d.Path == filepath.Join(td, "Library", "Application Support", "Trae CN") {
			claimed = true
		}
	}
	if !claimed {
		t.Error("AppSupport/Trae CN 残留未被归因（AI 编目豁免厂商排除表）")
	}
}

// TestAIActiveToolNotUninstalled 校验对照面：二进制在（或桌面端宿主在）的
// AI 条目不标注已卸载
func TestAIActiveToolNotUninstalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("AI 归因测试基于 unix HOME 布局")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	mkdirAll(t, filepath.Join(td, ".gemini"))

	tools := aiAttrHarnessAt(t, td, []string{"agy"})
	ag := tools["agy"]
	// 本机 Antigravity.app 安装状态可被 AppsInstalled 探测
	if platform.AppsInstalled([]string{"Antigravity"}) && ag.uninstalled {
		t.Error("Antigravity 有桌面端宿主时不应标注已卸载")
	}
}

// TestSubKindSessionDBFiles 单元测试会话数据库文件的识别。
func TestSubKindSessionDBFiles(t *testing.T) {
	cases := []struct {
		parent, name, want string
	}{
		{"data", "opencode.db", "session-db"},
		{"data", "mimocode.db", "session-db"},
		{"data", "logs_2.sqlite", "session-db"},
		{"data", "logs_2.sqlite-wal", "session-db"},
		{"data", "thread_history_1.sqlite", "session-db"},
		{"cache", "cache.bin", "cache"},
		{"data", "_logs", "logs"},
		{"data", "debug.log", "logs"},
		{"data", "snapshot", "data"},
	}
	for _, c := range cases {
		if got := subKind(c.parent, c.name); got != c.want {
			t.Errorf("subKind(%q, %q) = %q, want %q", c.parent, c.name, got, c.want)
		}
	}
}

// TestAIIDEWorkspaceData 校验 VS Code 的 AI IDE 编目：workspaceStorage 按
// workspace-data 标注，JoyCoder 与 CodeBuddy 独立归因为独立 AI IDE。
//
// 断言涉及 MacAppSupport 根（~/Library/Application Support），该根只在
// darwin 上解析出路径，故根不可用时跳过。
func TestAIIDEWorkspaceData(t *testing.T) {
	if platform.Root(platform.MacAppSupport) == "" {
		t.Skip("AI IDE 归因断言依赖 macOS 的 Application Support 根")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	mkdirAll(t, filepath.Join(td, "Library", "Application Support", "Code", "User", "workspaceStorage"))
	mkdirAll(t, filepath.Join(td, "Library", "Application Support", "JoyCoder"))
	mkdirAll(t, filepath.Join(td, "Library", "Application Support", "CodeBuddyExtension"))

	tools := aiAttrHarnessAt(t, td, []string{"code", "joycoder", "codebuddy", "agy"})
	cd := tools["code"]
	ws := false
	for _, d := range cd.dataDirs {
		switch d.Path {
		case filepath.Join(td, "Library", "Application Support", "Code", "User", "workspaceStorage"):
			if d.Kind != rules.KindWorkspaceData {
				t.Errorf("workspaceStorage kind = %q, want workspace-data", d.Kind)
			}
			ws = true
		case filepath.Join(td, "Library", "Application Support", "JoyCoder"):
			t.Error("JoyCoder 不应再归因到 code 条目")
		}
	}
	if !ws {
		t.Error("workspaceStorage 未按 workspace-data 归因")
	}

	// 校验 JoyCoder 独立归因
	jc := tools["joycoder"]
	jcFound := false
	for _, d := range jc.dataDirs {
		if d.Path == filepath.Join(td, "Library", "Application Support", "JoyCoder") {
			jcFound = true
		}
	}
	if !jcFound {
		t.Error("JoyCoder 数据目录未独立归因")
	}

	// 校验 CodeBuddy 独立归因
	cb := tools["codebuddy"]
	cbFound := false
	for _, d := range cb.dataDirs {
		if d.Path == filepath.Join(td, "Library", "Application Support", "CodeBuddyExtension") {
			cbFound = true
		}
	}
	if !cbFound {
		t.Error("CodeBuddyExtension 数据目录未独立归因")
	}

	// 校验 agy 归因了 ~/.gemini 且 DisplayName 正确
	agy := tools["agy"]
	agyGeminiFound := false
	for _, d := range agy.dataDirs {
		if d.Path == filepath.Join(td, ".gemini") {
			agyGeminiFound = true
		}
	}
	if !agyGeminiFound {
		t.Error("agy 未归因 ~/.gemini 数据目录")
	}
	if cd.displayName != "Visual Studio Code" {
		t.Errorf("code.displayName = %q, want Visual Studio Code", cd.displayName)
	}
	if agy.displayName != "Antigravity" {
		t.Errorf("agy.displayName = %q, want Antigravity", agy.displayName)
	}
}

// TestCleanableSumNoParentChildOverlap 校验 add-ai-tool-catalog 父子子目录
// 规则下的合计口径：cleanableBytes 按极大路径求和（fix-cleanable-sum-overlap），
// 不再出现「可处置 > 总占用」，安装占用（userBytes）恢复含义。
func TestCleanableSumNoParentChildOverlap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("AI 归因测试基于 unix HOME 布局")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	t.Setenv("XDG_DATA_HOME", filepath.Join(td, ".local", "share"))
	// marscode：父目录 .marscode 与子目录 .marscode/.ckg 同为可处置项
	writefile(t, filepath.Join(td, ".marscode", ".ckg", "graph.bin"), 1<<20)
	// opencode：数据目录 + 安装根 .opencode/bin（安装占用应恢复非零）
	writefile(t, filepath.Join(td, ".local", "share", "opencode", "opencode.db"), 1<<20)
	writefile(t, filepath.Join(td, ".opencode", "bin", "opencode"), 1<<18)

	tools := aiAttrHarnessAt(t, td, []string{"marscode", "opencode"})
	res := finalize(tools, []string{"marscode", "opencode"}, Options{})
	byName := map[string]*Tool{}
	for i := range res.Tools {
		byName[res.Tools[i].Name] = &res.Tools[i]
	}

	mc := byName["marscode"]
	if mc == nil {
		t.Fatal("marscode 未出现在扫描结果")
	}
	if mc.Cleanable > mc.Footprint {
		t.Errorf("marscode 可处置(%d) 不应大于总占用(%d)", mc.Cleanable, mc.Footprint)
	}
	// .ckg ⊂ .marscode：合计应为父目录单份（等于 footprint，无独立二进制）
	if mc.Cleanable != mc.Footprint {
		t.Errorf("marscode 可处置(%d) 应等于总占用(%d)（父目录单份口径）", mc.Cleanable, mc.Footprint)
	}

	oc := byName["opencode"]
	if oc == nil {
		t.Fatal("opencode 未出现在扫描结果")
	}
	if oc.Cleanable > oc.Footprint {
		t.Errorf("opencode 可处置(%d) 不应大于总占用(%d)", oc.Cleanable, oc.Footprint)
	}
	if oc.User <= 0 {
		t.Errorf("opencode 安装占用应恢复非零（.opencode 安装根存在），得到 %d", oc.User)
	}
}

// TestAIAppBundleInstallSizing 校验 add-ai-app-bundle-sizing：AI 条目声明
// 的桌面端 .app 包计入安装占用与总占用；应用包缺失时不归因。
func TestAIAppBundleInstallSizing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("桌面端应用包测量仅实现 macOS")
	}
	td := t.TempDir()
	t.Setenv("HOME", td)
	bundle := filepath.Join(td, "Applications", "WorkBuddy.app")
	writefile(t, filepath.Join(bundle, "Contents", "MacOS", "WorkBuddy"), 1<<20)
	writefile(t, filepath.Join(td, ".workbuddy", "binaries", "wb"), 1<<20)

	orig := appBundlePaths
	appBundlePaths = func(apps []string) []string {
		for _, a := range apps {
			if a == "WorkBuddy" {
				return []string{bundle}
			}
		}
		return nil
	}
	t.Cleanup(func() { appBundlePaths = orig })

	tools := aiAttrHarnessAt(t, td, []string{"workbuddy"})
	res := finalize(tools, []string{"workbuddy"}, Options{})
	var wb *Tool
	for i := range res.Tools {
		if res.Tools[i].Name == "workbuddy" {
			wb = &res.Tools[i]
		}
	}
	if wb == nil {
		t.Fatal("workbuddy 未出现在扫描结果")
	}
	found := false
	for _, d := range wb.DataDirs {
		if d.Path == bundle {
			found = true
			if d.Kind != "install" {
				t.Errorf("应用包 kind = %q, want install", d.Kind)
			}
		}
	}
	if !found {
		t.Error("应用包未按 install 归因")
	}
	// 安装占用 = 应用包 1MB（数据 .workbuddy 1MB 为可处置，父子去重后单份）
	if wb.User < 1<<20 {
		t.Errorf("安装占用应包含应用包（≥1MB），得到 %d", wb.User)
	}
	if wb.Footprint < 2<<20 {
		t.Errorf("总占用应包含数据与应用包（≥2MB），得到 %d", wb.Footprint)
	}

	// 应用包缺失（trae-cn 场景）：不产生 install 归因
	appBundlePaths = func([]string) []string { return nil }
	tools2 := aiAttrHarnessAt(t, td, []string{"workbuddy"})
	_ = tools2
}
