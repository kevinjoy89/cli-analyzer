package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// mkSharedBin 构造一个含 node 家族命令 + 其他工具的共享 bin 目录
func mkSharedBin(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestNodejsInstallRootSharedDir 验证共享 bin 目录（/usr/local/bin 形态：node
// 与 docker/python/git 等混放）不得被整目录算作 nodejs 安装根——手动放置
// 的 node 只是单个二进制，目录里其余命令不属于 node 运行时（此前仅检查
// 目录内存在 node，/usr/local/bin 整个 78MB 被计入 nodejs footprint）。
func TestNodejsInstallRootSharedDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix 场景用例")
	}
	shared := mkSharedBin(t, "node", "docker", "python3")
	if got := nodejsInstallRoot(shared); got != "" {
		t.Errorf("共享 bin 目录被误判为 nodejs 安装根: %q", got)
	}
	// 纯 Node 运行时目录（nvm 版本 bin 形态）应命中
	ndir := mkSharedBin(t, "node", "npm", "npx", "corepack", "node-gyp")
	if got := nodejsInstallRoot(ndir); got != ndir {
		t.Errorf("纯 Node 运行时目录应命中，got %q want %q", got, ndir)
	}
	// Windows 官方安装器目录形态（含 .ps1、.man 清单与文档文件）应命中
	wdir := t.TempDir()
	for _, n := range []string{"node.exe", "npm.cmd", "npx.cmd", "corepack.cmd", "npm.ps1", "npx.ps1", "corepack.ps1", "install_tools.bat", "nodevars.bat", "CHANGELOG.md", "LICENSE", "node_etw_provider.man", "node_perfctr_provider.man"} {
		if err := os.WriteFile(filepath.Join(wdir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := nodejsInstallRoot(wdir); got != wdir {
		t.Errorf("Windows 官方安装目录应命中（.ps1/.man/文档非其他工具），got %q", got)
	}
}

// TestGitInstallRootSharedDir 验证共享 bin 目录不得被整目录算作 git 安装根
// （git 与 python 等混放时，目录不是 Git 专用命令目录）
func TestGitInstallRootSharedDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix 场景用例")
	}
	shared := mkSharedBin(t, "git", "python3")
	if got := gitInstallRoot(shared); got != "" {
		t.Errorf("共享 bin 目录被误判为 git 安装根: %q", got)
	}
	// Git 专用命令目录（Git for Windows cmd/ 形态）应命中
	gdir := mkSharedBin(t, "git", "git-lfs", "scalar", "tig")
	if got := gitInstallRoot(gdir); got != gdir {
		t.Errorf("Git 专用命令目录应命中，got %q want %q", got, gdir)
	}
	// Git 官方安装器目录形态（含 gitk/git-gui GUI 附属、git-bash/git-cmd
	// 启动器与文档）应命中——真实 Git for Windows 的 cmd/ 目录即此形态
	mdir := t.TempDir()
	for _, n := range []string{"git", "gitk", "git-gui", "git-lfs", "tig", "git-bash", "git-cmd"} {
		if err := os.WriteFile(filepath.Join(mdir, n), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{"README.txt", "LICENSE", "whatsnew.txt"} {
		if err := os.WriteFile(filepath.Join(mdir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := gitInstallRoot(mdir); got != mdir {
		t.Errorf("Git 官方安装目录应命中（gitk/git-gui/git-bash/git-cmd 是随 Git 分发的附属），got %q", got)
	}
}
