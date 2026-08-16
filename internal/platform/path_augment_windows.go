//go:build windows

package platform

import (
	"os"
	"path/filepath"
)

// augmentUserDirs 补充 Windows 上不在 PATH 的已知 CLI 二进制目录。
//
// clink（cmd 增强工具）经注册表注入 cmd，二进制不进入 PATH：其安装目录
// %LocalAppData%\clink 同时含 clink.exe 与用户数据（历史/日志）。不并入
// 发现列表的话 clink 永不归认，其数据目录会被当成孤儿展示。存在性过滤：
// 未安装（目录不存在）时跳过，不影响扫描。
func augmentUserDirs(seen map[string]bool, out []string) []string {
	var dirs []string
	if la := Root(LocalAppData); la != "" {
		dirs = append(dirs, filepath.Join(la, "clink"))
	}
	for _, d := range dirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			abs = d
		}
		if seen[abs] {
			continue
		}
		if st, err := os.Stat(abs); err != nil || !st.IsDir() {
			continue // 不存在则跳过（discover 也会跳过，但这里过滤更干净）
		}
		seen[abs] = true
		out = append(out, abs)
	}
	return out
}
