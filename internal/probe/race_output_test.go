package probe

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestProbeConcurrentOutputNoRace 探测命令同时写 stdout 与 stderr 时
// runWithTimeout 不得有数据竞态（Start+Wait 分离后两个复制 goroutine
// 并发写同一 bytes.Buffer——无锁即 race）。
func TestProbeConcurrentOutputNoRace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix shell 场景")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "both")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n(while :; do echo -n o; done) & (while :; do echo -n e >&2; done) & wait\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 无限循环命令必然超时（ok=false）——测试目的仅验证超时路径的
	// 并发输出（stdout/stderr 双写）在 race detector 下无数据竞态
	_, _ = ProbeVersion(script)
}
