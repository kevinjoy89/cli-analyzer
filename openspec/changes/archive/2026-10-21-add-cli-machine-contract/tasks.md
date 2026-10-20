## 1. 契约骨架

- [x] 1.1 `internal/cli/output.go`：统一 `emitJSON(kind string, payload any, pretty bool)` —— 注入 `schemaVersion: 1` 与 `kind`，**默认单行紧凑**，`--pretty` 缩进；`emitJSONError` 输出带契约字段的错误文档
- [x] 1.2 `schemaVersion` / `kind` 在**编码时注入**（而非加进 `ScanResult` 结构体）—— 这样 GUI 的 Wails 绑定与缓存文件都不受影响，只有 CLI 输出带契约字段
  - 偏差说明：`CleanReport` **未**扩充 `disposition` / `skipReason` 逐项字段。现有报告是 `deleted` / `skipped` / `errors` 三个字符串数组，`skipped` 里已经带着原因（如 `"<path> (subsumed)"`）。逐项结构化需要改 cleaner 的返回类型，牵动 GUI，本次未做；已在 proposal 的「不在本次范围」之外单列说明。
- [x] 1.3 契约**结构**快照：`internal/cli/testdata/contract/*.txt`（9 份）+ `contract_test.go` 三个测试（结构比对 / 契约字段存在 / 紧凑单行）。只冻结字段名 + JSON 类型，不冻结 `desc` / `keep` 等值。更新方式 `UPDATE_CONTRACT=1 go test ./internal/cli/ -run TestContract`

## 2. 各子命令 --json

- [x] 2.1 `clean --json`（含 `--list --json`、`--dry-run`）：输出 `CleanReport`；`--dry-run` 不写回收站索引（cleaner 在 dryRun 时直接返回，不调用 `trash.Trash`）
- [x] 2.2 `cache --json`（`CacheReport`）；`trash list` / `trash restore` / `trash empty` 全部支持 `--json`（`trash empty --json` 无 `--yes` 时拒绝执行而非等 stdin）
- [x] 2.3 `trends --json`（`TrendsReport`：points + topGrowers + omittedGrowers）
- [x] 2.4 `uninstall --json` 纳入统一契约 + 新增 `--dry-run`；`update check` / `upgrade check|run` 纳入统一契约（改为紧凑 + 带 kind）
- [x] 2.5 `-j` 与 `--json` 在所有子命令等价；`cli.usage` 三语同步

## 3. 脚本化便利

- [x] 3.1 `scan --top N` 与 `trends --top N`；`scan --order=size|cleanable|name`
- [x] 3.2 `scan --baseline <file>`：逐工具增量（`BaselineDiff`），基线不可读或不是 scan 文档时报错退出（非零）
- [x] 3.3 `scan --baseline b.json --fail-on-grow <size>`：任一工具 Cleanable 增长超阈即退出码 1，并列出 `exceeded`
- [x] 3.4 `--dry-run` 与内置回收站配置解耦：`uninstall --dry-run` 只输出计划，不代跑命令、不动文件、不写跟踪事件
- [x] 3.5 （实施中发现并修复）`reorderFlags` 会拆散带值 flag：`scan --baseline b.json --json` 被重排成 `--baseline --json b.json`，基线参数吞掉下一个 flag。新增 `reorderFlagsWithValue`，带值 flag 与其值绑定后一起前移

## 4. update / upgrade 命令分工

- [x] 4.1 新增 `upgrade` 子命令（`internal/cli/upgrade.go`）：`upgrade check <tool>` / `upgrade run <tool> [--yes]`，工具侧逻辑从 `update.go` 迁出
- [x] 4.2 `update` 只保留应用自身更新；`update check <tool>` / `update run <tool>` 保留为别名，stderr 输出 `cli.deprecated` 提示
- [x] 4.3 `kind` 区分 `app-update` / `tool-check` / `tool-upgrade`；`run.go` 分发与 i18n（三语）同步

## 5. 验证

- [x] 5.1 `go test ./...` 全绿；新增 `contract_test.go`、`internal/rules/user_test.go`、`internal/uninstall/tracking_test.go`、`internal/trash/manifest_test.go`
- [x] 5.2 实机验证：
  - `scan --json` → `kind: "scan"`、`schemaVersion: 1`、`omittedTools` 生效
  - `scan --baseline base.json --json` → `kind: "baseline-diff"`，退出码 0
  - `scan --baseline base.json --fail-on-grow 1B --json` → 退出码 1，`exceeded` 非空
  - `clean --all --dry-run --json` → 输出 `CleanReport`，文件系统无变化
  - `cache --json` / `trash list --json` / `trends --json` / `update run <tool> --json` 均输出单行带 kind 的 JSON
- [x] 5.3 `.github/workflows/ci.yml` 增加「Verify JSON contract structure」步骤
- [x] 5.4 README / README.zh-CN 的 CLI 段落补 `--json`、`--top`、`--baseline`、`upgrade` 示例
- [x] 5.5 `clean --dry-run` 合计与 GUI 确认弹窗合计**同口径**（改用对测而非同屏对照）：
  - 把 GUI 内联的去重逻辑抽成 `keepMaximalPaths`（`frontend/src/lib/grouping.ts`），`render.ts` 的 `selectedItems` 与 `formSubtotals` 共用它——原本两处各写一遍
  - `frontend/src/lib/clean-parity.test.ts`（7 例）测**真实实现**，覆盖互不相交 / 父子 / 三层嵌套 / 前缀相似（`/a` vs `/ab`）/ 选择顺序无关 / Windows 反斜杠 / 全选合计 ≤ footprint
  - `internal/cleaner/parity_test.go` 用**同一组场景**跑 Go 侧 `Clean`，两边用例一一对应，任一边改口径都会红
  - 局限：没做到「同一台机器上同时看 GUI 弹窗与 CLI 输出」的逐字节同屏对照（Wails 是原生窗口，内置浏览器碰不到）。改为用共享实现 + 对应用例锁住口径——比肉眼对照更可靠，但**不等于**验证过渲染层最终显示的字符串
