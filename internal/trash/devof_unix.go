//go:build darwin || linux

package trash

import (
	"errors"
	"os"
	"syscall"
)

// devOf 通过文件系统设备号标识一个路径所在的文件系统
func devOf(p string) (uint64, error) {
	st, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	if sys, ok := st.Sys().(*syscall.Stat_t); ok {
		return uint64(sys.Dev), nil
	}
	return 0, errors.New("无法获取设备号")
}
