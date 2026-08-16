package probe

import (
	"time"

	"cli-analyzer/internal/scanner"
)

// FillVersions 为版本未知（空）的工具批量填充探测版本（缓存优先）。
// budget 为总时间预算（<=0 表示不限）；供 CLI 同步输出前调用，避免拖慢打印。
func FillVersions(tools []scanner.Tool, budget time.Duration) {
	deadline := time.Now().Add(budget)
	for i := range tools {
		if tools[i].Version != "" || len(tools[i].Binaries) == 0 {
			continue
		}
		b := tools[i].Binaries[0]
		// 安全门同 GUI probeAll：InstOther 需二进制不在 .app 包内或命令名
		// 是公认 CLI（见 scanner.ProbeSafeBinary）
		if !scanner.ProbeSafeBinary(scanner.Installer(tools[i].Installer), b.Real, b.Name) {
			continue
		}
		if budget > 0 && time.Now().After(deadline) {
			break
		}
		if v, ok := CachedOrRun(b.Real, b.Path, b.Size); ok && v != "" {
			tools[i].Version = v
		}
	}
	Save()
}
