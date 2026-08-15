package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCollectBackupsSkipsSymlinkTarget 验证 .bak/.old 备份文件若同时是某个
// PATH 命令（symlink）的真实目标，不得列为 backup 清理项——删除它会直接
// 破坏仍可用的命令（此前 isCommand 只记录 b.Path（symlink 入口），未记录
// b.Real，指向 kimi.bak 的 kimi 命令其真实文件会被当成可清理备份）。
func TestCollectBackupsSkipsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "kimi.bak")
	if err := os.WriteFile(realFile, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realFile, filepath.Join(dir, "kimi")); err != nil {
		t.Fatal(err)
	}
	// 再放一个真·备份文件（无命令引用），应正常列为 backup
	orphanBak := filepath.Join(dir, "orphan.old")
	if err := os.WriteFile(orphanBak, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := map[string]*toolBuilder{
		"kimi": {name: "kimi", binaries: []Binary{
			{Path: filepath.Join(dir, "kimi"), Real: realFile, Size: 1},
		}},
	}
	got := collectBackups(tools, []string{"kimi"})
	if _, ok := got[realFile]; ok {
		t.Error("kimi.bak 是 PATH 命令 kimi 的真实目标，不应列为 backup 清理项")
	}
	if _, ok := got[orphanBak]; !ok {
		t.Error("无命令引用的 orphan.old 应列为 backup 清理项")
	}
}
