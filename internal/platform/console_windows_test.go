//go:build windows

package platform

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// writeFakePE writes a minimal file whose header declares the given
// IMAGE_SUBSYSTEM value (2 = GUI, 3 = CUI).
func writeFakePE(t *testing.T, subsystem uint16) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fake.exe")
	buf := make([]byte, 0x100)
	buf[0], buf[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(buf[0x3c:], 0x80) // e_lfanew
	buf[0x80], buf[0x81], buf[0x82], buf[0x83] = 'P', 'E', 0, 0
	binary.LittleEndian.PutUint16(buf[0x80+24+68:], subsystem)
	if err := os.WriteFile(p, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIsConsoleExe(t *testing.T) {
	if got := IsConsoleExe(writeFakePE(t, 3)); !got { // CUI console app
		t.Error("CUI exe should be a console command")
	}
	if got := IsConsoleExe(writeFakePE(t, 2)); got { // GUI app (Xshell etc.)
		t.Error("GUI-subsystem exe must be filtered out")
	}
	// non-.exe PATH entries are always commands
	cmd := filepath.Join(t.TempDir(), "code.cmd")
	if err := os.WriteFile(cmd, []byte("@echo off\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsConsoleExe(cmd) {
		t.Error(".cmd should be a console command")
	}
}
