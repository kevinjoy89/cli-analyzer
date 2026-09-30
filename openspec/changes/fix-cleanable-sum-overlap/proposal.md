## Why

AI 编目引入父子子目录规则（如 `.kimi-code` 与 `.kimi-code/bin`、`.marscode` 与 `.marscode/.ckg` 同时为可处置项）后，`cleanableBytes` 的朴素求和把同一份字节计入两遍：实机出现「可处置 7.9G > 总占用 3.9G」（marscode）、「可处置 940MB > 总占用 556MB」（kimi），且 `userBytes = footprint − cleanable` 被挤成负数钳到 0，安装占用列失去意义（opencode 的 347M 安装根显示 0 B）。

## What Changes

- `cleanableBytes` 语义修正：按**极大路径**口径求和——可处置项与其祖先可处置项并存时，子目录字节不重复计入；保证 `cleanableBytes ≤ footprintBytes`，`userBytes` 恢复含义。
- cleaner 父子同选去重：选中集合中某路径严格位于另一选中路径之内时跳过（随父项处置），不再产生二次删除报错；新增跳过原因文案（三语）。
- GUI 确认弹窗合计与 Clean 调用同口径去重（`pathInside` 纯函数 + 测试）。

## Capabilities

### New Capabilities
<!-- 无 -->

### Modified Capabilities
- `tool-attribution`: 「归因目录均为可处置项」需求中的 `cleanableBytes` 求和口径由朴素求和改为极大路径求和，并新增不重复计数场景。

## Impact

- `internal/scanner/attribute.go`（finalize 合计）、`internal/cleaner/clean.go`（同选去重）、`frontend/src/render.ts` + `frontend/src/lib/grouping.ts`（确认合计同口径）。
- i18n：`cln.subsumed` 三语。
- 修复后本机 AI 段 cleanableBytes 由 42.1G 回落到 ≈31G（与 footprint 同量级）。
