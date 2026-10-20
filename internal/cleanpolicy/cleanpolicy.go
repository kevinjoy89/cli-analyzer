// Package cleanpolicy 判定某个 Kind 是否具备「批量清理资格」。
//
// 这是 CLI `clean --all` 与 GUI 批量选择的**唯一**判定处：两处各写一份白名单
// 会漂移（用户会看到「命令行清了 7 类，界面清了 5 类」），而这类漂移在本项目
// 已经发生过一次（CLI 与 GUI 的清理合计口径分叉）。
//
// 为什么不按 Tier 推导：Tier 表达「能否安全删除」，批量资格表达「默认替用户
// 全选时该不该包含它」。runtime 是可重下的 safe，但删除会立即破坏工具可用性，
// 所以它不在批量集合里。详见 docs/adr/0005-tier-is-not-bulk-cleanable.md。
package cleanpolicy

import (
	"sort"

	"cli-analyzer/internal/config"
)

// defaultBulkKinds 是内置的可批量清理 Kind。
//
// 这份清单与 Tier 无关，刻意保持显式：新增一个 Kind 时应当**有意决定**它要
// 不要进一键全选，而不是被 Tier 自动带进来。
var defaultBulkKinds = []string{
	"cache",
	"old-version",
	"backup",
	"download",
	"index",
	"telemetry",
	"logs",
}

// BulkKinds 返回最终的可批量清理 Kind 集合：(内置默认 ∪ cfg.BulkKinds) − cfg.BulkExclude。
// 结果按字典序排序，便于展示与测试比对。
func BulkKinds(cfg config.CleanConfig) []string {
	set := make(map[string]bool, len(defaultBulkKinds)+len(cfg.BulkKinds))
	for _, k := range defaultBulkKinds {
		set[k] = true
	}
	for _, k := range cfg.BulkKinds {
		if k = normalize(k); k != "" {
			set[k] = true
		}
	}
	for _, k := range cfg.BulkExclude {
		delete(set, normalize(k))
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// BulkEligible 报告某个 Kind 是否可被 `clean --all` 默认选中。
//
// 只影响「批量默认选择」：用户逐项确认或显式传 id 时不受此限（cleaner 不看
// 这个函数，它只处理调用方已经选好的 id）。
func BulkEligible(kind string, cfg config.CleanConfig) bool {
	k := normalize(kind)
	if k == "" {
		return false
	}
	for _, ex := range cfg.BulkExclude {
		if normalize(ex) == k {
			return false // 剔除优先于追加与内置默认
		}
	}
	for _, d := range defaultBulkKinds {
		if d == k {
			return true
		}
	}
	for _, add := range cfg.BulkKinds {
		if normalize(add) == k {
			return true
		}
	}
	return false
}

func normalize(s string) string {
	// Kind 是自由字符串，但实际取值都是小写短横线；只去空白不做大小写折叠，
	// 避免把用户有意区分的大小写变体悄悄合并。
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			out = append(out, r)
		}
	}
	return string(out)
}
