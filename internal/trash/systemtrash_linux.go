//go:build linux

package trash

import (
	"fmt"
	"os/exec"
)

// systemTrash 将路径移动到 Linux 系统回收站（gio trash）；工具缺失时返回错误以便调用方降级
func systemTrash(p string) error {
	if _, err := exec.LookPath("gio"); err != nil {
		return errNoSystemTrash
	}
	if out, err := exec.Command("gio", "trash", p).CombinedOutput(); err != nil {
		return fmt.Errorf("gio trash 失败: %v: %s", err, out)
	}
	return nil
}
