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
	{Pattern: "orbstack"},
	{Pattern: "uuremote"}, // 远程控制客户端
	{Pattern: "warp"},     // Warp 终端（GUI；其 CLI 为伴侣）
	{Pattern: "apple"},
	{"code", nil, true}, // VS Code 数据目录；仅数据上下文（避免拦掉真实 PATH 目录）
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
func pathSegments(path string) []string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "\\", "/")
	segs := strings.Split(path, "/")
	for i := range segs {
		segs[i] = strings.ToLower(segs[i])
	}
	return segs
}
