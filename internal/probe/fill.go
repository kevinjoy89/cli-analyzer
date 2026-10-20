package probe

import (
	"time"

	"cli-analyzer/internal/scanner"
)

// FillVersions 为版本未知（空）的工具批量填充探测版本（缓存优先）。
// budget 为总时间预算（<=0 表示不限）；供 CLI 同步输出前调用，避免拖慢打印。
//
// 预算耗尽时**不 break 而是继续遍历**：已缓存的结果命中零成本（不再起
// 子进程），而 break 会让排在后面的工具永远轮不到——它们每次都因预算先
// 被前面的工具吃完而错过，表现为「版本永远显示 —」。真正需要探测时用
// remaining() 检查预算，超了就先跳过。
// FillVersions 为 CLI 版本未知的工具批量填充探测版本（缓存优先）。
// budget 为总时间预算（<=0 表示不限）；供 CLI 同步输出前调用，避免拖慢打印。
//
// 判定条件是 **CLIVersion == ""** 而不是 Version == ""：Version 可能是桌面端
// 版本兜底填上的（纯桌面工具或探测前的双形态工具），拿它做跳过条件会让
// 双形态工具永远探测不到 CLI 版本（agy / kimi 就是这样丢了 CLI 版本）。
//
// 预算耗尽时不 break 而是继续遍历：已缓存的结果命中零成本，而 break 会让
// 排在后面的工具永远轮不到。真正需要探测时用 remaining() 检查预算。
func FillVersions(tools []scanner.Tool, budget time.Duration) {
	deadline := time.Now().Add(budget)
	remaining := func() bool {
		return budget <= 0 || time.Now().Before(deadline)
	}

	// 两遍：先全量填缓存命中（零成本，不受预算影响），再拿剩余预算探测。
	// 分成两遍是必要的：预算耗尽发生在单遍中途时，后面的工具连缓存都来不及
	// 查，而缓存命中恰恰是零成本的。
	for i := range tools {
		if tools[i].CLIVersion != "" || len(tools[i].Binaries) == 0 {
			continue
		}
		b := tools[i].Binaries[0]
		if v, ok := cachedOnly(b.Real, b.Size); ok && v != "" {
			tools[i].CLIVersion = v
			if tools[i].Version == "" {
				tools[i].Version = v
			}
		}
	}

	for i := range tools {
		if tools[i].CLIVersion != "" || len(tools[i].Binaries) == 0 {
			continue
		}
		b := tools[i].Binaries[0]
		// 安全门同 GUI probeAll：InstOther 需二进制不在 .app 包内或命令名
		// 是公认 CLI（见 scanner.ProbeSafeBinary）
		if !scanner.ProbeSafeBinary(scanner.Installer(tools[i].Installer), b.Real, b.Name) {
			continue
		}
		if !remaining() {
			continue // 预算用完：跳过这一项，但继续看后面的缓存命中
		}
		if v, ok := CachedOrRun(b.Real, b.Path, b.Size); ok && v != "" {
			tools[i].CLIVersion = v
			if tools[i].Version == "" {
				tools[i].Version = v
			}
		}
	}
	Save()
}
