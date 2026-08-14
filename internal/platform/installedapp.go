package platform

import "strings"

// minAppNamePrefixLen 是已安装应用交叉验证前缀匹配的最小长度门槛：短名
// （pc/git/go/code 等）不参与前缀匹配，避免把真实 CLI 数据目录误判为
// 已安装 GUI 应用的数据目录。
const minAppNamePrefixLen = 5

// MatchInstalledAppName 报告数据根顶层目录名 name 是否与任一已安装应用
// 名称 apps 匹配（大小写不敏感、去空白）。匹配基于安装证据（应用显示名 /
// 快捷方式名）而非名称形态启发式，规则：
//
//  1. 精确相等（任意长度）；
//  2. 目录名是应用名前缀且目录名长度 ≥ 5——应用显示名常为
//     "目录名 + 版本/描述"（如 PeaZip ⊂ "PeaZip 11.0.0 (WIN64)"、
//     IObit ⊂ "IObit Uninstaller"）；
//  3. 应用名是目录名前缀、应用名长度 ≥ 5 且目录名余段以 "-" 开头——
//     "<App>-updater" 类目录（如 "termius-updater" ⊃ Termius、
//     "腾讯电脑管家-全局信息" ⊃ "腾讯电脑管家"）。
//
// 规则 3 的 "-" 后缀约束防止普通复合名误伤（"gitea" ⊃ "git" 不命中，
// "notepadnext" ⊃ "notepad" 不命中）。
func MatchInstalledAppName(name string, apps []string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	for _, a := range apps {
		al := strings.ToLower(strings.TrimSpace(a))
		if al == "" {
			continue
		}
		if al == n {
			return true
		}
		if len(n) >= minAppNamePrefixLen && strings.HasPrefix(al, n) {
			return true
		}
		if len(al) >= minAppNamePrefixLen && strings.HasPrefix(n, al) && strings.HasPrefix(n[len(al):], "-") {
			return true
		}
	}
	return false
}
