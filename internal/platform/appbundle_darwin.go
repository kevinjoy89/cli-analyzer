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
	appNamesOnce.Do(func() {
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
	})
	for _, a := range apps {
		if installedApps[strings.ToLower(strings.TrimSpace(a))] {
			return true
		}
	}
	return false
}
