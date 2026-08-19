package upgrade

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"testing"

	"cli-analyzer/internal/scanner"
)

// ---- CheckTool 编排 ----

func TestCheckToolBrewHasUpdate(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return `{"formulae":[{"name":"ripgrep","installed_versions":["14.1.1"],"current_version":"14.1.2"}]}`, nil
	})
	res := CheckTool(scanner.Tool{Name: "ripgrep", Installer: string(scanner.InstBrew)})
	if !res.Detected || !res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v, want true/true", res.Detected, res.HasUpdate)
	}
	if res.Current != "14.1.1" || res.Latest != "14.1.2" {
		t.Errorf("current=%q latest=%q", res.Current, res.Latest)
	}
	if res.Command != "brew upgrade ripgrep" || !res.Runnable {
		t.Errorf("command=%q runnable=%v, want brew upgrade ripgrep/true", res.Command, res.Runnable)
	}
	if res.Error != "" {
		t.Errorf("error=%q, want empty", res.Error)
	}
}

func TestCheckToolNpmUpToDate(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return "", nil
	})
	res := CheckTool(scanner.Tool{Name: "cli-analyzer", Installer: string(scanner.InstNpm)})
	if !res.Detected || res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v, want true/false", res.Detected, res.HasUpdate)
	}
	if res.Error != "" {
		t.Errorf("error=%q, want empty", res.Error)
	}
}

func TestCheckToolFailureStillGivesCommand(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return "", errors.New("network unreachable")
	})
	res := CheckTool(scanner.Tool{Name: "ripgrep", Installer: string(scanner.InstBrew)})
	if res.Detected || res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v, want false/false", res.Detected, res.HasUpdate)
	}
	if res.Command != "brew upgrade ripgrep" || !res.Runnable {
		t.Errorf("degraded check must still carry command, got %q runnable=%v", res.Command, res.Runnable)
	}
	if res.Error == "" {
		t.Error("failure should carry a readable error")
	}
}

func TestCheckToolNoCapability(t *testing.T) {
	res := CheckTool(scanner.Tool{Name: "dlv", Installer: string(scanner.InstGo)})
	if res.Detected || res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v, want false/false", res.Detected, res.HasUpdate)
	}
	if res.Command == "" || res.Runnable {
		t.Errorf("go should get hint only, got command=%q runnable=%v", res.Command, res.Runnable)
	}
}

// TestOfficialForCargoMapped 验证 cargo 工具按二进制名归类（rg）时，代跑命令
// 用真实 crate 名（ripgrep），而非二进制名（code review #6 修复）。
func TestOfficialForCargoMapped(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return "ripgrep v14.1.1:\n    /usr/local/bin/rg\n", nil
	})
	cmd := OfficialFor(scanner.Tool{Name: "rg", Installer: string(scanner.InstCargo), Binaries: []scanner.Binary{{Name: "rg"}}})
	if cmd.Command != "cargo install ripgrep --force" {
		t.Errorf("command = %q, want cargo install ripgrep --force（crate 名）", cmd.Command)
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "ripgrep" {
		t.Errorf("args = %v, want [install ripgrep --force]", cmd.Args)
	}
}

// TestCheckToolCargoMapped 验证 cargo 工具（二进制 rg / crate ripgrep）检测时
// 命令与检测都用 crate 名。
func TestCheckToolCargoMapped(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		// 第一次是 cargoCrateOf 的 install --list，第二次是 detectCargo 的 install --list，
		// 第三次是 cargo search
		switch calls {
		case 1, 2:
			return "ripgrep v14.1.1:\n    /usr/local/bin/rg\n", nil
		case 3:
			return "ripgrep = \"14.1.2\"    # A fast search tool\n", nil
		}
		return "", errors.New("unexpected extra call")
	})
	res := CheckTool(scanner.Tool{Name: "rg", Installer: string(scanner.InstCargo), Binaries: []scanner.Binary{{Name: "rg"}}})
	if !res.Detected || !res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v, want true/true", res.Detected, res.HasUpdate)
	}
	if res.Command != "cargo install ripgrep --force" {
		t.Errorf("command = %q, want cargo install ripgrep --force", res.Command)
	}
	if res.Current != "14.1.1" || res.Latest != "14.1.2" {
		t.Errorf("cur=%q latest=%q, want 14.1.1/14.1.2", res.Current, res.Latest)
	}
}

// ---- 版本比较 ----

func TestNewerThan(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"14.1.2", "14.1.1", true},
		{"14.1.1", "14.1.1", false},
		{"0.3.0", "0.2.3", true},
		{"0.2.4", "0.2.3", true},
		{"1.0.0", "1.0.0.1", false},     // 四段式参与数值比较
		{"2.0.0-beta.1", "1.9.0", true}, // 解析失败回退字符串不等
		{"1.0.0-beta.1", "1.0.0", true}, // 字符串前缀更短 → 视为有更新（诚实降级）
	}
	for _, c := range cases {
		if got := newerThan(c.latest, c.current); got != c.want {
			t.Errorf("newerThan(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// ---- 命令表 ----

func TestOfficialCommand(t *testing.T) {
	cases := []struct {
		name      string
		installer scanner.Installer
		cmd       string
		runnable  bool
	}{
		{"ripgrep", scanner.InstBrew, "brew upgrade ripgrep", true},
		{"cli-analyzer", scanner.InstNpm, "npm update -g cli-analyzer", true},
		{"black", scanner.InstPipx, "pipx upgrade black", true},
		{"ripgrep", scanner.InstCargo, "cargo install ripgrep --force", true},
		{"uv", scanner.InstLocalBin, "curl -LsSf https://astral.sh/uv/install.sh | sh", false},
		{"poetry", scanner.InstLocalBin, "curl -sSL https://install.python-poetry.org | python3 -", false},
		{"unknown-tool", scanner.InstLocalBin, "重新运行官方安装脚本（见工具官网）", false},
		{"dlv", scanner.InstGo, "重新执行当时的 go install（需模块路径）", false},
		{"python", scanner.InstPyenv, "无统一官方升级命令（参考工具官网）", false},
		{"whatever", scanner.InstOther, "无统一官方升级命令（参考工具官网）", false},
	}
	for _, c := range cases {
		got := OfficialCommand(c.installer, c.name, "")
		if got.Command != c.cmd || got.Runnable != c.runnable {
			t.Errorf("OfficialCommand(%s, %q) = {%q runnable=%v}, want {%q runnable=%v}",
				c.installer, c.name, got.Command, got.Runnable, c.cmd, c.runnable)
		}
		if got.Runnable {
			if got.Bin == "" || len(got.Args) == 0 {
				t.Errorf("runnable command missing Bin/Args: %+v", got)
			}
		}
	}
	// 代跑参数细节：npm scoped 包名原样传入
	got := OfficialCommand(scanner.InstNpm, "@scope/pkg", "")
	if len(got.Args) != 3 || got.Args[2] != "@scope/pkg" {
		t.Errorf("npm scoped args = %v", got.Args)
	}
	// npmToolID 映射的短工具名 → 真实包名（pi → @earendil-works/pi-coding-agent）
	got = OfficialCommand(scanner.InstNpm, "pi", "")
	if got.Args[2] != "@earendil-works/pi-coding-agent" {
		t.Errorf("npm mapped tool should use real pkg name, args = %v", got.Args)
	}
	// 未知 local-bin 回退通用提示
	got = OfficialCommand(scanner.InstLocalBin, "some-script-tool", "")
	if got.Command == "" {
		t.Error("unknown local-bin should fall back to generic hint")
	}
}

// ---- 代跑执行 ----

func TestRunOfficialFailure(t *testing.T) {
	err := RunOfficial(Command{Bin: "definitely-not-a-real-cmd-xyz", Args: []string{}}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("missing command should error")
	}
}

func TestRunOfficialSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses unix shell")
	}
	var buf bytes.Buffer
	err := RunOfficial(Command{Bin: "sh", Args: []string{"-c", "echo upgrade-ok"}}, &buf, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("RunOfficial: %v", err)
	}
	if buf.String() != "upgrade-ok\n" {
		t.Errorf("output = %q, want upgrade-ok", buf.String())
	}
}
