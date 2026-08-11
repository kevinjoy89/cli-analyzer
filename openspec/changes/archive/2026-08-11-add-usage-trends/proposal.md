## Why

现有扫描结果是"一次性快照"——只能回答"现在多大"，回答不了"磁盘怎么又满了 / 比上周多了多少"。磁盘是持续增长的，趋势才是用户真正的问题，也是判断"该不该清理、清理后是否反弹"的依据。

## What Changes

- **扫描时间线持久化**：每次扫描把 `(时间, 工具, footprint/cleanable/user)` 追加到本地 SQLite 时间线，跨会话保留
- **趋势视图**：GUI 新增趋势视图，展示总占用 / 可清理量随时间的折线，以及"cleanable 增量 Top N"增长大户排行
- **阈值提醒**：某工具可清理量超过阈值时，GUI 顶部出现提示条；阈值可配置
- **数据落地**：趋势库存放于应用数据根（与内置回收站同根，`<data-root>/cli-analyzer/`），数据与缓存目录语义分离
- 历史记录按策略滚动清理（如保留最近 90 天 / 按条数上限），避免无限增长

## Capabilities

### New Capabilities
- `usage-trends`: 扫描快照时间线持久化、趋势视图、cleanable 阈值提醒与历史数据滚动清理

### Modified Capabilities
<!-- 无：openspec/specs/ 目前为空，且趋势属新增能力，不改动现有 spec 要求 -->

## Impact

- **新增** `internal/history`：SQLite 时间线的写入、查询、滚动清理（无第三方依赖或引入 `modernc.org/sqlite`）
- `internal/platform`：新增应用数据根 `DataRoot()`（与内置回收站 change 共用，实现时保持两份 change 的接口一致）
- `gui/service.go`：新增 `GetTrends` / `GetTopGrowers` / `GetReminderConfig` 等 Wails 绑定
- `internal/cli`：`scan` 追加历史；新增 `trends` 子命令（CLI 侧查看）
- `frontend`：趋势视图（手写 SVG/DOM 折线，延续现有零依赖风格）+ 提醒条
