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
	// Subsystem lives at optional-header offset 68; optional header starts
	// right after the 24-byte PE signature block.
	var sub [2]byte
	if _, err := f.ReadAt(sub[:], eLfanew+24+68); err != nil {
		return true
	}
	// IMAGE_SUBSYSTEM_WINDOWS_GUI == 2
	return binary.LittleEndian.Uint16(sub[:]) != 2
}
