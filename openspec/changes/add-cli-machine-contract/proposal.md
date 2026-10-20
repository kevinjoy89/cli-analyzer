## Why

CLI 与 GUI 共用同一套领域模型（`ScanResult` 是唯一 JSON 契约），但机器可读输出是**零散铺上的**：`scan`（`-j`/`--json`，indent 输出）、`update`、`uninstall` 各有一套，`clean` / `cache` / `trash` / `trends` 完全没有。已 JSON 化的三处也互不一致：`scan -j` 缩进、`update check -j` 缩进、错误分支却是紧凑；没有 `kind` 字段，消费方只能靠猜字段判断报告类型。结果：

- 想脚本化的人只能去解析 GUI 或抓 `scan` 的 JSON 再自己算，而「处置了什么、回收了多少字节」这类事实只有文本输出。
- 契约没有版本号，字段一旦调整，下游消费者无从判断兼容性。
- 没有「两次扫描对比」的出口，趋势图只存在于 GUI，CI 里无法做「这次改动让某工具占用暴涨」的门禁。

目标：让 CLI 成为一等公民——每个子命令都有稳定的机器可读输出，且该输出可被 CI 用来防回退。

## What Changes

- **统一输出契约**：所有子命令支持 `--json`（等价短选项 `-j`，与现有 `scan` 一致），**默认输出单行紧凑 JSON**（缩进输出保留为 `--pretty`）；默认仍是人类可读文本。改造面为 `clean` / `cache` / `trash` / `trends` 四个**尚无 JSON**的子命令，外加把已有三处（`scan` / `update` / `uninstall`）纳入同一编码函数。
  - 注：现有 `scan -j` 与 `update check -j` 输出缩进，改为紧凑是行为变更；正经消费方式是 `jq`（对两者无差别），且这是 `schemaVersion: 1` 的首次发布——契约刚立时统一口径最便宜。
- **契约版本**：新增顶层字段 `schemaVersion`（整数，本次定为 `1`）与 `kind`（标识这份 JSON 是哪种报告：`scan` / `clean` / `trash-list` / `trends` / `uninstall` / `app-update` / `tool-upgrade` / `tool-check`），使消费方可判别类型与兼容性。
- **`clean` 报告**：`CleanReport` 已有 `DryRun` 字段——补全 JSON 契约（每条处置项的路径、Kind、字节、去向：built-in trash / system trash / purge / skipped，以及跳过原因）。
- **`--dry-run` 覆盖到 `uninstall`**：`clean --dry-run` 已存在，缺的是 JSON 报告与「预演不碰内置回收站索引」的保证；`uninstall` 需要新增 `--dry-run`。预演永远不产生副作用，与回收站配置解耦。
- **`--top N`**：`scan` 与 `trends` 支持只输出前 N 个工具（按 Footprint 或 Cleanable 排序），避免脚本里截断大 JSON。
- **对比子命令**：`scan --baseline <file>` 输出与给定基线快照的逐工具增量（字节变化、新增/消失的工具）。
- **CI 门禁信号**：`--baseline` 配 `--fail-on-grow <size>`：任一工具的 **Cleanable** 增长超过阈值即退出码非 0。指标只取 Cleanable——Footprint 会因新装工具正常增长，误报率高。阈值判断留在 CLI 里（退出码是 CI 唯一认的信号），不让用户写 `jq` 表达式。
- **命令名与术语对齐**：`CONTEXT.md` 规定 Upgrade = 工具升级、Update = 应用自身更新，并明确禁止混用；但现有 `update check`（应用）/ `update check <tool>`（工具检查）/ `update run <tool>`（工具升级）一个动词干了三件事。改为：新增 `upgrade` 子命令承载工具侧（`upgrade check <tool>` / `upgrade run <tool>`），`update` 只保留应用自身更新；旧的 `update check <tool>` / `update run <tool>` 保留为**别名**并输出 deprecation 提示到 stderr。JSON 侧用 `kind` 区分 `app-update` / `tool-check` / `tool-upgrade`。
- **CI 门禁**：`.github/workflows/ci.yml` 增加一步契约快照测试。快照**只冻结结构与类型**（字段名 + JSON 类型 + 是否可选），不冻结 `desc` / `keep` / `sub` 等文案与内容型字段的值——契约测试要防的是「字段被删/改名/类型变了」，不是「文案改了」。

## Capabilities

### New Capabilities
- `cli-machine-contract`: CLI 每个子命令的机器可读输出契约（`schemaVersion` / `kind` / 字段冻结范围）、`--json` 与 `--top` 的语义、`scan --baseline` + `--fail-on-grow` 的对比与门禁口径、`update`/`upgrade` 的命令分工，以及契约快照测试的要求。

### Modified Capabilities
- 无。`tool-attribution` / `trash-recycle-bin` / `usage-trends` 的**领域口径不变**，本变更只增加输出通道，不改变任何字节计算规则。

## Impact

- `internal/cli/`：`output.go`（统一 JSON 编码与 `schemaVersion`/`kind` 注入、`--pretty`）、`scan.go`（`--top`、`--baseline`、`--fail-on-grow`）、`clean.go`、`cache.go`、`trash.go`、`trends.go`、`uninstall.go`（`--dry-run`）、`update.go`（拆分出工具侧逻辑）、新增 `upgrade.go`、`run.go`（命令分发与帮助文本）。
- `internal/scanner/types.go`：`ScanResult` 增 `schemaVersion` / `kind`；`CleanReport` 补字段。**不删不改现有字段名**（见 ADR 0002 的兼容策略）。
- `internal/cleaner/`：把「去向」与「跳过原因」暴露到 `CleanReport`（当前只在日志/返回里）。
- `frontend/`：无改动（GUI 不消费 CLI 文本输出；`ScanResult` 新增字段对 Wails 绑定透明）。
- 新增 `internal/cli/testdata/contract/*.json` 结构快照（字段名 + 类型 + 可选性，值不冻结）；`go test ./internal/cli/...` 覆盖。
- 文档：`README.md` / `README.zh-CN.md` 的 CLI 段落补 `--json`、`--top`、`--baseline`、`upgrade` 示例；`CONTEXT.md` 的 Upgrade/Update 条目下补一句命令面映射。

## 不在本次范围

- 规则包外置（用户自定义 `rules.d/*.json`）——牵动 `Installer` / 编目条目的定义，单独立项。
- CSV / Markdown 输出格式——先只做 JSON；`--format` 参数留待有真实消费者再加。
- 对 `ScanResult` 字段的**重命名**（如 `cleanableBytes`）——只加不删，重命名留给下一个大版本。
