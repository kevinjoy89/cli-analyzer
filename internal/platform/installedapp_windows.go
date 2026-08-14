//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

// installedAppNamesOverride 供测试注入固定应用名单（非 nil 时优先于真实加载）。
var installedAppNamesOverride []string

var installedAppNamesOnce sync.Once
var installedAppNamesVal []string

// resetInstalledAppNames 仅测试使用：重置 Once，使下一次读取重新加载名单。
func resetInstalledAppNames() { installedAppNamesOnce = sync.Once{} }

// IsInstalledAppDataDir 报告数据根顶层目录名是否命中已安装应用交叉验证
// （注册表卸载项 DisplayName + 开始菜单快捷方式基名）。仅 Windows 生效；
// 名单惰性加载并缓存（sync.Once），一次扫描只枚举一次（毫秒级）。
func IsInstalledAppDataDir(name string) bool {
	return MatchInstalledAppName(name, installedAppNames())
}

func installedAppNames() []string {
	installedAppNamesOnce.Do(func() {
		if installedAppNamesOverride != nil {
			installedAppNamesVal = installedAppNamesOverride
			return
		}
		installedAppNamesVal = loadInstalledAppNames()
	})
	return installedAppNamesVal
}

// loadInstalledAppNames 收集已安装应用名称证据：
//   - 注册表卸载项 DisplayName：HKCU 与 HKLM 的
//     Software\Microsoft\Windows\CurrentVersion\Uninstall，另含 HKLM 的
//     WOW6432Node（32 位视图）；
//   - 开始菜单快捷方式基名：用户（%APPDATA%）与公共（%ProgramData%）两级，
//     递归枚举 *.lnk 仅取文件名（不解析链接目标，避免 COM/二进制解析）。
func loadInstalledAppNames() []string {
	var names []string
	seen := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			return
		}
		seen[strings.ToLower(s)] = true
		names = append(names, s)
	}

	const uninstallPath = `Software\Microsoft\Windows\CurrentVersion\Uninstall`
	addUninstallDisplayNames(add, registry.CURRENT_USER, uninstallPath)
	addUninstallDisplayNames(add, registry.LOCAL_MACHINE, uninstallPath)
	addUninstallDisplayNames(add, registry.LOCAL_MACHINE, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`)

	for _, dir := range []string{
		filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`),
		filepath.Join(os.Getenv("ProgramData"), `Microsoft\Windows\Start Menu\Programs`),
	} {
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".lnk") {
				add(strings.TrimSuffix(filepath.Base(p), filepath.Ext(p)))
			}
			return nil
		})
	}
	return names
}

// addUninstallDisplayNames 把卸载键 basePath 下所有子键的 DisplayName 加入
// 名单（注册表读取失败静默降级——仅损失该证据源，厂商表兜底）。
func addUninstallDisplayNames(add func(string), root registry.Key, basePath string) {
	k, err := registry.OpenKey(root, basePath, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return
	}
	subs, err := k.ReadSubKeyNames(-1)
	k.Close()
	if err != nil {
		return
	}
	for _, sub := range subs {
		sk, err := registry.OpenKey(root, basePath+`\`+sub, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		displayName, _, err := sk.GetStringValue("DisplayName")
		sk.Close()
		if err == nil {
			add(displayName)
		}
	}
}
