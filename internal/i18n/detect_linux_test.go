//go:build linux

package i18n

import "testing"

// TestDetectLinuxCLocale 验证 Linux 的 C/POSIX locale（服务器常见默认）
// 解析为英文界面而非回退中文（detect 曾整体跳过 C 导致 LANG=C 环境中文界面）
func TestDetectLinuxCLocale(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	if got := DetectSystem(); got != "en" {
		t.Errorf("LC_ALL=C 应解析为 en，got %q", got)
	}
	t.Setenv("LC_ALL", "C.UTF-8")
	if got := DetectSystem(); got != "en" {
		t.Errorf("LC_ALL=C.UTF-8 应解析为 en，got %q", got)
	}
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")
	if got := DetectSystem(); got != "en" {
		t.Errorf("ja 无法识别时应继续查 LANG，got %q", got)
	}
}
