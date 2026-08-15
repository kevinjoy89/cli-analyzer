//go:build windows

package updater

import "testing"

// TestQuoteStartArg 验证 cmd start 的参数引号：路径含空格或 cmd 元字符
// （& | < > ^ ( ) 等 Windows 合法文件名字符）必须加引号，否则 cmd 会拆参数
// 或把它们解释为命令拼接/重定向，打开安装包失败。
func TestQuoteStartArg(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`C:\Users\John Doe\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe`, `"C:\Users\John Doe\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe"`},
		{`C:\Users\Jane&John\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe`, `"C:\Users\Jane&John\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe"`},
		{`C:\Users\jdoe\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe`, `C:\Users\jdoe\Downloads\CLI-Analyzer-0.3.0-windows-amd64-installer.exe`},
		{`"already quoted"`, `"already quoted"`},
	}
	for _, c := range cases {
		if got := quoteStartArg(c.in); got != c.want {
			t.Errorf("quoteStartArg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
