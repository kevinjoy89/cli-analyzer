// Package cmdexec 提供子进程执行的共享辅助：在（含增强的）PATH 中解析
// 可执行文件，并为子进程注入完整 PATH。uninstall 与 upgrade 的代跑/
// 查询共用（design D5），避免 upgrade→uninstall 的异味依赖方向。
package cmdexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli-analyzer/internal/platform"
)

// ResolveCommand 在（含增强的）PATH 目录中解析命令的绝对路径。
// GUI 从 Finder 启动时进程 PATH 是系统最小集，直接 exec "npm" 会报
// "executable file not found in $PATH"——与扫描漏工具的根因相同。
func ResolveCommand(bin string) (string, error) {
	for _, dir := range platform.PathDirs(false) {
		p := filepath.Join(dir, bin)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && platform.IsExecutable(st) {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w: %q not found on PATH", os.ErrNotExist, bin)
}

// AugmentedPathEnv 返回含增强目录的 PATH 环境变量，供子进程使用：
// npm 等工具内部会再派生子进程（如 node），子进程继承的 PATH 必须完整。
func AugmentedPathEnv() string {
	return strings.Join(platform.PathDirs(false), string(os.PathListSeparator))
}

// WithPath 返回替换 PATH 后的环境变量切片（保留其余环境）。
// 键匹配大小写不敏感：Windows 系统环境变量名为 "Path"（大写 P 小写 ath），
// 只匹配 "PATH=" 会残留旧变量并追加新变量，子进程拿到重复 PATH。
// GUI 与 CLI 代跑命令共用（行为一致）。
func WithPath(env []string, path string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			continue
		}
		out = append(out, e)
	}
	return append(out, "PATH="+path)
}
