package scanner

import (
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
