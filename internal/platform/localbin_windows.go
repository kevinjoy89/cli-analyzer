//go:build windows

package platform

// LocalBinDir 在 Windows 上无 ~/.local/bin 惯例（安装器落点由 PATH 覆盖），
// 返回空串：classify/uninstall 的 local-bin 分支自然不命中。
func LocalBinDir() string { return "" }
