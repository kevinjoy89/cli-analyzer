package platform

import (
	"os"
	"runtime"
	"time"
)

// RenameReplace 原子替换 rename：src 覆盖 dst。Windows 上刚写入/刚被替换
// 的文件可能被杀软或索引服务瞬时锁定（MoveFileEx 返回 ACCESS_DENIED，
// os.IsPermission 为 true），短暂退避重试即可成功——并发 Save 与 GUI+CLI
// 双进程写配置/缓存时均可能命中。unix 上 rename 原子覆盖，直接返回。
func RenameReplace(src, dst string) error {
	for attempt := 0; ; attempt++ {
		err := os.Rename(src, dst)
		if err == nil || runtime.GOOS != "windows" || !os.IsPermission(err) || attempt >= 9 {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
}
