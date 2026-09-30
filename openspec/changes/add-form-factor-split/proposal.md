## Why

双形态 AI 工具（opencode、codex、dsh、kimi、agy 等）的客户端与 CLI 数据合并在一个条目里：用户能看到总和，却无法回答「客户端占了多少、CLI 占了多少」。数据已经具备（.app 包 = 桌面端安装、AppSupport 桌面目录 vs dotdir CLI 目录），缺的是编目层面的形态标注与展示小计。

## What Changes

- 编目规则新增 `Form` 维度（desktop / cli，空 = 未标按 cli 计）：桌面端 AppSupport 目录、.app 应用包（scanner 按路径自动标注）标 desktop；dotdir、npm 安装根、XDG 目录标 cli。
- `DataDir` JSON 契约透传 `form` 字段。
- GUI 详情页新增「形态占用」小计：桌面端 / CLI 两行，按极大路径口径分别合计（父子包含不重复计数）。
- i18n：ui.form.desktop / ui.form.cli 三语。

## Capabilities

### New Capabilities
<!-- 无 -->

### Modified Capabilities
- `ai-tool-catalog`: 新增「形态占用小计」要求——双形态条目的详情页 SHALL 分桌面端 / CLI 展示占用小计。

## Impact

- `internal/rules/`（Form 字段 + 条目标注）、`internal/scanner/types.go` + `attribute.go`（透传 + .app 自动标注）、`frontend/src/lib/grouping.ts`（formSubtotals + 测试）、`frontend/src/render.ts`（详情页小计块）、`frontend/src/lib/types.ts`、i18n ×3。
- CLI 输出不变（form 为新增可选字段）。
