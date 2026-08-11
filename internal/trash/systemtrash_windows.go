//go:build windows

package trash

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// 常量来自 shellapi.h
const (
	foDelete     = 0x0003 // FO_DELETE
	fofAllowUndo = 0x0040 // FOF_ALLOWUNDO：保留撤销信息（进入回收站）
	fofNoUI      = 0x1400 // FOF_NO_UI：静默执行，不弹任何对话框
)

// shFileOpStructW 对应 C 结构 SHFILEOPSTRUCTW（64 位布局，共 56 字节）
type shFileOpStructW struct {
	hwnd      uintptr // 0
	wFunc     uint32  // 8
	_         uint32  // padding
	pFrom     *uint16 // 16
	pTo       *uint16 // 24
	fFlags    uint32  // 32
	fAborted  uint32  // 36
	hMappings uintptr // 40
	progress  uintptr // 48
}

var shFileOperationW = syscall.NewLazyDLL("shell32.dll").NewProc("SHFileOperationW")

// systemTrash 将路径移动到 Windows 系统回收站（Recycle Bin）；失败返回错误以便调用方降级
func systemTrash(p string) error {
	// pFrom 必须以双 null 结尾，否则 SHFileOperationW 会越界读取
	src16 := utf16.Encode([]rune(p))
	src16 = append(src16, 0, 0)

	op := shFileOpStructW{
		wFunc:  foDelete,
		fFlags: fofAllowUndo | fofNoUI,
		pFrom:  &src16[0],
	}
	r, _, callErr := shFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	// 调用期间保持 src16 底层数组存活，防止被 GC 回收
	runtime.KeepAlive(src16)
	if r != 0 {
		return fmt.Errorf("SHFileOperationW 失败: 0x%x (%v)", r, callErr)
	}
	if op.fAborted != 0 {
		return errors.New("回收站操作被中断")
	}
	return nil
}
