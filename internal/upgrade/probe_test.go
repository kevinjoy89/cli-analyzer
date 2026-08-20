package upgrade

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/scanner"
)

// ---- -h 探测解析 ----

func TestParseSelfUpgradeSub(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want string
	}{
		{
			"claude",
			"Usage: claude [options] [command] [prompt]\n\nCommands:\n  install [options] [target]  Install Claude Code native build.\n  update|upgrade          Check for updates and install if available\n  doctor                    Run diagnostics\n",
			"update",
		},
		{
			"kimi",
			"Usage: kimi [options] [command]\n\nCommands:\n  auth     Login\n  upgrade|update    Upgrade Kimi Code to the latest version.\n  version  Print version\n",
			"upgrade",
		},
		{
			"codegraph",
			"Usage: codegraph [options] [command]\n\nCommands:\n  init      Initialize CodeGraph\n  upgrade [options] [version]  Update CodeGraph to the latest release\n  version   Print version\n",
			"upgrade",
		},
		{
			"uv-grouped",
			"Usage: uv [OPTIONS] <COMMAND>\n\nCommands:\n  self       Manage the uv executable\n  python     Manage Python versions\n",
			"",
		},
		{
			"update-not-command",
			"Usage: tool [options]\n\nOptions:\n  --update-config  Update config\n",
			"",
		},
	}
	// 带程序名的形态需要 binBase 才能识别（opencode upgrade → 取 upgrade）
	if got := parseSelfUpgradeSub(`Commands:
  opencode completion         generate shell completion
  opencode upgrade [target]   upgrade opencode to the latest
  opencode plugin <module>    install plugin and update config
`, "opencode"); got != "upgrade" {
		t.Errorf("opencode-style with binBase = %q, want upgrade", got)
	}
	// 描述文本里的 update 不误命中（opencode plugin 行的 "update config"）
	if got := parseSelfUpgradeSub("Commands:\n  opencode plugin <module>  install plugin and update config\n", "opencode"); got != "" {
		t.Errorf("description update should not match, got %q", got)
	}
	// 大小写不敏感：UPDATE → update
	if got := parseSelfUpgradeSub("Usage: tool\n\nCommands:\n  UPDATE  do something\n", ""); got != "update" {
		t.Errorf("uppercase UPDATE = %q, want update", got)
	}
	for _, c := range cases {
		if got := parseSelfUpgradeSub(c.out, ""); got != c.want {
			t.Errorf("%s: parseSelfUpgradeSub = %q, want %q", c.name, got, c.want)
		}
	}
}

// ---- 探测执行（真实子进程，unix only）----

// probeBin 写入一个响应 `-h` 的小脚本，返回路径。
func probeBin(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses unix shell")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "fakecli")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func TestProbeSelfUpgradeHits(t *testing.T) {
	bin := probeBin(t, `case "$1" in -h|--help) printf 'Commands:\n  upgrade|update  Upgrade to latest\n';; esac`)
	if got := ProbeSelfUpgrade(context.Background(), bin); got != "upgrade" {
		t.Errorf("ProbeSelfUpgrade = %q, want upgrade", got)
	}
}

func TestProbeSelfUpgradeMiss(t *testing.T) {
	bin := probeBin(t, `printf 'Usage: tool [options]\n'`)
	if got := ProbeSelfUpgrade(context.Background(), bin); got != "" {
		t.Errorf("ProbeSelfUpgrade = %q, want empty", got)
	}
}

func TestProbeSelfUpgradeNoHelpFlag(t *testing.T) {
	// 只响应 --help，不响应 -h：应回退到 --help
	bin := probeBin(t, `case "$1" in --help) printf 'Commands:\n  update  Update\n';; *) exit 1;; esac`)
	if got := ProbeSelfUpgrade(context.Background(), bin); got != "update" {
		t.Errorf("ProbeSelfUpgrade = %q, want update (fallback --help)", got)
	}
}

func TestProbeSelfUpgradeCache(t *testing.T) {
	// 假 helpOutput 计数：首次探测后缓存命中不再执行子进程
	bin := probeBin(t, `printf 'Commands:\n  update  Update\n'`)
	calls := 0
	helpOutput = func(_ context.Context, _ string, _ ...string) string {
		calls++
		return "Commands:\n  update  Update\n"
	}
	defer func() { helpOutput = defaultHelpOutput }()
	if got := ProbeSelfUpgrade(context.Background(), bin); got != "update" {
		t.Fatalf("first call = %q, want update", got)
	}
	if got := ProbeSelfUpgrade(context.Background(), bin); got != "update" {
		t.Fatalf("cached call = %q, want update", got)
	}
	if calls != 1 {
		t.Errorf("exec calls = %d, want 1 (cached)", calls)
	}
}

// ---- probeCommand 编排 ----

func TestProbeCommandHits(t *testing.T) {
	bin := probeBin(t, `printf 'Commands:\n  upgrade|update  Upgrade to latest\n'`)
	cmd := probeCommand(context.Background(), scanner.InstOther, "fakecli", bin)
	if !cmd.Runnable || cmd.Command != "fakecli upgrade" {
		t.Errorf("probeCommand = {%q runnable=%v}, want fakecli upgrade/true", cmd.Command, cmd.Runnable)
	}
	if cmd.Bin != bin || len(cmd.Args) != 1 || cmd.Args[0] != "upgrade" {
		t.Errorf("Bin/Args = %s %v, want %s [upgrade]", cmd.Bin, cmd.Args, bin)
	}
}

func TestProbeCommandMiss(t *testing.T) {
	bin := probeBin(t, `printf 'Usage: fakecli [options]\n'`)
	cmd := probeCommand(context.Background(), scanner.InstOther, "fakecli", bin)
	if cmd.Runnable {
		t.Errorf("probeCommand miss should not be runnable, got %+v", cmd)
	}
	if cmd.Command == "" {
		t.Error("probeCommand miss should fall back to generic hint")
	}
}

// ---- Windows 兼容（静态表键/程序名匹配）----

func TestStripExeExt(t *testing.T) {
	cases := map[string]string{
		"claude":     "claude",
		"claude.exe": "claude",
		"claude.cmd": "claude",
		"npm.bat":    "npm",
		"tool.COM":   "tool",
		"uv":         "uv",
	}
	for in, want := range cases {
		if got := stripExeExt(in); got != want {
			t.Errorf("stripExeExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsCmdShim(t *testing.T) {
	if !isCmdShim("npm.cmd") || !isCmdShim("C:\\Users\\x\\bin\\tool.bat") {
		t.Error("cmd/bat should be shims")
	}
	if isCmdShim("claude.exe") || isCmdShim("/usr/bin/claude") || isCmdShim("npm") {
		t.Error("exe/bare names should not be shims")
	}
}

// TestSelfUpgradeTableWindowsName 验证 Windows 带扩展名的二进制名也能命中
// 静态表（claude.exe → claude update），HasCommand 与 OfficialCommand 对齐。
func TestSelfUpgradeTableWindowsName(t *testing.T) {
	if !HasCommand(scanner.InstVersioned, "claude.exe") {
		t.Error("HasCommand(claude.exe) should hit self-upgrade table")
	}
	got := OfficialCommand(scanner.InstVersioned, "claude.exe", "claude.exe")
	if got.Command != "claude.exe update" || !got.Runnable {
		t.Errorf("OfficialCommand(claude.exe) = {%q runnable=%v}, want claude.exe update/true", got.Command, got.Runnable)
	}
}

// TestProbeHelpSubWindowsBinBase 验证探测解析时程序名剥扩展名：
// Windows 二进制名是 opencode.exe（help 里命令名是裸 opencode），
// "opencode upgrade" 形态应命中。filepath.Base 是平台相关的（unix 不按
// 反斜杠切），故直接传剥好扩展名的 base 名验证解析逻辑。
func TestProbeHelpSubWindowsBinBase(t *testing.T) {
	out := "Commands:\n  opencode completion  generate shell completion\n  opencode upgrade [target]  upgrade opencode to the latest\n"
	if got := parseSelfUpgradeSub(out, stripExeExt("opencode.exe")); got != "upgrade" {
		t.Errorf("windows binBase parse = %q, want upgrade", got)
	}
}
