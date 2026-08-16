// Package platform 的非 CLI（GUI 应用）排除体系。
//
// 范围原则：本应用只管理 CLI 工具及其残留。GUI 应用、其命令行伴侣、其数据
// 目录一律排除。排除表统一应用于可执行文件发现与数据目录归因两个环节。
package platform

import (
	"path/filepath"
	"strings"
)

// VendorExclusion 描述一个已知 GUI 产品厂商的排除规则：
// Pattern 是路径片段（小写精确匹配）；Allow 是例外——该厂商独立安装的纯 CLI
// 产品名（剥扩展名后小写精确匹配）。GUI 应用自带的命令行伴侣一律不例外。
// DataOnly 表示该模式仅用于数据目录归因（孤儿过滤），不用于 exe 发现——
// 短名（如 code）可能是用户的真实 PATH 目录，拦截会误伤其中的工具。
type VendorExclusion struct {
	Pattern  string
	Allow    []string
	DataOnly bool
}

// vendorExclusions 是预置的非 CLI 排除表。
// 判据：模式命中 GUI 产品安装目录或数据目录的路径片段；例外仅限独立安装、
// 拥有自己目录、不服务于任何 GUI 应用的纯 CLI 产品（如 AWS CLI、Azure CLI、
// Google Cloud SDK）。NetSarang 无例外（Xshell/Xftp 全家桶拦掉）。
var vendorExclusions = []VendorExclusion{
	// 厂商目录片段（exe 安装目录常以厂商名分层）
	{Pattern: "microsoft", Allow: []string{"az"}},
	{Pattern: "amazon", Allow: []string{"aws"}},
	{Pattern: "google", Allow: []string{"gcloud", "gsutil", "bq"}},
	{Pattern: "netsarang"},
	{Pattern: "netsarang computer"}, // 数据目录形态：%APPDATA%\NetSarang Computer
	{Pattern: "adobe"},
	{Pattern: "tencent"},
	{Pattern: "wechat"},
	{Pattern: "qq"},
	{Pattern: "discord"},
	{Pattern: "slack"},
	{Pattern: "steam"},
	{Pattern: "obsidian"},
	{Pattern: "notion"},
	{Pattern: "telegram"},
	{Pattern: "zoom"},
	{Pattern: "teams"},
	{Pattern: "jetbrains"},
	// 常见 GUI 产品的数据目录名（精确片段；产品级，非厂商级）
	{Pattern: "microsoft edge"},
	{Pattern: "microsoft teams"},
	{Pattern: "onedrive"},
	{Pattern: "google chrome"},
	{Pattern: "google drive"},
	{Pattern: "claude-3p"}, // Anthropic 桌面版（与 CLI claude 区分）
	{Pattern: "qianwen"},   // 通义千问桌面版
	{Pattern: "微信开发者工具"},   // 微信开发者工具（GUI IDE）
	{Pattern: "cursor"},    // Cursor（GUI 编辑器）
	{Pattern: "figma"},
	{Pattern: "xmind"},
	{Pattern: "trae cn"},
	{Pattern: "qclaw"},
	{Pattern: "sase"}, // 企业安全代理（DLP 类，内部件会触达通讯录等 TCC 资源）
	{Pattern: "parallels desktop"},
	// OrbStack（GUI Docker 桌面）：内部工具（docker-tools 等）排除；docker/
	// kubectl/docker-compose 是标准 CLI 产品（/usr/local/bin 与 ~/.orbstack/bin
	// 的符号链接都指向其 xbin），保留
	{Pattern: "orbstack", Allow: []string{"docker", "docker-compose", "kubectl"}},
	{Pattern: "uuremote"}, // 远程控制客户端
	{Pattern: "warp"},     // Warp 终端（GUI；其 CLI 为伴侣）
	{Pattern: "apple"},
	// VS Code：安装目录（bin 下 code.cmd/code-tunnel.exe 是其 GUI 命令行伴侣）
	// 双向拦。DataOnly 的 "code" 只覆盖 %APPDATA%\Code 数据目录。
	{Pattern: "microsoft vs code"},
	{Pattern: "vs code"}, // 便携/自定义布局（如 D:\tools\VS Code\bin）
	{"code", nil, true},  // VS Code 数据目录；仅数据上下文（避免拦掉真实 PATH 目录）

	// ---- 高频 GUI 产品扩充（产品级 DataOnly：只影响孤儿过滤，不拦 exe 发现）----
	// 浏览器/邮件
	{Pattern: "mozilla", DataOnly: true}, // Firefox/Thunderbird 数据根（%APPDATA%\Mozilla）
	{Pattern: "firefox", DataOnly: true},
	{Pattern: "bravesoftware", DataOnly: true},
	{Pattern: "vivaldi", DataOnly: true},
	{Pattern: "opera", DataOnly: true},
	{Pattern: "opera software", DataOnly: true},
	{Pattern: "thunderbird", DataOnly: true},
	{Pattern: "foxmail", DataOnly: true},
	// 聊天/会议/办公
	{Pattern: "dingtalk", DataOnly: true},
	{Pattern: "feishu", DataOnly: true},
	{Pattern: "lark", DataOnly: true},
	{Pattern: "wecom", DataOnly: true}, // 企业微信
	{Pattern: "signal", DataOnly: true},
	{Pattern: "whatsapp", DataOnly: true},
	{Pattern: "webex", DataOnly: true},
	{Pattern: "wemeet", DataOnly: true}, // 腾讯会议
	{Pattern: "wps", DataOnly: true},
	{Pattern: "kingsoft", DataOnly: true},
	{Pattern: "onlyoffice", DataOnly: true},
	// 网盘/下载
	{Pattern: "baidunetdisk", DataOnly: true},
	{Pattern: "aliyundrive", DataOnly: true},
	{Pattern: "quark", DataOnly: true},
	{Pattern: "xunlei", DataOnly: true},
	{Pattern: "dropbox", DataOnly: true},
	{Pattern: "megasync", DataOnly: true},
	// 音乐/视频/直播
	{Pattern: "spotify", DataOnly: true},
	{Pattern: "kugou", DataOnly: true},
	{Pattern: "kuwo", DataOnly: true},
	{Pattern: "netease", DataOnly: true},
	{Pattern: "bilibili", DataOnly: true},
	{Pattern: "douyin", DataOnly: true},
	{Pattern: "kuaishou", DataOnly: true},
	{Pattern: "iqiyi", DataOnly: true},
	{Pattern: "youku", DataOnly: true},
	{Pattern: "potplayer", DataOnly: true},
	{Pattern: "vlc", DataOnly: true},
	{Pattern: "kodi", DataOnly: true},
	{Pattern: "obs studio", DataOnly: true},
	{Pattern: "obs-studio", DataOnly: true},
	{Pattern: "streamlabs", DataOnly: true},
	// 编辑器/桌面工具（含 macOS XDG 实测漏网）
	{Pattern: "iterm2", DataOnly: true},
	{Pattern: "raycast", DataOnly: true},
	{Pattern: "alfred", DataOnly: true},
	{Pattern: "karabiner", DataOnly: true},
	{Pattern: "joplin", DataOnly: true},
	{Pattern: "joplin-desktop", DataOnly: true},
	{Pattern: "axure", DataOnly: true},
	{Pattern: "geany", DataOnly: true},
	{Pattern: "notepad", DataOnly: true},
	{Pattern: "jgit", DataOnly: true}, // Eclipse EGit（GUI 生态）
	// 游戏平台
	{Pattern: "epic games", DataOnly: true},
	{Pattern: "battlenet", DataOnly: true},
	{Pattern: "riot", DataOnly: true},
	{Pattern: "ubisoft", DataOnly: true},
	{Pattern: "gog", DataOnly: true},
	{Pattern: "roblox", DataOnly: true},
	{Pattern: "minecraft", DataOnly: true},
	{Pattern: "unity", DataOnly: true},
	{Pattern: "blender", DataOnly: true},
	// VPN/代理 GUI 客户端（clash/v2ray/mihomo 等核心 CLI 本体不拦）
	{Pattern: "nordvpn", DataOnly: true},
	{Pattern: "expressvpn", DataOnly: true},
	{Pattern: "surfshark", DataOnly: true},
	{Pattern: "protonvpn", DataOnly: true},
	{Pattern: "clash-verge", DataOnly: true},
	{Pattern: "clash verge", DataOnly: true},
	{Pattern: "v2rayn", DataOnly: true},
	{Pattern: "v2ray-n", DataOnly: true},
	// 密码管理
	{Pattern: "1password", DataOnly: true},
	{Pattern: "bitwarden", DataOnly: true},
	{Pattern: "lastpass", DataOnly: true},
	// 输入法/安全
	{Pattern: "sogou", DataOnly: true},
	{Pattern: "iflytek", DataOnly: true},
	{Pattern: "huorong", DataOnly: true},
	{"360", nil, true},
	// 笔记/阅读
	{Pattern: "calibre", DataOnly: true},
	{Pattern: "anki", DataOnly: true},
	{Pattern: "evernote", DataOnly: true},

	// VS Code 用户数据（配置/扩展/远程服务端）；仅数据语境
	{Pattern: ".vscode", DataOnly: true},
	{Pattern: ".vscode-server", DataOnly: true},

	// ---- Windows 实测漏网 GUI 产品（2026-08 Windows 实机孤儿扫描补充）----
	// 远程/远控
	{Pattern: "awesun", DataOnly: true},   // 向日葵远程（AweSun）
	{Pattern: "sunlogin", DataOnly: true}, // 向日葵远程（经典名）
	{Pattern: "oray", DataOnly: true},     // 贝锐 Oray（向日葵厂商）
	// 系统/工具
	{Pattern: "iobit", DataOnly: true},
	{Pattern: "rufus", DataOnly: true}, // GUI USB 工具
	{Pattern: "hd tune pro", DataOnly: true},
	{Pattern: "hdtunepro", DataOnly: true},
	{Pattern: "qarmin", DataOnly: true}, // Qarmin（GUI 文件管理器）
	{Pattern: "neatdm", DataOnly: true}, // Neat Download Manager
	{Pattern: "neat download manager", DataOnly: true},
	{Pattern: "utforpc", DataOnly: true}, // uTorrent for PC
	{Pattern: "utorrent", DataOnly: true},
	// 编辑器/媒体/传输
	{Pattern: "notepadnext", DataOnly: true}, // Notepad Next（GUI 编辑器）
	{Pattern: "notepad next", DataOnly: true},
	{Pattern: "potplayermini64", DataOnly: true}, // PotPlayer
	{Pattern: "daum", DataOnly: true},            // PotPlayer 厂商
	{Pattern: "xnviewmp", DataOnly: true},        // XnView MP
	{Pattern: "xnview", DataOnly: true},
	{Pattern: "pixpin", DataOnly: true}, // PixPin（GUI 截图工具）
	{Pattern: "localsend", DataOnly: true},
	{Pattern: "peazip", DataOnly: true},
	{Pattern: "tabby", DataOnly: true},   // Tabby 终端（GUI；其 CLI 为伴侣）
	{Pattern: "termius", DataOnly: true}, // Termius（GUI SSH 客户端）
	// 厂商/平台目录
	{Pattern: "qqex", DataOnly: true},                    // 腾讯 QQ 组件
	{Pattern: "gameviewer", DataOnly: true},              // 网易 GameViewer 远程
	{Pattern: "the quark authors", DataOnly: true},       // 夸克浏览器/网盘
	{Pattern: "flutter_webview_windows", DataOnly: true}, // Flutter 桌面 WebView 运行时
	{Pattern: "atlassian", DataOnly: true},               // Atlassian（Sourcetree/Jira 桌面端）

	// ---- 驱动/硬件厂商（双向拦：目录下为系统组件，不应作为工具扫描或探测）----
	{Pattern: "nvidia"},
	{Pattern: "nvidia corporation"}, // %ProgramFiles%\NVIDIA Corporation\（含空格目录名）
	{Pattern: "intel"},
	{Pattern: "realtek"},
	{Pattern: "logitech"},
	{Pattern: "razer"},
	{Pattern: "samsung"},
	{Pattern: "huawei"},
	{Pattern: "xiaomi"},
	{Pattern: "lenovo"},
	{Pattern: "canon"},
	{Pattern: "epson"},
	{Pattern: "brother"},
	{Pattern: "synaptics"},
	{Pattern: "elgato"},
	{Pattern: "corsair"},
	{Pattern: "steelseries"},
	// 短厂商名（易与用户目录/真实 PATH 目录冲突 → 仅数据语境）
	{Pattern: "amd", DataOnly: true},
	{Pattern: "hp", DataOnly: true},
	{Pattern: "dell", DataOnly: true},
	{Pattern: "asus", DataOnly: true},
	{Pattern: "acer", DataOnly: true},
	{Pattern: "msi", DataOnly: true},
	{Pattern: "baidu", DataOnly: true},
	// 微信开发者工具数据根等中文产品名
	{Pattern: "网易", DataOnly: true},
}

// ExcludedByVendor 报告 (path, name) 是否被非 CLI 排除表拦下（exe 发现语境，
// 跳过 DataOnly 模式）：路径任一片段命中厂商模式、且名称不在例外中。
func ExcludedByVendor(path, name string) bool {
	if v := matchVendorExclusion(path, false); v != nil {
		return !v.IsAllowedCLI(name)
	}
	return false
}

// ExcludedByVendorData 是数据目录归因语境（孤儿过滤）：包含 DataOnly 模式。
func ExcludedByVendorData(path, name string) bool {
	if v := matchVendorExclusion(path, true); v != nil {
		return !v.IsAllowedCLI(name)
	}
	return false
}

// matchVendorExclusion 报告路径的任一目录片段（小写）是否命中排除表厂商模式。
func matchVendorExclusion(path string, includeDataOnly bool) *VendorExclusion {
	for _, seg := range pathSegments(path) {
		for i := range vendorExclusions {
			v := &vendorExclusions[i]
			if seg == v.Pattern && (includeDataOnly || !v.DataOnly) {
				return v
			}
		}
	}
	return nil
}

// IsAllowedCLI 报告裸名称（剥扩展名后小写）是否属于该厂商规则的纯 CLI 例外。
func (v *VendorExclusion) IsAllowedCLI(name string) bool {
	n := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	for _, a := range v.Allow {
		if n == a {
			return true
		}
	}
	return false
}

// pathSegments 把路径拆成目录片段（兼容 / 与 \ 分隔符），小写化。
// macOS 的 .app 包目录片段带扩展名（"Trae CN.app"），与排除表的
// 产品名模式（"trae cn"）精确匹配不上——剥掉 ".app" 后缀后再匹配。
func pathSegments(path string) []string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "\\", "/")
	segs := strings.Split(path, "/")
	for i := range segs {
		segs[i] = strings.ToLower(strings.TrimSuffix(segs[i], ".app"))
	}
	return segs
}
