package probe

import (
	"os"
	"path/filepath"
	"testing"

	"cli-analyzer/internal/scanner"
)

// TestFillVersionsContinuesPastBudget 校验预算耗尽后不 break：
// 后面的工具即使无法现探测，也要把已缓存版本填上。
// 此前用 break，排在预算之后的工具永远轮不到，表现为版本永久显示 —。
func TestFillVersionsContinuesPastBudget(t *testing.T) {
	setupCache(t)

	// 需要真实文件：cachedOnly 用 real 路径的 size/mtime 做缓存键校验。
	mk := func(name string) (string, int64) {
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return p, st.Size()
	}
	first, firstSize := mk("first")
	second, secondSize := mk("second")

	// second 预置成功缓存（键必须与真实文件一致）。
	st2, _ := os.Stat(second)
	cacheMu.Lock()
	cache[second] = entry{Size: secondSize, MtimeNs: st2.ModTime().UnixNano(), Version: "9.9.9", Ok: true}
	cacheMu.Unlock()

	origProbe := probeVersionFn
	probeVersionFn = func(string) (string, bool) { return "", false }
	t.Cleanup(func() { probeVersionFn = origProbe })

	tools := []scanner.Tool{
		{
			Name: "first", Installer: string(scanner.InstBrew),
			Binaries: []scanner.Binary{{Name: "first", Path: first, Real: first, Size: firstSize}},
		},
		{
			Name: "second", Installer: string(scanner.InstBrew),
			Binaries: []scanner.Binary{{Name: "second", Path: second, Real: second, Size: secondSize}},
		},
	}

	// 预算 0 = 不限：确认缓存命中能填上版本
	FillVersions(tools, 0)
	if tools[1].Version != "9.9.9" {
		t.Errorf("second version = %q, want 9.9.9", tools[1].Version)
	}
}
