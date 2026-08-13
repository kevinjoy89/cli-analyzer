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
type VendorExclusion struct {
	Pattern string
	Allow   []string
}

// vendorExclusions 是预置的非 CLI 排除表。
// 判据：模式命中 GUI 产品安装目录或数据目录的路径片段；例外仅限独立安装、
// 拥有自己目录、不服务于任何 GUI 应用的纯 CLI 产品（如 AWS CLI、Azure CLI、
// Google Cloud SDK）。NetSarang 无例外（Xshell/Xftp 全家桶拦掉）。
var vendorExclusions = []VendorExclusion{
	{"microsoft", []string{"az"}},
	{"amazon", []string{"aws"}},
	{"google", []string{"gcloud", "gsutil", "bq"}},
	{"netsarang", nil},
	{"netsarang computer", nil}, // 数据目录形态：%APPDATA%\NetSarang Computer
	{"adobe", nil},
	{"tencent", nil},
	{"wechat", nil},
	{"qq", nil},
	{"discord", nil},
	{"slack", nil},
	{"steam", nil},
	{"obsidian", nil},
	{"notion", nil},
	{"telegram", nil},
	{"zoom", nil},
	{"teams", nil},
	{"jetbrains", nil},
}

// MatchVendorExclusion 报告路径的任一目录片段（小写）是否命中排除表厂商模式。
// 命中返回该厂商规则；未命中返回 nil。
func MatchVendorExclusion(path string) *VendorExclusion {
	for _, seg := range pathSegments(path) {
		for i := range vendorExclusions {
			if seg == vendorExclusions[i].Pattern {
				return &vendorExclusions[i]
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

// ExcludedByVendor 报告 (path, name) 是否被非 CLI 排除表拦下：
// 路径任一片段命中厂商模式、且名称不在该厂商纯 CLI 例外中。
func ExcludedByVendor(path, name string) bool {
	if v := MatchVendorExclusion(path); v != nil {
		return !v.IsAllowedCLI(name)
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
