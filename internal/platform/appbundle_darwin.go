//go:build darwin

package platform

import (
	"bytes"
	"encoding/xml"
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

// AppBundleVersion 读 .app 包 Info.plist 里的版本号（CFBundleShortVersionString，
// 缺失时回退 CFBundleVersion）。桌面端工具往往没有 PATH 二进制，--version 探测
// 无从下手，但应用包自己就带着版本：这是「装着却显示 —」的根因。
// 返回空表示读不到（非 macOS、包不存在、plist 无版本键）。
func AppBundleVersion(appPath string) string {
	plist := filepath.Join(appPath, "Contents", "Info.plist")
	b, err := os.ReadFile(plist)
	if err != nil {
		return ""
	}
	// plist 是 <key>X</key><string>Y</string> 交替的平铺结构，不是按 key 命名
	// 的元素，encoding/xml 的 struct tag 匹配不上——必须按事件流扫描。
	// 用标准库而非 PlistBuddy：无外部依赖，且可在测试里构造输入。
	vals := map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(b))
	var pendingKey string
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				var s string
				if err := dec.DecodeElement(&s, &t); err == nil {
					pendingKey = strings.TrimSpace(s)
				}
			case "string":
				var s string
				if err := dec.DecodeElement(&s, &t); err == nil && pendingKey != "" {
					vals[pendingKey] = strings.TrimSpace(s)
					pendingKey = ""
				}
			}
		}
	}
	if v := vals["CFBundleShortVersionString"]; v != "" {
		return v
	}
	return vals["CFBundleVersion"]
}
