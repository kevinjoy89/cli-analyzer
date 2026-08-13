package platform

import (
	"regexp"
	"strings"
)

// 结构性 GUI 信号：系统约定形态，确定性判定（非启发式）。
// 用于孤儿数据过滤——macOS Containers 下的 bundle-id、Windows UWP 包族名。

var (
	// containerBundleID 匹配 macOS ~/Library/Containers/<bundle-id> 的目录名，
	// 如 com.apple.Safari / com.microsoft.VSCode。
	containerBundleID = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+$`)

	// uwpFamilyName 匹配 Windows %LOCALAPPDATA%\Packages\<family>_<hash> 的
	// 目录名，如 Microsoft.WindowsTerminal_8wekyb3d8bbwe（尾部 8+ 位字母数字）。
	uwpFamilyName = regexp.MustCompile(`^.+_[a-z0-9]{8,}$`)
)

// IsContainerBundleDir 报告目录名是否为 macOS 沙盒容器 bundle-id 形态。
func IsContainerBundleDir(name string) bool {
	return containerBundleID.MatchString(strings.ToLower(name))
}

// IsUWPFamilyDir 报告目录名是否为 Windows UWP 包族形态。
func IsUWPFamilyDir(name string) bool {
	return uwpFamilyName.MatchString(strings.ToLower(name))
}
