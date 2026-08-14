package platform

import "strings"

// systemDataDirs 是孤儿过滤用的系统/共享结构目录（数据根顶层目录名，小写）。
// 它们不是任何工具的数据残留：删除是危险或无效的（运行时共享状态、系统
// 数据库、被大量 CLI 共用的基础设施），必须整体跳过。
var systemDataDirs = map[string]bool{
	// node 生态共享目录（update-notifier / configstore 被大量 CLI 工具共用）
	"configstore":            true,
	"simple-update-notifier": true,
	// .NET/Mono 运行时隔离存储与共享状态
	".mono":             true,
	"isolatedstorage":   true,
	".isolated-storage": true,
	// man 数据库（~/.local/share/man 等）
	"man": true,
}

// windowsSystemDataDirs 是 Windows 系统组件目录（%LocalAppData% 顶层）。
// Programs 是 GUI 应用安装根（%LocalAppData%\Programs\<App>），Temp 是系统
// 临时目录，CrashDumps/D3DSCache 是系统崩溃转储与着色器缓存，lxss 是 WSL
// 根文件系统，Comms 是通讯组件。
var windowsSystemDataDirs = map[string]bool{
	"programs":   true,
	"temp":       true,
	"crashdumps": true,
	"d3dscache":  true,
	"lxss":       true,
	"comms":      true,
}

// IsSystemDataDir 报告数据根（root 参数）顶层目录名是否为系统/共享结构
// 目录，孤儿数据过滤时整体跳过。
func IsSystemDataDir(root RootKind, name string) bool {
	n := strings.ToLower(name)
	if systemDataDirs[n] {
		return true
	}
	if root == LocalAppData && windowsSystemDataDirs[n] {
		return true
	}
	return false
}
