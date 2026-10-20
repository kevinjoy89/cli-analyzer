//go:build !darwin

package platform

// AppsInstalled 在非 macOS 平台恒为 false：v1 无跨平台的 GUI 应用安装证据
// 来源（Windows 需注册表卸载项/开始菜单枚举）。AI 条目的「已卸载」判定
// 在这些平台退化为二进制缺失——桌面端仍在但 CLI 未上 PATH 的场景会被
// 误标，属可接受的 v1 局限（见 add-ai-tool-catalog design D3）。
func AppsInstalled(apps []string) bool { return false }

// AppBundlePaths 在非 macOS 平台返回空：桌面端应用包测量仅实现 macOS
// （Windows 的 Program Files / 注册表口径留待后续）。
func AppBundlePaths(apps []string) []string { return nil }

// AppBundleVersion 在非 macOS 平台返回空：.app 包及其 Info.plist 是 macOS
// 专有格式（Windows 的版本在 PE 资源/注册表里，口径不同）。
func AppBundleVersion(appPath string) string { return "" }
