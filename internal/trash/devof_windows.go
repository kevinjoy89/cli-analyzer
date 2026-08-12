//go:build windows

package trash

import (
	"path/filepath"
	"strings"

	"cli-analyzer/internal/i18n"
)

// devOf 以盘符卷名（如 "C:"）的 FNV-1a 哈希标识文件系统，同卷返回相同值
func devOf(p string) (uint64, error) {
	v := strings.ToUpper(filepath.VolumeName(p))
	if v == "" {
		return 0, i18n.NewError("err.volumeName")
	}
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	var h uint64 = offset64
	for i := 0; i < len(v); i++ {
		h = (h ^ uint64(v[i])) * prime64
	}
	return h, nil
}
