//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"strings"
)

// guiImportDLLs 是 GUI 程序的强信号导入库：纯控制台工具不会直接链接它们
// （shell32/ole32 等会被命令行工具用于路径/COM 操作，故不在此列）。
var guiImportDLLs = map[string]bool{
	"user32.dll":   true,
	"gdi32.dll":    true,
	"comctl32.dll": true,
	"uxtheme.dll":  true,
	"dwmapi.dll":   true,
	"d2d1.dll":     true,
}

// importsGUI walks the PE import table and reports whether the executable
// directly imports a GUI-only DLL. This catches GUI apps whose Subsystem flag
// lies (built as CUI so no console window flashes — e.g. NetSarang Xshell),
// which the subsystem check alone would let through.
func importsGUI(f *os.File, peOff, optOff int64, optSize int, numSec uint16) bool {
	// Import table = DataDirectory[1]. PE32: data dirs at opt+96; PE32+: +112.
	ddOff := optOff + 96
	if optSize == 240 { // PE32+
		ddOff = optOff + 112
	}
	var dd [8]byte
	if _, err := f.ReadAt(dd[:], ddOff+8); err != nil { // [1] → +8
		return false
	}
	impRVA := int64(binary.LittleEndian.Uint32(dd[0:]))
	impSize := int64(binary.LittleEndian.Uint32(dd[4:]))
	if impRVA == 0 {
		return false
	}

	// Section headers start right after the optional header.
	secOff := optOff + int64(optSize)
	var secs []struct {
		va, vsize, foff, fsize uint32
	}
	for i := 0; i < int(numSec); i++ {
		var sh [40]byte
		if _, err := f.ReadAt(sh[:], secOff+int64(i)*40); err != nil {
			break
		}
		secs = append(secs, struct {
			va, vsize, foff, fsize uint32
		}{
			va:    binary.LittleEndian.Uint32(sh[12:]),
			vsize: binary.LittleEndian.Uint32(sh[8:]),
			foff:  binary.LittleEndian.Uint32(sh[20:]),
			fsize: binary.LittleEndian.Uint32(sh[16:]),
		})
	}

	rvaToOff := func(rva int64) int64 {
		for _, s := range secs {
			va, vs := int64(s.va), int64(s.vsize)
			if vs == 0 {
				vs = int64(s.fsize)
			}
			if rva >= va && rva < va+vs {
				return int64(s.foff) + (rva - va)
			}
		}
		return -1
	}

	// Walk import descriptors (20 bytes each) until an all-zero entry.
	for off := int64(0); off < impSize && off < 0x4000; off += 20 {
		fo := rvaToOff(impRVA + off)
		if fo < 0 {
			return false
		}
		var d [20]byte
		if _, err := f.ReadAt(d[:], fo); err != nil {
			return false
		}
		allZero := true
		for _, b := range d {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			break
		}
		nameRVA := int64(binary.LittleEndian.Uint32(d[12:]))
		if nameRVA == 0 {
			continue
		}
		nfo := rvaToOff(nameRVA)
		if nfo < 0 {
			continue
		}
		var buf [24]byte
		if _, err := f.ReadAt(buf[:], nfo); err != nil {
			continue
		}
		name := strings.ToLower(strings.TrimRight(string(buf[:]), "\x00"))
		if guiImportDLLs[name] {
			return true
		}
	}
	return false
}
