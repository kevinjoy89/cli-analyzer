# Proposal: harden-concurrency-and-data-layer

## Why

v0.4.0 发布后的加固审计发现 GUI 并发边界与数据层存在真实缺陷：`probeAll` 与并发扫描存在数据竞态、扫描多入口并发触发会双记录历史快照（趋势图出现重复点）并产生双倍全量 IO、损坏的 history.db 会让趋势功能永久瘫痪、npm 包名解析为空时会产出注定失败的残缺代跑命令、前端存在一个 XSS 漏网点。这些问题在 `go test -race` 与发布验证流程中均不可见，需要一次性收敛。

## What Changes

- GUI 扫描触发（`Scan` / `ScanIfChanged`）采用丢弃式 single-flight：扫描进行中时新触发直接丢弃（ADR-0001），消除双倍 IO、双份历史快照与多个 `scan:done` 事件。
- `probeAll` 探测目标改为锁内快照拷贝，探测全程不触碰共享 `s.last`，消除与并发扫描的数据竞态。
- history.db 损坏自愈：打开时检测 corruption 错误，把库文件备份为 `history.db.corrupt-<时间戳>`（不静默丢弃）后重建空库。
- npm 包名解析为空时（旧缓存缺 Package 且逆映射表未命中），upgrade/uninstall 代跑命令降级为提示文案，不再产出 `npm update -g ""` 残缺命令。
- 前端动态值（版本号、工具名等来自 GitHub API / 扫描结果的字符串）进入 innerHTML 前统一经 `esc()` 转义，保留文案自身的 `<b>` 标记。
- 升级/卸载代跑 PATH 解析失败时明确报错（新增 `up.resolveFailed` / `un.resolveFailed` 三语文案），不再静默降级为裸名执行。
- `IsBlocked` 循环剥离多层可执行扩展名（`python.exe.exe` 也拦截）。
- 发布验证链路固化 race 检测：`scripts/test-all.sh` 与 docs/release-process.md 加入 `go test -race ./gui/`。

## Capabilities

### New Capabilities

（无——本变更均为对既有行为的加固与收敛，不引入新能力域。）

### Modified Capabilities

- `scan-change-detection`: 新增 GUI 侧扫描触发合并（single-flight）的需求——同一时刻至多一轮扫描在执行，并发触发 SHALL 被丢弃且不产生副作用（无重复历史记录、无多余事件）。
- `usage-trends`: 新增历史库损坏自愈需求——打开检测到损坏时 SHALL 备份原文件后重建空库，趋势功能 MUST NOT 因库损坏永久不可用。
- `tool-upgrade`: 修改代跑命令生成的需求——包名无法解析时 SHALL 降级为不可代跑的提示，代跑命令 MUST NOT 携带空参数；PATH 解析失败时 SHALL 报明确错误。
- `tool-uninstall`: 同上，卸载代跑命令的空包名降级与 PATH 解析失败明确报错。
- `app-update`: 修改更新弹窗渲染需求——来自检查结果的动态字符串（版本号等）渲染进 DOM 前 SHALL 经 HTML 转义。

## Impact

- 代码：`gui/service.go`（single-flight、probeAll、代跑错误）、`internal/history/history.go`（损坏自愈）、`internal/upgrade/command.go` 与 `internal/uninstall/uninstall.go`（空包名降级、IsBlocked）、`internal/i18n/locales/*.json`（新文案键）、`frontend/src/flows.ts`（转义）。
- 测试：`go test -race` 纳入 scripts/test-all.sh；新增损坏自愈、空包名契约、多层扩展名测试。
- 文档：CONTEXT.md（领域术语表）与 docs/adr/0001（single-flight 决策）随本变更一并入库。
- 兼容性：无破坏性变更；事件契约（`scan:done`、`probe:done`）保持不变。
