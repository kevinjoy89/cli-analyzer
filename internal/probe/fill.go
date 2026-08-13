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
		if budget > 0 && time.Now().After(deadline) {
			break
		}
		b := tools[i].Binaries[0]
		if v, ok := CachedOrRun(b.Real, b.Size); ok && v != "" {
			tools[i].Version = v
		}
	}
	Save()
}
