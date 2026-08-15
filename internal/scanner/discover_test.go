package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cli-analyzer/internal/platform"
)

func TestExcludedByVendor(t *testing.T) {
	// NetSarang 全家桶：任何 exe 名都拦（无例外）
	if !platform.ExcludedByVendor(`C:\Program Files (x86)\NetSarang\Xshell 8`, "xftpcl.exe") {
		t.Error("xftpcl.exe under NetSarang should be excluded")
	}
	if !platform.ExcludedByVendor(`C:\Program Files (x86)\NetSarang\Xshell 8`, "installanchorservice.exe") {
		t.Error("installanchorservice.exe under NetSarang should be excluded")
	}
	// 纯 CLI 例外：amazon 目录下的 aws 保留
	if platform.ExcludedByVendor(`C:\Program Files\Amazon\AWSCLIV2`, "aws.exe") {
		t.Error("aws.exe under Amazon should survive via allowlist")
	}
	// 例外只认精确名称：amazon 目录下非例外的 exe 仍拦
	if !platform.ExcludedByVendor(`C:\Program Files\Amazon\Something`, "ssm.exe") {
		t.Error("non-allowlisted exe under Amazon should be excluded")
	}
	// 片段精确匹配：code 不误伤 opencode
	if platform.ExcludedByVendor(`C:\Users\me\bin\opencode`, "opencode.exe") {
		t.Error("opencode must not match pattern 'code'")
	}
	// 无关目录不受影响
	if platform.ExcludedByVendor(`C:\Program Files\nodejs`, "node.exe") {
		t.Error("nodejs dir should not be excluded")
	}
	// 大小写不敏感
	if !platform.ExcludedByVendor(`c:\program files (x86)\netsarang\Xshell 8`, "xftpcl.exe") {
		t.Error("case-insensitive segment match failed")
	}
	// 多级片段（数据目录形态）：%APPDATA%\NetSarang Computer
	if !platform.ExcludedByVendor(`C:\Users\me\AppData\Roaming\NetSarang Computer`, "NetSarang Computer") {
		t.Error("data dir with vendor segment should be excluded")
	}
}

func TestStructuralSignals(t *testing.T) {
	for _, n := range []string{"com.apple.Safari", "com.microsoft.VSCode", "io.github.some-tool"} {
		if !platform.IsContainerBundleDir(n) {
			t.Errorf("IsContainerBundleDir(%q) = false, want true", n)
		}
	}
	for _, n := range []string{"npm", "go-build", "nodejs", "simple"} {
		if platform.IsContainerBundleDir(n) {
			t.Errorf("IsContainerBundleDir(%q) = true, want false", n)
		}
	}
	for _, n := range []string{"Microsoft.WindowsTerminal_8wekyb3d8bbwe", "Foo.Bar_12345678"} {
		if !platform.IsUWPFamilyDir(n) {
			t.Errorf("IsUWPFamilyDir(%q) = false, want true", n)
		}
	}
	for _, n := range []string{"nodejs", "opencode", "MyApp"} {
		if platform.IsUWPFamilyDir(n) {
			t.Errorf("IsUWPFamilyDir(%q) = true, want false", n)
		}
	}
}

// 确保测试里 strings 被使用（工具链健全性）。

func TestVendorDataOnlyContext(t *testing.T) {
	// "code" 是 DataOnly：exe 发现语境不拦截（真实 PATH 目录安全），
	// 数据目录语境拦截（VS Code 数据目录）。
	if platform.ExcludedByVendor(`C:\code`, "git.exe") {
		t.Error("DataOnly pattern must not affect exe discovery")
	}
	if !platform.ExcludedByVendorData(`/Users/me/Library/Application Support/Code`, "Code") {
		t.Error("DataOnly pattern must apply in data-dir context")
	}
	// 非 DataOnly 模式两语境都生效
	if !platform.ExcludedByVendor(`C:\Program Files (x86)\NetSarang\Xshell 8`, "xftpcl.exe") {
		t.Error("vendor pattern should apply in exe discovery")
	}
}

func TestProbeSafeInstaller(t *testing.T) {
	for _, i := range []Installer{InstBrew, InstNpm, InstPipx, InstCargo, InstGo, InstPyenv, InstVersioned, InstRustup} {
		if !ProbeSafeInstaller(i) {
			t.Errorf("ProbeSafeInstaller(%q) = false, want true", i)
		}
	}
	if ProbeSafeInstaller(InstOther) {
		t.Error("ProbeSafeInstaller(other) must be false — never execute unknown-origin binaries")
	}
}

func TestDiscoverSkipsLibraries(t *testing.T) {
	// 库文件（.dylib/.so）不得作为工具发现；.bak/.old 同样
	for _, n := range []string{"libzmq.5.dylib", "libgit2.dylib", "libfoo.so", "tool.bak", "tool.old"} {
		if isLibraryOrBackup(n) {
			continue
		}
		t.Errorf("expected %q to be skipped", n)
	}
	for _, n := range []string{"git", "node", "cli-analyzer"} {
		if isLibraryOrBackup(n) {
			t.Errorf("expected %q to be kept", n)
		}
	}
}

// TestDiscoverSymlinkToDirResolvesRealBinary 验证 PATH 中指向目录的符号链接
// （sdkman current 形态：symlink -> 目录，目录内含同名可执行文件）扫描后
// Binary.Real 必须是真实二进制文件而非目录——此前 Real=EvalSymlinks(入口)
// = 目录，导致二进制大小恒 0、probe 对目录执行、卸载残留检测误判。
func TestDiscoverSymlinkToDirResolvesRealBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only symlink 场景")
	}
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "mytool"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	pathDir := t.TempDir()
	if err := os.Symlink(binDir, filepath.Join(pathDir, "mytool")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", pathDir)
	execs := discoverExecs()
	var found bool
	for _, ex := range execs {
		if ex.Name == "mytool" {
			found = true
			// 指向目录的链接必须解析到目录内的真实二进制（扫描器直接用
			// execEntry.Real，不再对目录做 EvalSymlinks 得到目录本身；
			// macOS 上 /var 是 /private/var 的符号链接，用 Eval 归一化比较）
			if ex.Real == "" {
				t.Fatal("execEntry.Real 为空：symlink-to-dir 未携带真实二进制路径")
			}
			want, err := filepath.EvalSymlinks(filepath.Join(binDir, "mytool"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := filepath.EvalSymlinks(ex.Real)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("real = %q, want %q（指向目录的链接必须解析到目录内的真实二进制）", got, want)
			}
			break
		}
	}
	if !found {
		t.Fatal("symlink-to-dir 命令未被发现")
	}
}
