## ADDED Requirements

### Requirement: 每个子命令都提供机器可读输出

CLI 的每个子命令（`scan` / `clean` / `cache` / `trash` / `trends` / `update` / `uninstall`）SHALL 支持 `--json`（与短选项 `-j` 等价），在 stdout 输出**单行紧凑 JSON**，人类可读文本与诊断信息 SHALL 走 stderr 或默认（无 `--json`）模式。`--json` 模式下 stdout SHALL 只含一份 JSON 文档，便于 `jq` 与管道消费。

#### Scenario: scan 的 JSON 输出

WHEN 执行 `cli-analyzer scan -j`

THEN stdout 是一份完整 JSON 文档，可被 `jq` 解析，且不含任何进度或提示文字

#### Scenario: clean 的 JSON 输出

WHEN 执行 `cli-analyzer clean --json`

THEN stdout 输出一份清理报告 JSON，包含每条处置项的路径、Kind、字节数与去向

#### Scenario: 短选项等价

WHEN 分别执行 `cli-analyzer trends -j` 与 `cli-analyzer trends --json`

THEN 两者输出完全一致

### Requirement: 机器可读输出携带契约版本与报告类型

每份 JSON 文档 SHALL 含顶层字段 `schemaVersion`（整数）与 `kind`（报告类型标识，取值 `scan` / `clean` / `trash-list` / `trends` / `uninstall` / `update`）。消费方 SHALL 能仅凭这两个字段判断文档类型与兼容性，无需试探字段。契约 SHALL 只增不删：已发布字段不得重命名或移除，破坏性调整必须提升 `schemaVersion`。

#### Scenario: 判别报告类型

WHEN 消费方拿到一份 cli-analyzer 的 JSON

THEN 读取顶层 `kind` 即可确定它属于哪类报告，读取 `schemaVersion` 即可判断是否兼容

#### Scenario: 字段只增不删

WHEN 后续版本需要表达新信息

THEN 只新增可选字段，不重命名或移除既有字段；若必须破坏性调整，则提升 `schemaVersion`

### Requirement: 处置类命令支持无副作用的预演

`clean` 与 `uninstall` SHALL 支持 `--dry-run`：输出「将被处置的条目、合计字节、各自去向」，且 SHALL NOT 移动、删除任何文件，SHALL NOT 写入内置回收站索引。预演结果 SHALL 与实际执行时的条目集合同口径（同一套极大路径去重与父项包含跳过规则）。

#### Scenario: 预演不动文件

WHEN 执行 `cli-analyzer clean --dry-run --json`

THEN 输出与真实执行同口径的条目清单与合计，文件系统与内置回收站状态均无变化

#### Scenario: 预演与执行同口径

WHEN 选中集合中存在父子关系的两个可处置项

THEN 预演与真实执行都只计父项一次，并在报告中标注子项被父项包含而跳过

### Requirement: 扫描结果可与基线对比

`scan --baseline <file>` SHALL 输出本次扫描相对基线快照的逐工具增量：每个工具的占用变化、可清理变化，以及新增与消失的工具。基线文件格式 SHALL 与 `scan --json` 的输出兼容；基线不可读或版本不兼容时 SHALL 报错退出而不是静默忽略。

#### Scenario: 输出增量

WHEN 执行 `cli-analyzer scan --baseline baseline.json`

THEN 输出只包含发生变化的工具及其字节增减，以及新增 / 消失的工具清单

#### Scenario: 基线损坏

WHEN 基线文件不是合法的 scan JSON

THEN 命令以非零退出码结束并给出明确错误，不输出看似正常的增量报告

### Requirement: 增长超阈可作为 CI 门禁

`scan --baseline <file> --fail-on-grow <size>` SHALL 在任一工具的 **Cleanable** 增长超过给定阈值时以非零退出码结束，且 SHALL 在输出中标明是哪个工具、增长了多少。门禁指标 SHALL 只取 Cleanable（Footprint 会因新装工具正常增长，不适合作门禁）。未给 `--fail-on-grow` 时 SHALL 恒为 0，只作报告。

#### Scenario: 增长超阈

WHEN 某工具 Cleanable 相对基线增长 600MB 且阈值为 500MB

THEN 命令以非零退出码结束，并指出该工具及其增长量

#### Scenario: 只报告不门禁

WHEN 执行 `scan --baseline b.json` 而未给 `--fail-on-grow`

THEN 退出码为 0，只输出增量报告

### Requirement: update 与 upgrade 各司其职

`upgrade` SHALL 承载工具侧动作（`upgrade check <tool>` 检查新版本、`upgrade run <tool>` 代跑升级命令）；`update` SHALL 只承载应用自身更新。旧的 `update check <tool>` 与 `update run <tool>` SHALL 继续可用并行为不变，但 SHALL 向 stderr 输出 deprecation 提示。两类动作的 JSON 报告 SHALL 用不同的 `kind` 区分（`app-update` / `tool-check` / `tool-upgrade`）。

#### Scenario: 工具升级走 upgrade

WHEN 执行 `cli-analyzer upgrade run ripgrep`

THEN 代跑 ripgrep 的官方升级命令，行为与旧 `update run ripgrep` 一致

#### Scenario: 旧写法仍可用但提示

WHEN 执行 `cli-analyzer update run ripgrep`

THEN 功能照旧，同时 stderr 出现指向 `upgrade run` 的 deprecation 提示

#### Scenario: 应用更新不碰工具

WHEN 执行 `cli-analyzer update check`（无工具参数）

THEN 只检查 cli-analyzer 自身版本，JSON `kind` 为 `app-update`

### Requirement: 大结果可按量截断

`scan` 与 `trends` SHALL 支持 `--top N`：只输出按指定指标（默认 Footprint）排序的前 N 个工具，并在输出中标明被截断的数量，使脚本无需自行解析大文档。

#### Scenario: 只取前几名

WHEN 执行 `cli-analyzer scan --top 10 --json`

THEN 输出的 tools 数组最多 10 项，按 Footprint 降序，并附带被省略的工具数

### Requirement: 契约快照只冻结结构与类型

契约测试 SHALL 冻结每个 JSON 报告的**结构**：字段名、JSON 类型、是否可选；SHALL NOT 冻结 `desc` / `keep` / `sub` 等内容型字段的**值**。`schemaVersion` 或 `kind` 缺失、字段被重命名或移除、字段类型变更时测试 SHALL 失败；仅文案变化时 SHALL NOT 失败。

#### Scenario: 文案变化不触发失败

WHEN 只修改了某条可处置项的 `desc` 文案

THEN 契约快照测试仍通过

#### Scenario: 字段被删触发失败

WHEN 从 `ScanResult` 移除或重命名一个已发布字段

THEN 契约快照测试失败，要求显式提升 `schemaVersion`
