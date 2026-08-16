## Why

GUI 验收反馈两个小问题：① 安全清理详情页中 0 字节子项（如 `npm-cache\_update-notifier-last-checked`）被设计为不可勾选（`selectable = !!(s.id && s.bytes > 0)`），用户想清理空文件/空目录却选不了；② 子项类型与父项不同时渲染琥珀色类型标签（`_logs` → 「日志」），logs 类子项在安全清理项里看起来像独立类别，用户认为不应贴「日志」标签。

## What Changes

- **0 字节子项可选**：`subRows` 的 selectable 判定去掉 `bytes > 0` 门槛——有 id 的子项即可勾选；0 字节项照常显示 `0 B`，清理后由 `applyCleanLocally` 正常移除（释放 0 字节，清除条目）
- **logs 子项不贴「日志」标签**：子项类型标签条件增加 `s.kind !== 'logs'`——`_logs`/`*.log` 子项不再显示琥珀色「日志」标签（logs 类型本身保留：回收站展示、精确清理类型仍用）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯前端交互/展示调整 -->

## Impact

- **代码**：`frontend/src/main.ts`（subRows）
- **验证**：前端测试 18/18 通过；`tsc && vite build` 通过；重建安装包/便携 zip
- **不引入**：后端 subKind 分类不变（logs 类型仍用于回收站与清理语义）；其余子类型标签（old-version/download 等）保留
