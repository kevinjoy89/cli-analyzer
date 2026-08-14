## Purpose

发现未被任何工具认领的数据目录（死掉或从未上 PATH 的 CLI 工具残留），经非 CLI 排除体系过滤后展示，仅可移入内置回收站。

## ADDED Requirements

### Requirement: 孤儿数据发现

扫描 SHALL 遍历各平台 CLI 工具主导的数据根（macOS/Linux 的 XDG cache/data/config；Windows 的 AppData/LocalAppData）的顶层目录；未被任何已扫描工具认领的目录 MUST 列为孤儿数据候选，并计算其占用大小。macOS Application Support/Caches/Preferences 等 GUI 应用主导的目录 SHALL 仅用于已认领工具的归因，不作为孤儿数据来源。

#### Scenario: 扫描产生孤儿数据列表

WHEN 扫描完成且数据根下存在未认领目录 `~/.config/old-tool`（占用 120MB）

THEN 扫描结果包含孤儿数据项（路径、大小、所在数据根类型）

#### Scenario: 已认领目录不列为孤儿

WHEN 数据根下目录 `~/.config/gh` 已被工具 `gh` 认领

THEN 该目录不进入孤儿数据列表

#### Scenario: GUI 主导根不作为孤儿来源

WHEN macOS 上存在 `~/Library/Application Support/App Store` 等 GUI 应用数据目录

THEN 该目录不进入孤儿数据列表（Application Support 仅用于已认领工具的归因）

### Requirement: 排除体系应用于孤儿数据

孤儿数据候选 MUST 经过非 CLI 排除体系过滤：系统/OS 目录、本应用自身（回收站根、cli-analyzer 数据）、结构性 GUI 信号目录、命中厂商排除表的目录 MUST NOT 列为孤儿数据。

#### Scenario: 系统目录排除

WHEN 数据根下存在 `%APPDATA%\Local\Microsoft\Edge` 等系统目录

THEN 不列为孤儿数据

#### Scenario: 厂商排除表目录排除

WHEN 数据根下存在 `%APPDATA%\NetSarang Computer` 且排除表含 `netsarang`

THEN 不列为孤儿数据

### Requirement: 孤儿数据展示与处置

GUI SHALL 在工具面板以"未认领数据"标签页展示（与工具列表并列的独立视图，顶部汇总卡展示总占用），每项 MUST 显示路径、占用大小，并按数据根分组；支持全选与批量移入内置回收站；孤儿数据 SHALL 为 USER 级，唯一处置操作 MUST 是移入内置回收站（可恢复），MUST NOT 提供永久删除路径。

#### Scenario: GUI 展示孤儿数据

WHEN 扫描结果包含孤儿数据且用户在 GUI 打开未认领数据标签页

THEN 每项展示路径、大小，按数据根分组

#### Scenario: 移入回收站

WHEN 用户对孤儿数据项执行处置

THEN 该目录移入内置回收站，用户可恢复，且无永久删除选项

### Requirement: CLI 契约

CLI `scan` 输出（含 `--json`）SHALL 包含孤儿数据字段；GUI 与 CLI MUST 使用同一过滤逻辑，结果一致。

#### Scenario: CLI 输出孤儿数据

WHEN 执行 `cli-analyzer scan --json`

THEN 输出包含孤儿数据数组（路径、大小、数据根类型）
