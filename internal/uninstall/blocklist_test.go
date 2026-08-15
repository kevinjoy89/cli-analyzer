package uninstall

import "testing"

// TestIsBlockedBrewFormulaVariants 验证 brew 公式形态的 python（python@3.13）
// 也命中系统关键工具黑名单：扫描器对 brew 安装返回公式名（Cellar/<formula>），
// 仅黑名单字面量 python3.13 无法拦截 brew 公式形态，代跑会真实执行
// `brew uninstall python@3.13`，可能破坏依赖 python 的工具链。
func TestIsBlockedBrewFormulaVariants(t *testing.T) {
	for _, name := range []string{"python@3.13", "python@3.12", "python@3.11", "Python@3.13", "PYTHON@3.12"} {
		if !IsBlocked(name) {
			t.Errorf("IsBlocked(%q) = false, want true (brew formula of system python)", name)
		}
	}
}
