//go:build windows

package platform

// augmentUserDirs 在 Windows 上无操作：PATH 已覆盖安装位置，不引入 Unix 目录。
func augmentUserDirs(seen map[string]bool, out []string) []string {
	return out
}
