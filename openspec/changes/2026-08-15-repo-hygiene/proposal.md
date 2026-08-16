# 仓库卫生：README 漂移、死参数、变更归档

## Why

仓库存在三类"积累性债务"，都小但持续误导：

1. **README 漂移**：`README.md` / `README.zh-CN.md` 引用的 `docs/screenshots/app-*.png` 从未提交（图片断裂）；CLI 命令列表漏 `uninstall`（main.go 实际支持）；版本示例陈旧（0.3.3 vs 实际 0.3.8）
2. **死参数**：`scan --full` 已无任何行为（`findUnattributed` 恒执行，`scanner.go` 注释"孤儿数据始终计算"），flag/Options.Full/传参三处残留，误导用户与维护者
3. **流程未闭环**：`openspec/changes/` 下 5 个已完成变更（tasks 全部打勾）未移入 `archive/`

## What Changes

- **README 修复**：删除断裂的截图表格（替换为"截图待补"占位注释，截图生成后恢复）；Usage 命令列表补 `uninstall <tool>` 两条；版本示例更新为 0.3.8
- **移除 `-full` 死参数**：`internal/cli/scan.go` 删 flag 定义与传参、`internal/scanner/types.go` 删 `Options.Full` 字段、`internal/scanner/scanner.go:109` 改 `Options{}`
- **OpenSpec 归档**：`openspec/changes/2026-08-14-*` 下 5 个已完成变更（fix-orphan-gui-data / windows-test-portability / git-family-merge / npm-global-shim-attribution / cleanup-ui-tweaks）移入 `openspec/changes/archive/`（用 `git mv` 保留历史；有 openspec CLI 则用 `openspec archive`）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯文档/死代码/流程 -->

## Impact

- **代码**：`README.md`、`README.zh-CN.md`、`internal/cli/scan.go`、`internal/scanner/types.go`、`internal/scanner/scanner.go`；`openspec/changes/` 目录结构
- **行为**：`scan --full` 变为未知 flag（flag.Parse 报错）——语义上它是无操作参数，移除合理；README 不再有断裂图片、命令列表与实现一致
- **验证**：`go vet ./...` + `go test ./internal/scanner/ ./internal/cli/` 全绿；`grep -rn "docs/screenshots" README*.md` 无输出
- **不引入**：不改扫描/清理语义；不重写 README 内容（只修漂移点）
