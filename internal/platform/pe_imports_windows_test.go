//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writePE builds a minimal PE (32- or 64-bit) with one section holding an
// import table. subsystem: 2 = GUI, 3 = CUI. dllNames empty = no import table;
// multiple DLLs produce one import descriptor each.
func writePE(t *testing.T, is64 bool, subsystem uint16, dllNames ...string) string {
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
	if len(dllNames) > 0 {
		// DataDirectory[1] (import): RVA 0x1000, size covers N descriptors
		// (each 20B) plus an all-zero terminator.
		// PE32 data dirs at opt+96; PE32+ at opt+112.
		dd := 96
		if is64 {
			dd = 112
		}
		binary.LittleEndian.PutUint32(buf[opt+dd+8:], 0x1000)
		binary.LittleEndian.PutUint32(buf[opt+dd+12:], uint32(20*(len(dllNames)+1)))
	}
	// Section header: name "", VirtualSize 0x100, VA 0x1000, RawSize 0x100, RawPtr 0x200.
	sh := opt + int(optSize)
	binary.LittleEndian.PutUint32(buf[sh+8:], 0x100)
	binary.LittleEndian.PutUint32(buf[sh+12:], 0x1000)
	binary.LittleEndian.PutUint32(buf[sh+16:], 0x100)
	binary.LittleEndian.PutUint32(buf[sh+20:], 0x200)
	// DLL 名放在全部描述符（含终止符）之后，避免与描述符区域重叠。
	namesBase := 0x200 + 20*(len(dllNames)+1)
	for i, name := range dllNames {
		// Descriptor i at file 0x200+i*20 (Name RVA → file namesBase+i*0x20).
		binary.LittleEndian.PutUint32(buf[0x200+i*20+12:], 0x1000+uint32(namesBase-0x200+i*0x20))
		copy(buf[namesBase+i*0x20:], name)
	}
	if err := os.WriteFile(p, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIsConsoleExeImportTable(t *testing.T) {
	for _, is64 := range []bool{false, true} { // PE32 与 PE32+（64 位，Xshell 7 属于此类）
		// 子系统=GUI → 直接判定 GUI
		if IsConsoleExe(writePE(t, is64, 2)) {
			t.Errorf("is64=%v: subsystem GUI exe should be filtered", is64)
		}
		// 子系统=CUI 但导入多个 GUI 库（user32+gdi32，Xshell 类伪装 CUI）→ 判定 GUI
		if IsConsoleExe(writePE(t, is64, 3, "user32.dll", "gdi32.dll")) {
			t.Errorf("is64=%v: CUI exe importing multiple GUI dlls must be filtered", is64)
		}
		// 子系统=CUI、仅导入 user32（Git for Windows 控制台工具的真实形态）→ 保留
		if !IsConsoleExe(writePE(t, is64, 3, "user32.dll")) {
			t.Errorf("is64=%v: console exe importing only user32 should be kept", is64)
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
		if !IsConsoleExe(writePE(t, is64, 3)) {
			t.Errorf("is64=%v: console exe without imports should be kept", is64)
		}
	}
}
