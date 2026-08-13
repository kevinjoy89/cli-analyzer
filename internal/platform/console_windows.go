//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"strings"
)

// IsConsoleExe reports whether a PATH entry is a console (CLI) command.
//
// On Windows, IsExecutable only matches PATHEXT extensions, so GUI-subsystem
// executables (Xshell.exe, wails/electron apps, …) sitting on PATH would be
// reported as CLI tools. The PE optional header's Subsystem field
// (IMAGE_SUBSYSTEM_WINDOWS_GUI == 2) distinguishes them: GUI apps are not
// commands and are filtered out of scans. Non-.exe entries (.cmd/.bat/.com)
// are always treated as console commands.
func IsConsoleExe(path string) bool {
	low := strings.ToLower(path)
	if !strings.HasSuffix(low, ".exe") {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		// Unreadable (permissions, locked) → keep it; better to over-report
		// than to hide a real tool.
		return true
	}
	defer f.Close()

	var dos [64]byte
	if _, err := f.ReadAt(dos[:], 0); err != nil || dos[0] != 'M' || dos[1] != 'Z' {
		return true
	}
	eLfanew := int64(binary.LittleEndian.Uint32(dos[0x3c:]))
	if eLfanew <= 0 {
		return true
	}
	var pe [4]byte
	if _, err := f.ReadAt(pe[:], eLfanew); err != nil || pe[0] != 'P' || pe[1] != 'E' || pe[2] != 0 || pe[3] != 0 {
		return true
	}
	peOff := eLfanew
	optOff := eLfanew + 24
	var coff [20]byte
	if _, err := f.ReadAt(coff[:], eLfanew+4); err != nil {
		return true
	}
	numSec := binary.LittleEndian.Uint16(coff[2:])
	// Optional-header Magic: 0x10b = PE32, 0x20b = PE32+.
	var magic [2]byte
	if _, err := f.ReadAt(magic[:], optOff); err != nil {
		return true
	}
	optSize := 224
	if binary.LittleEndian.Uint16(magic[:]) == 0x20b {
		optSize = 240
	}
	// Subsystem lives at optional-header offset 68 (both PE32 and PE32+).
	var sub [2]byte
	if _, err := f.ReadAt(sub[:], optOff+68); err != nil {
		return true
	}
	// IMAGE_SUBSYSTEM_WINDOWS_GUI == 2 → GUI app.
	if binary.LittleEndian.Uint16(sub[:]) == 2 {
		return false
	}
	// Subsystem claims console, but some GUI apps are built as CUI (no console
	// window flashes) — e.g. NetSarang Xshell. Verify via the import table.
	if importsGUI(f, peOff, optOff, optSize, numSec) {
		return false
	}
	return true
}
