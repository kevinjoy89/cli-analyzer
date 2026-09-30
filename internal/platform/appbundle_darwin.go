//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// appDirs 是 macOS 应用安装目录（按优先级）。
func appDirs() []string {
	dirs := []string{"/Applications", "/System/Applications"}
	if h, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(h, "Applications"))
	}
	return dirs
}

var (
	appNamesOnce  sync.Once
	installedApps map[string]bool // 小写 .app 基名集合（"trae cn"、"zcode"…）
)

// AppsInstalled 报告任一给定应用名是否已安装（<name>.app 存在于任一应用
// 目录）。AI 编目条目的「已卸载」判定使用：无二进制且桌面端缺失 = 已卸载
// 残留。结果按进程缓存（应用安装状态在扫描周期内不变）。
func AppsInstalled(apps []string) bool {
	appNamesOnce.Do(loadInstalledApps)
	for _, a := range apps {
		if installedApps[strings.ToLower(strings.TrimSpace(a))] {
			return true
		}
	}
	return false
}

func loadInstalledApps() {
	installedApps = map[string]bool{}
	for _, dir := range appDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".app") {
				continue
			}
			installedApps[strings.ToLower(strings.TrimSuffix(name, ".app"))] = true
		}
	}
}

// AppBundlePaths 返回给定应用名中实际存在的 .app 包路径（按 appDirs 顺序
// 取首个命中，每个应用至多一个路径）。AI 编目条目的桌面端应用包测量
// 使用：应用包作为 Kind install 归因目录计入安装占用。应用不存在时不
// 返回该应用的路径（与「已卸载」口径一致）。
func AppBundlePaths(apps []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, a := range apps {
		name := strings.TrimSpace(a)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		for _, dir := range appDirs() {
			p := filepath.Join(dir, name+".app")
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				out = append(out, p)
				break
			}
		}
	}
	return out
}
