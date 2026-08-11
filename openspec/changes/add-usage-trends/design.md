## Context

`scanner.ScanResult` 已含 `Totals` 与每工具 `Footprint/Cleanable/User`，是现成的历史快照来源；前端零框架（原生 TS + DOM），趋势图需延续该风格。扫描缓存走 `platform.CacheRoot()`，回收站与趋势将共用 `platform.DataRoot()` 应用数据根（见 add-trash-recycle-bin 的 D1）。动机见 proposal.md - Why，行为契约见 specs/usage-trends/spec.md。

## Goals / Non-Goals

**Goals:**
- 每次扫描落一条持久化历史，跨会话可查询
- 趋势视图：总占用/可清理随时间的折线 + cleanable 增量 Top N
- cleanable 阈值提醒（可配置）
- 历史按保留策略滚动清理，默认 90 天
- 保持跨平台构建简单（不引入 cgo）

**Non-Goals:**
- 不做自动清理调度（launchd/cron 常驻）
- 不做按目录/按安装来源的多维分析
- 不做实时监控或磁盘事件监听
- 不做趋势的撤销/编辑历史

## Decisions

### D1: 存储 → SQLite（纯 Go 驱动）

`<DataRoot>/cli-analyzer/history.db`，使用 `modernc.org/sqlite`（纯 Go，无 cgo，保持 `GOOS=linux/windows go build` 直接交叉编译）。

- **理由**：查询按时间范围、按工具、Top N 都自然表达；未来可能与回收站/提醒共用本地状态，一个库更稳；纯 Go 驱动不破坏项目"pure Go stdlib 可交叉编译"的卖点。
- **备选**：JSONL append-only（零依赖，但查询需全量扫描，且并发写要自管锁）——若实现时确认体积/依赖敏感，可回退，接口不变。

### D2: 表结构

```sql
scans      (id INTEGER PRIMARY KEY, scanned_at TEXT, footprint INTEGER, cleanable INTEGER, user INTEGER)
scan_tools (scan_id INTEGER REFERENCES scans(id), tool TEXT, footprint INTEGER, cleanable INTEGER, user INTEGER)
```

- `scans` 提供时间序列聚合；`scan_tools` 提供每工具增量与 Top N。
- 无历史时（首跑）`scans` 为空，查询返回空列表而非报错。

### D3: 写入时机 → 扫描完成后的调用方层

`history.Record(res)` 由 `gui/service.go` 的 `Scan()` 与 `cli scan` 在扫描成功后调用；`internal/scanner` 核心不感知 history，保持纯核心纯净。写入失败（如 db 锁定）静默降级，不影响扫描主流程。

### D4: 趋势视图与 Top N

- 前端手写 SVG 折线（复用 `hb()` 等工具函数，延续零依赖）。
- `GetTrends(days)` 返回 `{ points: [{date, footprint, cleanable}], topGrowers: [{tool, deltaBytes}] }`。
- **Top N 增量算法**：取最近两次扫描的 `scan_tools`，按 cleanable 差降序取 Top N；历史不足两次时返回空并提示"数据积累中"。

### D5: 阈值提醒 → 扫描完成后计算

扫描完成后计算 `cleanable > threshold` 的工具列表，GUI 顶部显示提醒条。阈值存 `internal/config` 的 `config.json`（与回收站 change 的 D6 同文件），默认值（如 5 GB）可在配置中调整。阈值配置入口复用 add-trash-recycle-bin 引入的首选项面板（见该 change design 的 D8），本 change 不重复实现配置 UI，仅在面板中追加"cleanable 阈值"一项。

### D6: 滚动清理 → 写入后顺带

每次 `history.Record` 后执行 `DELETE` 超出保留范围（默认 90 天）的旧记录，避免无限增长。

### D7: CLI 暴露

新增 `trends` 子命令：打印最近若干次扫描的 ASCII 汇总表与 cleanable 增长 Top N，便于不进 GUI 也能看趋势。

## Risks / Trade-offs

- [引入 SQLite 依赖，项目此前标榜 pure stdlib] → `modernc.org/sqlite` 纯 Go 无 cgo；若不可接受则按 D1 备选回退 JSONL
- [趋势"慢热"，首周无价值] → UI 明确提示"数据积累中"，spec 已覆盖
- [GUI 与 CLI 并发扫描导致 db 写冲突] → SQLite 单连接 + 短事务重试，写入失败静默降级
- [history.db 膨胀] → D6 滚动清理
- [两个 change 都新增 `platform.DataRoot()`] → 实现时先落一份共用实现，两份 tasks 都引用同一接口，避免重复代码

## Migration Plan

首次运行自动建库建表，无迁移。回滚：删除 `history.db` 即回到无历史状态，扫描/清理主流程不受影响。

## Open Questions

无。
