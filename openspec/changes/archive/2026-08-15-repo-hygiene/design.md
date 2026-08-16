# Design: 仓库卫生（README 漂移、死参数、变更归档）

## Context

三类小债务长期存在：README 截图断裂（docs/screenshots 从未提交）、CLI 列表漏 uninstall、`scan --full` 死参数残留三处、openspec 已完成变更未归档。每项都小，但持续误导：新用户看断裂截图/缺命令，维护者被死 flag 迷惑，openspec 流程状态不真实（changes/ 下"已完成但未归档"）。

## Goals / Non-Goals

**Goals**
- README 与实现一致：截图占位、命令列表补全、版本示例更新
- `-full` 参数全链路移除（flag/Options/传参），不留死代码
- 5 个已完成变更归档，openspec/changes/ 恢复为空（无活跃变更）

**Non-Goals**
- 不重写 README 内容/结构（只修漂移点）
- 不生成截图（超出范围；占位注释 + 恢复指引）
- 不清理 openspec/specs/（能力规格保留）

## Decisions

### D1: 截图占位而非删除

两个 README 的截图表格（英文版 app-light-en.png/app-dark-en.png、中文版 app-light.png/app-dark.png）替换为一行占位注释：`<!-- 截图待补：docs/screenshots/app-*.png 生成后恢复此表 -->`。截图是营销/README 价值点，删除会丢失意图；占位保留恢复指引。

### D2: -full 三处移除

- `internal/cli/scan.go:21` 删 `full := fs.Bool("full", ...)`；`:42` 的 `Options{Full: *full, ...}` 改 `Options{NoCache: *noCache, ToolFilter: filters}`
- `internal/scanner/types.go` Options 删 `Full bool` 字段（注释一并删）
- `internal/scanner/scanner.go:109` `Options{Full: opts.Full}` 改 `Options{}`

风险点：`scan --full` 将变为未知 flag（`flag.Parse` 报错、exit 1）。这是预期的破坏性小变更——该参数本就无操作；README 未宣传过 `--full`，脚本依赖可能性极低。

### D3: 归档用 git mv 保留历史

优先 `openspec archive <slug>`（若 CLI 可用）；否则 `git mv openspec/changes/<slug> openspec/changes/archive/`（5 个目录）。归档后 `ls openspec/changes/` 应为空。注意：本变更执行时若还有"进行中"的变更（如同日的其他 4 个 change），只归档 tasks 已完成的 5 个旧变更，不归档进行中的。

## Risks / Trade-offs

- [移除 --full 破坏依赖脚本] → 参数本为无操作，破坏面趋近于零；若确有使用者，git 历史可恢复
- [归档顺序与进行中变更冲突] → D3 明确只归档 5 个已完成旧变更

## Migration Plan

无数据/持久化变化。README 即时生效；`-full` 移除后 `go vet` + 相关包测试全绿即验证；归档为纯目录移动。
