# ai-tool-catalog Specification

## Purpose

为 AI 编码工具（agent harness / AI IDE）提供一等公民编目：以 AI 语义分类（会话、索引、捆绑运行时等）归因其磁盘占用，识别已卸载工具的残留数据，并对会话数据库等不可安全清理的对象只报告不处置。

## Requirements

### Requirement: 第一批 AI 工具编目

系统 SHALL 维护 AI 编码工具编目表，第一批 MUST 覆盖：agent（claude、codex、opencode、pi、copilot、zcode、deepseek-harness、antigravity、kimi-code、workbuddy、u1s1、mimocode）、AI IDE（VS Code 及其 AI 扩展伴侣目录；已卸载残留 Trae CN、MarsCode）、已卸载残留（gemini CLI 数据）。agent 类指 agent harness，其桌面端应用（ZCode.app、OpenCode.app、DeepSeek Harness.app、Antigravity.app、Kimi Code.app、WorkBuddy.app、u1s1.app，以及承载 Codex 桌面端的 ChatGPT.app）SHALL 归入同一工具条目。每条编目 MUST 包含：名称、类别（agent / ai-ide）、目录规则（路径模板 + 语义 kind + tier）。目录规则 MUST 采用与现有工具编目一致的跨平台模式。

#### Scenario: 编目工具的目录带语义归因

WHEN 扫描完成且 `~/.codex` 存在 sessions 目录与 packages 目录

THEN 归因结果中 sessions 标注为 session（TierUser），packages 标注为 runtime（TierSafe），均归属 codex 条目

#### Scenario: 编目条目带类别

WHEN 工具列表渲染 zcode、kimi-code、VS Code

THEN 三者分别展示类别 agent、agent、ai-ide

#### Scenario: 桌面端与 CLI 归入同一工具条目

WHEN 扫描发现 brew 安装的 dsh CLI（数据在 `~/.dsh`）与 /Applications 下的 DeepSeek Harness.app（数据在 `~/Library/Application Support/@deepseek-ai`）

THEN 两者归因到 deepseek-harness 同一条目，`~/.dsh/dsh-runtimes` 标注 runtime、`~/.dsh/sessions` 标注 session、`@deepseek-ai` 按桌面端数据归因

#### Scenario: Kimi Code 的 CLI 与桌面端归入同一条目

WHEN 扫描发现 kimi CLI（数据在 `~/.kimi-code`）与 Kimi Code.app（数据在 AppSupport `kimi-code-app`）

THEN 两者归因到 kimi-code 同一条目，bin 标注 runtime（TierSafe）、sessions 标注 session（TierUser）

### Requirement: AI 语义分类

系统 SHALL 以 AI 语义 kind 标注编目工具的数据目录，kind 集合 MUST 至少包含：session（会话/转录）、session-db（会话数据库）、index（可再生代码索引/知识图谱）、runtime（捆绑运行时/npm 包）、plugins、artifacts（agent 产物/exec/上传）、logs、telemetry、checkpoint（文件历史/回滚点）、workspace-data（IDE 工作区存储）。每个 kind MUST 映射到现有 tier：session / checkpoint / workspace-data → TierUser；index / runtime / logs / telemetry → TierSafe。

#### Scenario: 单个会话数据库被识别

WHEN 归因 `~/.local/share/opencode/opencode.db`（SQLite 会话库）

THEN 该项标注为 session-db，归属 opencode，占用以其实际大小呈现

#### Scenario: 可再生索引标注为安全层

WHEN 归因 `~/.marscode/.ckg`（代码知识图谱索引）

THEN 该项标注为 index 且 tier 为 TierSafe（可再生缓存）

#### Scenario: 捆绑运行时标注为安全层

WHEN 归因 `~/.pi/agent/npm` 与 `~/.codex/packages`

THEN 两者均标注为 runtime 且 tier 为 TierSafe

### Requirement: 会话数据库仅报告

系统 SHALL NOT 将 session-db 项列入可清理项（cleanable）。展示 session-db 项时，系统 MUST 说明其为会话历史数据库、暂不支持直接清理。语义级清理（vacuum、按会话删行）不属于本能力。

#### Scenario: 会话数据库不进入清理计划

WHEN 用户对 opencode 执行 `clean --dry-run --all`

THEN 计划中不包含 opencode.db，且详情处说明其为会话数据库

#### Scenario: opencode fork 的会话数据库同样仅报告

WHEN 归因 mimocode（opencode fork）的 `~/.local/share/mimocode/mimocode.db`

THEN 该项标注为 session-db、归属 mimocode 条目，且不进入清理计划

#### Scenario: 树下钻展示说明

WHEN 用户在树下钻中展开到 opencode.db

THEN 该项展示会话数据库说明文案，不提供处置操作

### Requirement: 已卸载 AI 工具的残留归因

当编目中某 AI 工具满足：其编目数据目录存在、且未发现该工具的可执行文件、且系统中无对应 GUI 应用时，系统 SHALL 将其数据目录归因为该工具条目并标注「已卸载」，而 MUST NOT 将这些目录列入无语义的孤儿数据列表。

#### Scenario: 已卸载 IDE 的残留被归因

WHEN MarsCode 应用已卸载但 `~/.marscode`（3.9G，.ckg 索引）存在

THEN 扫描结果包含 MarsCode 条目（标注已卸载），.ckg 标注 index 可回收，且孤儿数据列表不再包含 `.marscode`

#### Scenario: 仍在使用的编目工具不受影响

WHEN claude 可执行文件存在且 `~/.claude` 存在

THEN claude 条目不标注「已卸载」，其目录按语义正常归因

### Requirement: AI IDE 目录归因豁免

对 AI 编目条目的数据目录（含 macOS Application Support 下的 AI IDE 目录，如 `~/Library/Application Support/Trae CN`、`Code`、`kimi-code-app`、`Antigravity`），系统 SHALL 在厂商排除表判定之前完成归因（豁免排除表）。厂商排除表对非 AI 的 GUI 产品照旧生效。

#### Scenario: AI IDE 目录不因 GUI 厂商排除而丢失

WHEN 排除表命中某厂商模式且该厂商旗下存在 AI 编目条目（如 VS Code）

THEN VS Code 的编目数据目录（如 `~/Library/Application Support/Code`）仍被归因并带 workspace-data 等语义标注

#### Scenario: 非 AI 的 GUI 产品照旧排除

WHEN 数据根下存在未认领目录 `~/Library/Application Support/NetSarang`

THEN 该目录仍不列为孤儿数据（厂商排除表生效）
