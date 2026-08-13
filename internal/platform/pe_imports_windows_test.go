//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writePE builds a minimal PE (32- or 64-bit) with one section holding an
// import table. subsystem: 2 = GUI, 3 = CUI. dllName "" = no import table.
func writePE(t *testing.T, is64 bool, subsystem uint16, dllName string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake.exe")
	buf := make([]byte, 0x400)
	buf[0], buf[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(buf[0x3c:], 0x80) // e_lfanew
	pe := 0x80
	buf[pe], buf[pe+1], buf[pe+2], buf[pe+3] = 'P', 'E', 0, 0
	coff := pe + 4
	binary.LittleEndian.PutUint16(buf[coff:], 0x14c) // i386
	binary.LittleEndian.PutUint16(buf[coff+2:], 1)   // 1 section
	optSize := uint16(0xE0)
	if is64 {
		binary.LittleEndian.PutUint16(buf[coff:], 0x8664) // AMD64
		optSize = 0xF0
	}
	binary.LittleEndian.PutUint16(buf[coff+16:], optSize)
	opt := coff + 20
	if is64 {
		binary.LittleEndian.PutUint16(buf[opt:], 0x20b) // PE32+
	} else {
		binary.LittleEndian.PutUint16(buf[opt:], 0x10b) // PE32
	}
	binary.LittleEndian.PutUint16(buf[opt+68:], subsystem)
	if dllName != "" {
		// DataDirectory[1] (import): RVA 0x1000, size 0x28 (1 desc + null).
		// PE32 data dirs at opt+96; PE32+ at opt+112.
		dd := 96
		if is64 {
			dd = 112
		}
		binary.LittleEndian.PutUint32(buf[opt+dd+8:], 0x1000)
		binary.LittleEndian.PutUint32(buf[opt+dd+12:], 0x28)
	}
	// Section header: name "", VirtualSize 0x100, VA 0x1000, RawSize 0x100, RawPtr 0x200.
	sh := opt + int(optSize)
	binary.LittleEndian.PutUint32(buf[sh+8:], 0x100)
	binary.LittleEndian.PutUint32(buf[sh+12:], 0x1000)
	binary.LittleEndian.PutUint32(buf[sh+16:], 0x100)
	binary.LittleEndian.PutUint32(buf[sh+20:], 0x200)
	if dllName != "" {
		// Import descriptor at file 0x200: Name RVA = 0x1020 → file 0x220.
		binary.LittleEndian.PutUint32(buf[0x200+12:], 0x1020)
		copy(buf[0x220:], dllName)
	}
	if err := os.WriteFile(p, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIsConsoleExeImportTable(t *testing.T) {
	for _, is64 := range []bool{false, true} { // PE32 与 PE32+（64 位，Xshell 7 属于此类）
		// 子系统=GUI → 直接判定 GUI
		if IsConsoleExe(writePE(t, is64, 2, "")) {
			t.Errorf("is64=%v: subsystem GUI exe should be filtered", is64)
		}
		// 子系统=CUI 但导入 user32（Xshell 类 GUI 应用伪装 CUI）→ 仍判定 GUI
		if IsConsoleExe(writePE(t, is64, 3, "user32.dll")) {
			t.Errorf("is64=%v: CUI exe importing user32 must be filtered", is64)
		}
		// 子系统=CUI、仅导入 kernel32（真命令行工具）→ 保留
		if !IsConsoleExe(writePE(t, is64, 3, "kernel32.dll")) {
			t.Errorf("is64=%v: console exe importing kernel32 should be kept", is64)
		}
		// 子系统=CUI、导入 shell32（命令行工具也会用）→ 保留
		if !IsConsoleExe(writePE(t, is64, 3, "shell32.dll")) {
			t.Errorf("is64=%v: console exe importing shell32 should be kept", is64)
		}
		// 无导入表、CUI → 保留
		if !IsConsoleExe(writePE(t, is64, 3, "")) {
			t.Errorf("is64=%v: console exe without imports should be kept", is64)
		}
	}
}
