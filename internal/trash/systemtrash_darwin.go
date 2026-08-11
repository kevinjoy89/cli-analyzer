//go:build darwin

package trash

import (
	"os"
	"path/filepath"
)

// systemTrash 将路径移动到 macOS 系统回收站（~/.Trash）；失败返回错误以便调用方降级
func systemTrash(p string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	trashDir := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return err
	}
	return os.Rename(p, uniquify(filepath.Join(trashDir, filepath.Base(p))))
}
