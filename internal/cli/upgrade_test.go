package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"cli-analyzer/internal/config"
	"cli-analyzer/internal/scanner"
)

// setupUpgradeToolCache 隔离配置目录与扫描缓存根，并写入含指定工具的扫描缓存。
func setupUpgradeToolCache(t *testing.T, tools ...scanner.Tool) {
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

// TestRunUpdateCheckToolGoJSON 验证 `update check <工具> --json` 对无检测
// 能力来源输出结构化 JSON 且不执行任何查询（go 来源只给提示）。
func TestRunUpdateCheckToolGoJSON(t *testing.T) {
	setupUpgradeToolCache(t, scanner.Tool{Name: "dlv", Installer: string(scanner.InstGo)})
	buf := captureStdout(t)
	if code := Run([]string{"update", "check", "dlv", "--json"}); code != 0 {
		t.Fatalf("exit = %d, want 0 (无更新)", code)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON parse: %v\n%s", err, buf.String())
	}
	if out["name"] != "dlv" || out["detected"] != false || out["hasUpdate"] != false {
		t.Errorf("JSON = %v, want detected=false hasUpdate=false", out)
	}
	if out["command"] == "" || out["runnable"] != false {
		t.Errorf("go source should carry hint only, got %v", out)
	}
}

// TestRunUpdateCheckToolNotFound 验证工具不存在时报错且退出码非零。
func TestRunUpdateCheckToolNotFound(t *testing.T) {
	setupUpgradeToolCache(t)
	buf := captureStderr(t)
	if code := Run([]string{"update", "check", "no-such-tool"}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "未找到") {
		t.Errorf("should report tool not found, got:\n%s", buf.String())
	}
}

// TestRunUpdateCheckNoScanResult 验证缓存缺失时报"没有扫描结果"而非"未找到工具"。
func TestRunUpdateCheckNoScanResult(t *testing.T) {
	cacheEnv(t)
	pinZhCN(t) // 固定 zh-CN：无扫描结果文案走 i18n，CI/Linux 英文环境也稳定断言
	buf := captureStderr(t)
	if code := Run([]string{"update", "check", "ripgrep"}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(buf.String(), "扫描结果") {
		t.Errorf("should report no scan result, got:\n%s", buf.String())
	}
}

// TestRunUpdateCheckNoArgBackwardCompat 验证 `update check` 无参保持应用
// 自身检查行为（向后兼容，design D7）。
func TestRunUpdateCheckNoArgBackwardCompat(t *testing.T) {
	setupUpdateTest(t, "0.2.3", []mockRelease{{TagName: "v0.3.0", HTMLURL: "https://github.com/x/releases/tag/v0.3.0"}})
	buf := captureStdout(t)
	if code := Run([]string{"update", "check"}); code != 2 {
		t.Fatalf("app update available exit = %d, want 2", code)
	}
	if !strings.Contains(buf.String(), "v0.3.0") {
		t.Errorf("should report app new version, got:\n%s", buf.String())
	}
}

// TestRunUpdateRunToolJSON 验证 `update run <工具> --json` 只输出命令信息、
// 不执行（镜像 uninstall 的 --json 契约）。
func TestRunUpdateRunToolJSON(t *testing.T) {
	setupUpgradeToolCache(t, scanner.Tool{Name: "ripgrep", Installer: string(scanner.InstBrew)})
	buf := captureStdout(t)
	if code := Run([]string{"update", "run", "ripgrep", "--json"}); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON parse: %v\n%s", err, buf.String())
	}
	if out["command"] != "brew upgrade ripgrep" || out["runnable"] != true {
		t.Errorf("JSON = %v, want brew upgrade ripgrep runnable=true", out)
	}
}

// TestRunUpdateRunToolNpmMapped 验证 npmToolID 映射的短工具名（pi）在
// `update run --json` 中输出真实包名命令（而非静默误报/代跑失败）。
func TestRunUpdateRunToolNpmMapped(t *testing.T) {
	setupUpgradeToolCache(t, scanner.Tool{Name: "pi", Installer: string(scanner.InstNpm)})
	buf := captureStdout(t)
	if code := Run([]string{"update", "run", "pi", "--json"}); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("JSON parse: %v\n%s", err, buf.String())
	}
	if out["command"] != "npm update -g @earendil-works/pi-coding-agent" || out["runnable"] != true {
		t.Errorf("JSON = %v, want real pkg name command runnable=true", out)
	}
}

// TestRunUpdateRunToolNotRunnable 验证非代跑来源（go）返回不可代跑（退出码 2）。
func TestRunUpdateRunToolNotRunnable(t *testing.T) {
	setupUpgradeToolCache(t, scanner.Tool{Name: "dlv", Installer: string(scanner.InstGo)})
	buf := captureStdout(t)
	if code := Run([]string{"update", "run", "dlv"}); code != 2 {
		t.Fatalf("exit = %d, want 2 (not runnable)", code)
	}
	if !strings.Contains(buf.String(), "不支持代跑") {
		t.Errorf("should report not runnable, got:\n%s", buf.String())
	}
}

// TestRunUpdateRunToolNotFound 验证 `update run` 工具不存在（退出码 2）。
func TestRunUpdateRunToolNotFound(t *testing.T) {
	setupUpgradeToolCache(t)
	buf := captureStderr(t)
	if code := Run([]string{"update", "run", "no-such-tool"}); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(buf.String(), "未找到") {
		t.Errorf("should report tool not found, got:\n%s", buf.String())
	}
}
