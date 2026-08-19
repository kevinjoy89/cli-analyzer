package upgrade

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"testing"

	"cli-analyzer/internal/scanner"
)

// setQuery 注入假查询实现，测试结束自动恢复。
func setQuery(t *testing.T, fn commandFunc) {
	t.Helper()
	old := runQuery
	runQuery = fn
	t.Cleanup(func() { runQuery = old })
}

func TestDetectBrewOutdated(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		if bin != "brew" {
			t.Errorf("bin = %q, want brew", bin)
		}
		return `{"formulae":[{"name":"ripgrep","installed_versions":["14.1.1"],"current_version":"14.1.2"}],"casks":[]}`, nil
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstBrew, "ripgrep")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v, want detected", detected, err)
	}
	if cur != "14.1.1" || latest != "14.1.2" {
		t.Errorf("cur=%q latest=%q, want 14.1.1/14.1.2", cur, latest)
	}
}

// TestDetectBrewMultiKeg 验证多 keg（brew 升序排列）时 current 取最新过时 keg，
// 而非最旧——避免把已装 2.18.1 的用户展示成 2.17.1（fontconfig 实测场景）。
func TestDetectBrewMultiKeg(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return `{"formulae":[{"name":"fontconfig","installed_versions":["2.17.1","2.18.1"],"current_version":"2.18.3"}],"casks":[]}`, nil
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstBrew, "fontconfig")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v, want detected", detected, err)
	}
	if cur != "2.18.1" || latest != "2.18.3" {
		t.Errorf("cur=%q latest=%q, want 2.18.1/2.18.3（最新过时 keg）", cur, latest)
	}
}

func TestDetectBrewUpToDate(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return `{"formulae":[],"casks":[]}`, nil
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstBrew, "ripgrep")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v, want detected", detected, err)
	}
	if cur != "" || latest != "" {
		t.Errorf("up-to-date should carry no versions, got cur=%q latest=%q", cur, latest)
	}
}

// TestDetectBrewOutdatedExit1 验证 brew 检出更新时退出码为 1 但输出有效：
// 只有"命令不存在/无法启动"类错误才降级为检测失败。
func TestDetectBrewOutdatedExit1(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec.ExitError 构造依赖 unix shell")
	}
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		err := exec.Command("sh", "-c", "exit 1").Run() // 真实 *exec.ExitError
		return `{"formulae":[{"name":"ripgrep","installed_versions":["14.1.1"],"current_version":"14.1.2"}]}`, err
	})
	_, _, detected, err := detect(context.Background(), scanner.InstBrew, "ripgrep")
	if err != nil || !detected {
		t.Fatalf("exit-1-with-valid-output should still detect, got detected=%v err=%v", detected, err)
	}
}

func TestDetectBrewCommandMissing(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	})
	if _, _, detected, err := detect(context.Background(), scanner.InstBrew, "ripgrep"); detected || err == nil {
		t.Fatalf("missing command should degrade: detected=%v err=%v", detected, err)
	}
}

func TestDetectNpmOutdated(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		if bin != "npm" {
			t.Errorf("bin = %q, want npm", bin)
		}
		return `{"cli-analyzer":{"current":"1.0.0","wanted":"1.1.0","latest":"1.1.0"}}`, nil
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstNpm, "cli-analyzer")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	if cur != "1.0.0" || latest != "1.1.0" {
		t.Errorf("cur=%q latest=%q", cur, latest)
	}
}

// TestDetectNpmUpToDate 验证 npm 最新态输出 `{}`（实测形状）与空串都判已最新。
func TestDetectNpmUpToDate(t *testing.T) {
	for _, out := range []string{"", "{}"} {
		setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
			return out, nil
		})
		cur, latest, detected, err := detect(context.Background(), scanner.InstNpm, "cli-analyzer")
		if err != nil || !detected {
			t.Fatalf("out=%q: detected=%v err=%v, want detected", out, detected, err)
		}
		if cur != "" || latest != "" {
			t.Errorf("out=%q: up-to-date should carry no versions, got cur=%q latest=%q", out, cur, latest)
		}
	}
}

// TestDetectNpmMappedToolName 验证 npmToolID 映射的短工具名（如 pi）经
// CheckTool 会用真实包名查询（@earendil-works/pi-coding-agent），否则静默
// 误报「已最新」。识别由调用方完成：优先 t.Package（扫描记录的真实包名），
// 旧缓存回退 NpmPackageFor 逆映射。
func TestDetectNpmMappedToolName(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		if len(args) < 4 || args[3] != "@earendil-works/pi-coding-agent" {
			t.Fatalf("npm outdated args = %v, want real pkg name", args)
		}
		return `{"@earendil-works/pi-coding-agent":{"current":"1.0.0","latest":"1.1.0"}}`, nil
	})
	res := CheckTool(scanner.Tool{Name: "pi", Installer: string(scanner.InstNpm), Package: "@earendil-works/pi-coding-agent"})
	if !res.Detected || !res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v", res.Detected, res.HasUpdate)
	}
	if res.Current != "1.0.0" || res.Latest != "1.1.0" {
		t.Errorf("cur=%q latest=%q", res.Current, res.Latest)
	}
	if res.Command != "npm update -g @earendil-works/pi-coding-agent" {
		t.Errorf("command = %q", res.Command)
	}
}

// TestDetectNpmRealShortPkgName 验证与映射短名撞名的真实 npm 包（如真包
// "pi"）不受逆映射误导：t.Package 为确切包名时按原包名查询。这是第二轮
// 审查发现的碰撞回归（code review #3 修复）。
func TestDetectNpmRealShortPkg(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		if len(args) < 4 || args[3] != "pi" {
			t.Fatalf("npm outdated args = %v, want exact short pkg", args)
		}
		return `{"pi":{"current":"2.0.0","latest":"2.0.5"}}`, nil
	})
	res := CheckTool(scanner.Tool{Name: "pi", Installer: string(scanner.InstNpm), Package: "pi"})
	if !res.Detected || !res.HasUpdate {
		t.Fatalf("detected=%v hasUpdate=%v", res.Detected, res.HasUpdate)
	}
	if res.Current != "2.0.0" || res.Latest != "2.0.5" {
		t.Errorf("cur=%q latest=%q", res.Current, res.Latest)
	}
	if res.Command != "npm update -g pi" {
		t.Errorf("command = %q", res.Command)
	}
}

func TestDetectNpmNotInOutput(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return `{"other-pkg":{"current":"1.0.0","latest":"2.0.0"}}`, nil
	})
	if _, _, detected, err := detect(context.Background(), scanner.InstNpm, "cli-analyzer"); detected || err == nil {
		t.Fatalf("pkg missing from output should degrade: detected=%v err=%v", detected, err)
	}
}

func TestDetectPipx(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1:
			return `{"venvs":{"black":{"metadata":{"main_package":{"package":"black","package_version":"23.1.0"}}}}}`, nil
		case 2:
			return "WARNING: pip index is currently an experimental command.\nblack (23.3.0)\nAvailable versions: 23.3.0, 23.2.0, 23.1.0", nil
		}
		return "", errors.New("unexpected extra call")
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstPipx, "black")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	if cur != "23.1.0" || latest != "23.3.0" {
		t.Errorf("cur=%q latest=%q, want 23.1.0/23.3.0", cur, latest)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (pipx list + pip index)", calls)
	}
}

func TestDetectPipxIndexParseFail(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		if calls == 1 {
			return `{"venvs":{"black":{"metadata":{"main_package":{"package":"black","package_version":"23.1.0"}}}}}`, nil
		}
		return "ERROR: No matching distribution found for black", errors.New("exit status 1")
	})
	if _, _, detected, err := detect(context.Background(), scanner.InstPipx, "black"); detected || err == nil {
		t.Fatalf("pip index failure should degrade: detected=%v err=%v", detected, err)
	}
}

// TestDetectPipxSuffixVenv 验证 venv 名带 --suffix（如 `pipx install --suffix
// foo uv` 建 venv `uv-foo`）时，`pip index versions` 用真实发行名（main_package
// .package = uv）而非 venv 名查询（code review #6 修复）。
func TestDetectPipxSuffixVenv(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1:
			return `{"venvs":{"uv-foo":{"metadata":{"main_package":{"package":"uv","package_version":"0.5.0"}}}}}`, nil
		case 2:
			if len(args) == 0 || args[len(args)-1] != "uv" {
				t.Errorf("pip index versions args = %v, want last arg uv（真实发行名）", args)
			}
			return "Available versions: 0.6.0, 0.5.0", nil
		}
		return "", errors.New("unexpected extra call")
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstPipx, "uv-foo")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	if cur != "0.5.0" || latest != "0.6.0" {
		t.Errorf("cur=%q latest=%q, want 0.5.0/0.6.0", cur, latest)
	}
}

func TestDetectCargo(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1:
			return "ripgrep v14.1.1:\n    /usr/local/bin/rg\n", nil
		case 2:
			// 扫描器按二进制名（rg）归类，detectCargo 应映射回 crate 名（ripgrep）
			if len(args) < 2 || args[1] != "ripgrep" {
				t.Errorf("cargo search args = %v, want args[1]=ripgrep（crate 名）", args)
			}
			return "ripgrep = \"14.1.2\"    # A fast search tool\n", nil
		}
		return "", errors.New("unexpected extra call")
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstCargo, "rg")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	if cur != "14.1.1" || latest != "14.1.2" {
		t.Errorf("cur=%q latest=%q, want 14.1.1/14.1.2", cur, latest)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (install --list + search)", calls)
	}
}

// TestDetectCargoGitInstall 验证 git 安装的 crate（条目带修订后缀）只取版本号，
// 不把修订串进 current（code review #6 修复）。
func TestDetectCargoGitInstall(t *testing.T) {
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1:
			return "ripgrep v0.1.0 (git+https://github.com/BurntSushi/ripgrep#abc123):\n    /usr/local/bin/rg\n", nil
		case 2:
			return "ripgrep = \"0.2.0\"    # A fast search tool\n", nil
		}
		return "", errors.New("unexpected extra call")
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstCargo, "rg")
	if err != nil || !detected {
		t.Fatalf("detected=%v err=%v", detected, err)
	}
	if cur != "0.1.0" || latest != "0.2.0" {
		t.Errorf("cur=%q latest=%q, want 0.1.0/0.2.0（不含 git 修订）", cur, latest)
	}
}

func TestDetectCargoNotFound(t *testing.T) {
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		return "other-crate v1.0.0:\n    /usr/bin/other\n", nil
	})
	if _, _, detected, err := detect(context.Background(), scanner.InstCargo, "rg"); detected || err == nil {
		t.Fatalf("crate not installed should degrade: detected=%v err=%v", detected, err)
	}
}

// TestDetectCargoExit1StillDetects 验证 cargo 命令退出码非零但输出有效时
// 仍可检测（与 brew/npm 的 execError 约定一致）：只有「命令缺失/无法启动」
// 类错误才降级为检测失败（code review #8 修复）。
func TestDetectCargoExit1StillDetects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec.ExitError 构造依赖 unix shell")
	}
	calls := 0
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		calls++
		if calls == 1 {
			// install --list 退出码 1 但带有效输出
			err := exec.Command("sh", "-c", "exit 1").Run() // 真实 *exec.ExitError
			return "ripgrep v14.1.1:\n    /usr/local/bin/rg\n", err
		}
		return "ripgrep = \"14.1.2\"    # A fast search tool\n", nil
	})
	cur, latest, detected, err := detect(context.Background(), scanner.InstCargo, "rg")
	if err != nil || !detected {
		t.Fatalf("exit-1-with-valid-output should still detect, got detected=%v err=%v", detected, err)
	}
	if cur != "14.1.1" || latest != "14.1.2" {
		t.Errorf("cur=%q latest=%q, want 14.1.1/14.1.2", cur, latest)
	}
}

// TestDetectNoCapability 验证无检测能力来源不发起任何查询（go/versioned/…）。
func TestDetectNoCapability(t *testing.T) {
	queried := false
	setQuery(t, func(_ context.Context, bin string, args ...string) (string, error) {
		queried = true
		return "", nil
	})
	if Detectable(scanner.InstGo) {
		t.Fatal("go should not be detectable")
	}
	_, _, detected, err := detect(context.Background(), scanner.InstGo, "dlv")
	if detected || err == nil {
		t.Fatalf("go should degrade: detected=%v err=%v", detected, err)
	}
	if queried {
		t.Fatal("no query should be issued for undetectable installer")
	}
}

func TestParsePipIndexVersions(t *testing.T) {
	out := "WARNING: pip index is currently an experimental command. It may be removed/changed in a future release without prior warning.\n\nblack (23.3.0)\nAvailable versions: 23.3.0, 23.2.0, 23.1.0\n"
	v, ok := parsePipIndexVersions(out)
	if !ok || v != "23.3.0" {
		t.Errorf("v=%q ok=%v, want 23.3.0", v, ok)
	}
	if _, ok := parsePipIndexVersions("ERROR: no such package"); ok {
		t.Error("garbage should not parse")
	}
}

func TestParseCargoInstallList(t *testing.T) {
	out := "bat v0.24.0:\n    /usr/local/bin/bat\nripgrep v14.1.1:\n    /usr/local/bin/rg\n"
	// 按二进制名（rg）反查 crate（ripgrep）与版本
	crate, v := parseCargoInstallList(out, "rg")
	if crate != "ripgrep" || v != "14.1.1" {
		t.Errorf("crate=%q v=%q, want ripgrep/14.1.1", crate, v)
	}
	if _, v := parseCargoInstallList(out, "missing"); v != "" {
		t.Errorf("v = %q, want empty", v)
	}
}

func TestParseCargoSearch(t *testing.T) {
	v, ok := parseCargoSearch("ripgrep = \"14.1.2\"    # A fast search tool\n", "ripgrep")
	if !ok || v != "14.1.2" {
		t.Errorf("v=%q ok=%v, want 14.1.2", v, ok)
	}
	if _, ok := parseCargoSearch("other = \"1.0.0\"\n", "ripgrep"); ok {
		t.Error("wrong crate should not parse")
	}
}
