package scanner

import "testing"

func TestIsVendorHelper(t *testing.T) {
	helper := []string{
		`C:\Program Files (x86)\NetSarang\Xshell 8\installanchorservice.exe`,
		`C:\Program Files (x86)\NetSarang\Xshell 8\RealCmdModule.exe`,
		`C:\Program Files (x86)\NetSarang\xftp 8\installanchorservice.exe`,
		`C:\Apps\Whatever\Uninstall.exe`,
		`C:\Apps\Whatever\setup.exe`,
		`C:\Apps\Whatever\update-agent.exe`,
	}
	for _, p := range helper {
		if !isVendorHelper(p) {
			t.Errorf("isVendorHelper(%q) = false, want true", p)
		}
	}
	real := []string{
		`C:\Program Files\nodejs\node.exe`,
		`C:\Program Files\Git\cmd\git.exe`,
		`C:\Program Files\Git\usr\bin\ssh.exe`,
		`C:\Users\me\.cargo\bin\cargo.exe`,
		`C:\Windows\System32\cmd.exe`,
	}
	for _, p := range real {
		if isVendorHelper(p) {
			t.Errorf("isVendorHelper(%q) = true, want false", p)
		}
	}
}

func TestIsVendorInstallDir(t *testing.T) {
	skip := []string{
		`C:\Program Files (x86)\NetSarang\Xshell 8`,
		`C:\Program Files (x86)\NetSarang\xftp 8`,
		`C:\Program Files (x86)\NetSarang\Common\7`,
	}
	for _, p := range skip {
		if !isVendorInstallDir(p) {
			t.Errorf("isVendorInstallDir(%q) = false, want true", p)
		}
	}
	keep := []string{
		`C:\Program Files\Git\cmd`,
		`C:\Program Files\nodejs`,
		`C:\Program Files\Git\usr\bin`,
	}
	for _, p := range keep {
		if isVendorInstallDir(p) {
			t.Errorf("isVendorInstallDir(%q) = true, want false", p)
		}
	}
}
