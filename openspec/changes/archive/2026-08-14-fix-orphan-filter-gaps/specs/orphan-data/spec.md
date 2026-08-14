## MODIFIED Requirements

### Requirement: 孤儿数据发现

扫描 SHALL 遍历各平台 CLI 工具主导的数据根（macOS/Linux 的 XDG cache/data/config；Windows 的 AppData/LocalAppData）的顶层目录；未被任何已扫描工具认领的目录 MUST 列为孤儿数据候选，并计算其占用大小。认领集合 SHALL 包含工具的别名集（工具名与别名）、其数据目录（dataDirs）与可清理项（cleanables）在各数据根下的顶层目录名，匹配 SHALL 大小写不敏感（Windows 上 PATH 名 `claude` 与数据目录名 `Claude` 视为同一认领）。孤儿候选 MUST 仅限目录：数据根下的普通文件（`.DS_Store`、`.localized` 等）与指向文件的符号链接 MUST NOT 列为孤儿，指向目录的符号链接 SHALL 仍为合法候选。macOS Application Support/Caches/Preferences 等 GUI 应用主导的目录 SHALL 仅用于已认领工具的归因，不作为孤儿数据来源。

#### Scenario: 扫描产生孤儿数据列表

WHEN 扫描完成且数据根下存在未认领目录 `~/.config/old-tool`（占用 120MB）

THEN 扫描结果包含孤儿数据项（路径、大小、所在数据根类型）

#### Scenario: 已认领目录不列为孤儿

WHEN 数据根下目录 `~/.config/gh` 已被工具 `gh` 认领

THEN 该目录不进入孤儿数据列表

#### Scenario: 大小写不同的目录仍被认领

WHEN PATH 上工具名为 `claude` 且数据根下存在目录 `~/.config/Claude`

THEN 该目录被认领，不进入孤儿数据列表

#### Scenario: 数据目录名异于工具名仍被认领

WHEN 工具 opencode 的数据目录规则含 `~/.config/oh-my-opencode`

THEN 该目录被认领，不进入孤儿数据列表

#### Scenario: 普通文件不列为孤儿

WHEN 数据根下存在 `.DS_Store` 文件（macOS 目录元数据）或指向文件的符号链接

THEN 不列为孤儿数据（仅目录是孤儿候选）

#### Scenario: GUI 主导根不作为孤儿来源

WHEN macOS 上存在 `~/Library/Application Support/App Store` 等 GUI 应用数据目录

THEN 该目录不进入孤儿数据列表（Application Support 仅用于已认领工具的归因）

### Requirement: 排除体系应用于孤儿数据

孤儿数据候选 MUST 经过非 CLI 排除体系过滤：系统/OS 目录、本应用自身（回收站根、cli-analyzer 数据）、结构性 GUI 信号目录、系统/共享结构目录（跨平台共享运行时目录如 `.mono`/`configstore`/`man`，Windows 系统组件目录如 `%LOCALAPPDATA%\Programs`/`Temp`/`CrashDumps`）、命中厂商排除表的目录 MUST NOT 列为孤儿数据。

#### Scenario: 系统目录排除

WHEN 数据根下存在 `%APPDATA%\Local\Microsoft\Edge` 等系统目录

THEN 不列为孤儿数据

#### Scenario: 共享运行时目录排除

WHEN 数据根下存在 `~/.config/.mono`、`~/.config/configstore`、`~/.local/share/man`、`~/.config/.isolated-storage`

THEN 不列为孤儿数据（共享基础设施，非任何工具的数据残留）

#### Scenario: Windows 系统组件目录排除

WHEN `%LOCALAPPDATA%` 下存在 `Programs`（GUI 应用安装根）、`Temp`、`CrashDumps`、`D3DSCache`、`lxss`、`Comms`

THEN 不列为孤儿数据

#### Scenario: 厂商排除表目录排除

WHEN 数据根下存在 `%APPDATA%\NetSarang Computer` 且排除表含 `netsarang`

THEN 不列为孤儿数据

#### Scenario: GUI 产品目录排除

WHEN 数据根下存在 `~/.config/iterm2`、`~/.config/raycast`、`%APPDATA%\Mozilla`、`%APPDATA%\dingtalk` 且排除表含对应产品模式

THEN 不列为孤儿数据（已安装 GUI 应用的数据目录）
