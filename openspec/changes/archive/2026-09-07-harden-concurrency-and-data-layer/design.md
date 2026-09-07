# Design: harden-concurrency-and-data-layer

## Context

加固审计（v0.4.0 后）定位了一批并发与数据层缺陷，详见 proposal.md。当前实现的关键约束：

- GUI 是 Wails 薄边界（`gui/service.go`），前端靠 `scan:done` / `probe:done` 事件与轮询接口驱动渲染；事件契约不能破坏。
- 扫描管线（scanner）本身无并发入口问题，问题集中在 GUI 触发层与历史写入。
- 前端本地化文案含结构标记（`<b>`），`t()` 是纯字符串替换，转义必须在插值前对动态值进行。
- 本变更的修复已实现并通过 `go test -race ./...` 与前端 tsc/vitest 全量验证；本文档记录决策供追溯。

## Goals / Non-Goals

**Goals:**
- GUI 并发边界无数据竞态（`-race` 干净）。
- 扫描触发语义可预测：至多一轮在执行，无重复历史。
- 数据层故障可自愈：history.db 损坏不瘫痪趋势功能，且不静默丢数据。
- 代跑命令永不携带空参数；PATH 解析失败有明确反馈。
- 前端动态值渲染统一转义。

**Non-Goals:**
- disk 遍历的 ctx 取消（重构成本高，另行提议）。
- 下载前 release 重新拉取的 TOCTOU 收窄（概率极低）。
- 历史库的迁移框架（当前无 schema 版本需求）。

## Decisions

- **D1 扫描 single-flight 采用丢弃语义**（ADR-0001）：`ScannerService` 增加互斥锁保护的 `scanning` 标志，`Scan`/`ScanIfChanged` 共用；进行中时新触发立即返回。备选的"排队再跑一轮"会把重复扫描串成链（N 倍 IO），"去抖合并"引入定时器复杂度，均不成比例。代价：扫描启动瞬间的目录变化等下一次触发，对分钟级场景可接受。
- **D2 probeAll 锁内快照拷贝**：探测目标（idx/real/execPath/size）在锁内拷出，探测 goroutine 全程不触碰 `s.last`；结果合并时在锁内二次校验 `s.last` 指针未变（变了则本轮作废）。备选的"全程持锁"会阻塞所有 GUI 调用方数秒。
- **D3 损坏自愈用「备份重建」而非「静默重建」**：corruption 错误时把库文件改名 `history.db.corrupt-<时间戳>`（含 -wal/-shm 侧车），再重建。检测按错误消息匹配（malformed / not a database / corrupt），这是 modernc.org/sqlite 暴露的唯一稳定信号。留档满足本项目"处置透明"原则，用户可自行清理。
- **D4 空包名降级为提示**：`officialCommand` 的 npm 分支在 pkg 为空时返回不可代跑的提示文案（upgrade/uninstall 两侧一致）。降级点放在命令构造层而非调用层，CLI 与 GUI 自动同时生效。
- **D5 动态值先转义再进 `t()`**：`t("upd.versions", { current: esc(res.current), ... })`——文案的 `<b>` 标记保留，动态插值部分已转义。备选的"让 `t()` 转义全部 vars"会破坏确需内嵌标记的既有文案，改动面大。
- **D6 race 检测固化在 test-all.sh**：`go test -race ./gui/` 常驻（GUI 是并发热点，约 +2s），全量 `-race ./...` 在改动并发相关包时按发布清单要求执行。

## Risks / Trade-offs

- [single-flight 丢弃触发后前端无感知] → 进行中那轮的 `scan:done` 自然刷新界面；前端状态机不依赖"每次点击必跑"。
- [损坏检测按错误消息匹配，驱动升级可能措辞变化] → 匹配宽泛（子串），且失败后果仅为不自愈（回到现状），无数据损失放大。
- [corrupt 留档文件占用磁盘] → 单个库文件量级（MB），命名自解释，用户可清理；不做自动过期删除以保持透明。
- [转义遗漏可能仍存在于未审计的插值点] → 已覆盖所有"动态值 + 含标记文案"组合；render.ts 主体早已全量 `esc()`。

## Migration Plan

无数据迁移。历史库损坏自愈在下次打开时自动生效；corrupt 留档无需处理。回滚即还原代码，无持久化副作用。

## Open Questions

（无——全部决策已在实现前确认。）
