//go:build !windows

package platform

import (
	"os"
	"path/filepath"
)

// LocalBinDir 返回用户本地二进制目录（官方脚本安装的落点，如 astral uv、
// poetry、rye）：优先 $XDG_BIN_HOME，否则 ~/.local/bin（unix 惯例）。
// 该目录在 PATH 上且可能被 augmentUserDirs 补齐（GUI 最小 PATH 场景），
// 其中的二进制是脚本直接放入的 CLI 工具，无包管理器可卸载。
func LocalBinDir() string {
	if x := os.Getenv("XDG_BIN_HOME"); x != "" {
		if abs, err := filepath.Abs(x); err == nil {
			return abs
		}
		return x
	}
	h := HomeDir()
	if h == "" {
		return ""
	}
	return filepath.Join(h, ".local", "bin")
}
