//go:build !darwin

package platform

// AppsInstalled 在非 macOS 平台恒为 false：v1 无跨平台的 GUI 应用安装证据
// 来源（Windows 需注册表卸载项/开始菜单枚举）。AI 条目的「已卸载」判定
// 在这些平台退化为二进制缺失——桌面端仍在但 CLI 未上 PATH 的场景会被
// 误标，属可接受的 v1 局限（见 add-ai-tool-catalog design D3）。
func AppsInstalled(apps []string) bool { return false }
