package probe

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain 预热"新可执行文件首次 exec"：macOS 上新建脚本文件首次执行会
// 触发安全扫描（实测全量套件并行负载下可长达 ~3s），而探测超时是产品语义
// （3s，测试不能放宽）。TestMain 先执行一个临时脚本把该一次性开销吸收掉，
// 避免第一个探测用例（TestProbeVersionOrder 等）被误判为超时而偶发失败。
// 注意预热必须 exec 脚本文件本身（shebang 路径），直接 exec /bin/sh 不会
// 触发同一扫描路径（实测 5ms 完成、未吸收任何开销）。Linux/Windows 无此
// 行为，预热仅增加 ~10ms 无害开销。
func TestMain(m *testing.M) {
	if runtime.GOOS != "windows" {
		dir, err := os.MkdirTemp("", "probe-warmup-*")
		if err == nil {
			defer os.RemoveAll(dir)
			script := filepath.Join(dir, "warmup")
			if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err == nil {
				_ = exec.Command(script).Run()
			}
		}
	}
	os.Exit(m.Run())
}
