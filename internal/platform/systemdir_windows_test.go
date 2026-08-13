//go:build windows

package platform

import "testing"

func TestIsSystemDirWindows(t *testing.T) {
	t.Setenv("WINDIR", `C:\Windows`)
	cases := []struct {
		dir  string
		want bool
	}{
		{`C:\Windows`, true},
		{`C:\Windows\System32`, true},
		{`c:\windows\system32`, true},         // 大小写不敏感
		{`C:\Windows\System32\OpenSSH`, true}, // ssh.exe 所在
		{`C:\Windows\System32\WindowsPowerShell\v1.0`, true},
		{`C:\Windows\SysWOW64`, true},
		{`C:\Program Files\Git\cmd`, false},
		{`C:\Users\me\.cargo\bin`, false},
		{`C:\Program Files\nodejs`, false},
		{`C:\WindowsApps`, false}, // 不在 Windows 目录下
	}
	for _, c := range cases {
		if got := isSystemDir(c.dir); got != c.want {
			t.Errorf("isSystemDir(%q) = %v, want %v", c.dir, got, c.want)
		}
	}
}
