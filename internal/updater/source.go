package updater

import (
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"cli-analyzer/internal/buildinfo"
)

// dpkgCommand 可被测试替换为假命令。
var dpkgCommand = "dpkg"

// ResolveInstallSource 返回更新应使用的安装来源：
// 优先构建期注入值（deb/tarball/installer/portable/dmg）；
// 为 unknown 时在 Linux 上尝试 dpkg 探测，确认可执行文件由包管理器管理则按 deb 处理；
// 仍无法确定则返回 "unknown"，调用方按 design D6 打开 Release 页面兜底。
func ResolveInstallSource(exePath string) string {
	if buildinfo.InstallSource != "unknown" {
		return buildinfo.InstallSource
	}
	if goruntime.GOOS == "linux" && probeDPKG(exePath) {
		return "deb"
	}
	return "unknown"
}

// probeDPKG 判断 exePath 是否由 dpkg 包管理器管理（仅 Debian 系可用）。
// 先解析符号链接到真实路径，再 `dpkg -S` 查询归属。
func probeDPKG(exePath string) bool {
	real, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return false
	}
	out, err := exec.Command(dpkgCommand, "-S", real).CombinedOutput()
	if err != nil {
		// dpkg 不存在（非 Debian 系）或文件不受任何包管理 → 非 deb 安装
		return false
	}
	return len(out) > 0
}
