package uninstall

import "testing"

// 红测试：pip/pip3 是 Python 生态核心工具，与 npm/yarn/pnpm 同级，必须被拦截
func TestIsBlockedPip(t *testing.T) {
	for _, name := range []string{"pip", "pip3", "PIP", "Pip3"} {
		if !IsBlocked(name) {
			t.Errorf("IsBlocked(%q) = false, want true (system-critical Python tool)", name)
		}
	}
}
