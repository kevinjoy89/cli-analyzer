package platform

import (
	"path/filepath"
	"testing"
)

// TestVendorExclusionsDataOnly 验证 DataOnly 模式仅拦截数据目录归因
// （孤儿过滤），不拦截 exe 发现——短名/产品名可能是真实 PATH 目录。
func TestVendorExclusionsDataOnly(t *testing.T) {
	dataOnlyEntries := []string{"iterm2", "raycast", "mozilla", "code", "amd", "360", "tabby", "rufus", "iobit", "awesun"}
	for _, pat := range dataOnlyEntries {
		p := filepath.Join("/Users/wei/.config", pat)
		if !ExcludedByVendorData(p, "whatever") {
			t.Errorf("ExcludedByVendorData(%q): want hit for DataOnly %q", p, pat)
		}
		if ExcludedByVendor(p, "whatever") {
			t.Errorf("ExcludedByVendor(%q): DataOnly %q must not block exe discovery", p, pat)
		}
	}
}

// TestVendorExclusionsExpanded 验证扩充后的高频 GUI 产品/厂商条目：
// 数据语境全部命中；双向厂商（nvidia 等）在 exe 语境同样拦截。
func TestVendorExclusionsExpanded(t *testing.T) {
	dataOnlyCases := []struct{ path, name string }{
		{"/Users/wei/.config/iterm2", "iterm2"},                 // iTerm2（macOS 实测漏网）
		{"/Users/wei/.config/raycast", "raycast"},               // Raycast
		{"/Users/wei/.config/joplin-desktop", "joplin-desktop"}, // Joplin
		{"/Users/wei/.local/share/karabiner", "karabiner"},      // Karabiner-Elements
		{"/Users/wei/.config/geany", "geany"},                   // Geany
		{"/Users/wei/.local/share/Axure", "Axure"},              // Axure（大小写不敏感）
		{"/Users/wei/.config/notepad", "notepad"},               // Notepad
		{`C:\Users\wei\AppData\Roaming\Mozilla\Firefox`, "Mozilla"},
		{`C:\Users\wei\AppData\Roaming\dingtalk`, "dingtalk"},
		{`C:\Users\wei\AppData\Roaming\feishu`, "feishu"},
		{`C:\Users\wei\AppData\Roaming\Tencent\WeCom`, "WeCom"}, // 企业微信（tencent 已拦）
		{`C:\Users\wei\AppData\Roaming\BraveSoftware`, "BraveSoftware"},
		{`C:\Users\wei\AppData\Roaming\obs-studio`, "obs-studio"},
		{`C:\Users\wei\AppData\Local\1Password`, "1Password"},
		{`C:\Users\wei\AppData\Roaming\360`, "360"},
		{`C:\Users\wei\AppData\Roaming\BaiduNetdisk`, "BaiduNetdisk"},
		{`C:\Users\wei\AppData\Roaming\Spotify`, "Spotify"},
		{`C:\Users\wei\AppData\Roaming\Epic Games`, "Epic Games"},
		{`C:\Users\wei\AppData\Roaming\clash-verge`, "clash-verge"},
		{`C:\Users\wei\AppData\Roaming\v2rayN`, "v2rayN"},
		// 2026-08 Windows 实机孤儿扫描补充条目
		{`C:\Users\wei\AppData\Roaming\AweSun`, "AweSun"},
		{`C:\Users\wei\AppData\Roaming\Oray`, "Oray"},
		{`C:\Users\wei\AppData\Roaming\IObit`, "IObit"},
		{`C:\Users\wei\AppData\Roaming\NeatDM`, "NeatDM"},
		{`C:\Users\wei\AppData\Roaming\NotepadNext`, "NotepadNext"},
		{`C:\Users\wei\AppData\Roaming\PotPlayerMini64`, "PotPlayerMini64"},
		{`C:\Users\wei\AppData\Roaming\QQEX`, "QQEX"},
		{`C:\Users\wei\AppData\Roaming\Qarmin`, "Qarmin"},
		{`C:\Users\wei\AppData\Roaming\The Quark Authors`, "The Quark Authors"},
		{`C:\Users\wei\AppData\Roaming\XnViewMP`, "XnViewMP"},
		{`C:\Users\wei\AppData\Roaming\utForpc`, "utForpc"},
		{`C:\Users\wei\AppData\Roaming\tabby`, "tabby"},
		{`C:\Users\wei\AppData\Roaming\Termius`, "Termius"},
		{`C:\Users\wei\AppData\Roaming\HD Tune Pro`, "HD Tune Pro"},
		{`C:\Users\wei\AppData\Local\GameViewer`, "GameViewer"},
		{`C:\Users\wei\AppData\Local\PixPin`, "PixPin"},
		{`C:\Users\wei\AppData\Local\Rufus`, "Rufus"},
		{`C:\Users\wei\AppData\Local\flutter_webview_windows`, "flutter_webview_windows"},
		{`C:\Users\wei\AppData\Roaming\Atlassian`, "Atlassian"},
		{`C:\Users\wei\AppData\Local\Atlassian`, "Atlassian"},
	}
	for _, c := range dataOnlyCases {
		if !ExcludedByVendorData(c.path, c.name) {
			t.Errorf("ExcludedByVendorData(%q): want hit", c.path)
		}
	}
	// 双向厂商：exe 发现语境同样拦截（驱动/硬件目录不应作为工具扫描）
	for _, c := range []struct{ path, name string }{
		{`C:\Program Files\NVIDIA Corporation\NVSMI`, "nvidia-smi"},
		{`C:\Program Files\NVIDIA\NVSMI`, "nvidia-smi"},
		{`C:\Program Files\Intel\DriverStore`, "x"},
		{`C:\Program Files\Realtek`, "x"},
		{`C:\Program Files\Logitech`, "x"},
		{`C:\Program Files\Huawei`, "x"},
		{`C:\Program Files\Samsung`, "x"},
	} {
		if !ExcludedByVendor(c.path, c.name) {
			t.Errorf("ExcludedByVendor(%q): want hit (bidirectional vendor)", c.path)
		}
	}
}

// TestVendorExclusionsDoNotBlockCLICores 验证代理/网络核心 CLI（clash、
// v2ray、mihomo、openvpn 等）不在排除表内——它们是有独立二进制的 CLI
// 工具，只有 GUI 客户端（clash-verge/v2rayN 等）被拦。
func TestVendorExclusionsDoNotBlockCLICores(t *testing.T) {
	for _, c := range []struct{ path, name string }{
		{"/Users/wei/.config/mihomo", "mihomo"},
		{"/Users/wei/.config/clash", "clash"},
		{"/Users/wei/.config/v2ray", "v2ray"},
	} {
		if ExcludedByVendorData(c.path, c.name) {
			t.Errorf("ExcludedByVendorData(%q): CLI core must stay eligible as orphan", c.path)
		}
	}
}
