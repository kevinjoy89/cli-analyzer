//go:build !windows

package platform

// IsInstalledAppDataDir 在非 Windows 平台恒为 false：macOS/Linux 的孤儿
// 数据根（XDG 目录）不由 GUI 应用主导（GUI 数据在 ~/Library 等非孤儿
// 来源），无需安装证据交叉验证。
func IsInstalledAppDataDir(name string) bool { return false }
