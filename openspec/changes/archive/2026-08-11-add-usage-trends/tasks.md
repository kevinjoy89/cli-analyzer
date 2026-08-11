## 1. 基础：platform.DataRoot + 依赖

- [x] 1.1 确认 `internal/platform` 的 `DataRoot()` 实现（与 add-trash-recycle-bin 共用同一接口，避免重复代码）
- [x] 1.2 引入 `modernc.org/sqlite` 依赖（纯 Go，无 cgo，保持交叉编译）

## 2. 核心：internal/history

- [x] 2.1 建库建表（`scans` / `scan_tools`），首次运行自动创建
- [x] 2.2 实现 `Record(res)`：写入快照与各工具行；写入失败静默降级，不影响扫描主流程
- [x] 2.3 实现 `Trends(days)`：返回 `{ points: [{date, footprint, cleanable}], topGrowers }`
- [x] 2.4 实现 TopGrowers：最近两次扫描的 cleanable 差降序取 Top N；历史不足两次返回空
- [x] 2.5 实现滚动清理：每次 `Record` 后删除超出保留范围（默认 90 天）的旧记录
- [x] 2.6 为 `internal/history` 编写单元测试（写入/查询往返、Top N、滚动清理），用临时文件 db 隔离真实文件系统

## 3. 集成

- [x] 3.1 `gui/service.go` 的 `Scan()` 成功后调用 `history.Record`
- [x] 3.2 `cli scan` 成功后调用 `history.Record`
- [x] 3.3 `gui/service.go` 新增 `GetTrends` / `GetTopGrowers` / `GetReminderConfig` 绑定

## 4. GUI 前端

- [x] 4.1 趋势视图：手写 SVG 折线展示总占用/可清理随时间变化；历史不足时提示"数据积累中"
- [x] 4.2 cleanable 增量 Top N 排行展示
- [x] 4.3 阈值提醒条：cleanable 超过阈值（默认 5 GB，可配置）的工具在界面顶部提示
- [x] 4.4 阈值配置项接入首选项面板（复用 add-trash-recycle-bin 的首选项面板与入口，不重复实现）

## 5. CLI

- [x] 5.1 新增 `trends` 子命令：打印最近若干次扫描的 ASCII 汇总表与 cleanable 增长 Top N

## 6. 收尾

- [x] 6.1 全量测试 `go test ./...`
- [x] 6.2 更新 README：趋势功能说明与 `trends` 命令用法
