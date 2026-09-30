## Why

本机扫描 295 个工具行，16 个 AI 编目条目埋在 279 行通用 CLI 下面——上一轮编目给了 AI 条目徽章，但列表仍是平铺，「AI 优先」只停留在标签层面。定位收窄为「AI 编码工具的磁盘管家」后（已决策不扩展通用 IDE、不删非 AI 能力），列表呈现需要同向聚焦：AI 段置顶常展开，279 行通用工具默认折叠为一行摘要（保留其 45G 占用的可见性，一键展开）。

## What Changes

- GUI 工具列表分两段：「AI 编码工具」（category 非空，agent + ai-ide）置顶且始终展开；「其他 CLI 工具」默认折叠，段头展示条目数与合计占用，点击展开/收起（会话内记忆，重扫不重置）。
- 排序键在段内继续生效；搜索过滤同时作用于两段，其他 CLI 工具段存在命中时自动展开（避免"搜到了却像消失"）。
- 纯逻辑抽 `lib/grouping.ts`（过滤 + 分段 + 合计）并补 vitest；i18n 补段头文案三语；CSS 新增段头样式。
- 后端零改动（category 字段已在 JSON 契约中）。

## Capabilities

### New Capabilities
<!-- 无 -->

### Modified Capabilities
- `ai-tool-catalog`: 新增「AI 优先分组视图」要求——工具列表按 AI 段/其他 CLI 工具段分组展示，其他 CLI 工具段默认折叠带合计，搜索联动自动展开。

## Impact

- 仅前端：`frontend/src/render.ts`（renderToolList 分组渲染）、`frontend/src/state.ts`（折叠状态）、`frontend/src/lib/grouping.ts`（新增，纯逻辑 + 测试）、`frontend/src/style.css`（段头样式）。
- `internal/i18n/locales/*.json`（ui.group.ai / ui.group.others 三语）。
- 无后端逻辑变化；CLI 输出不变。
