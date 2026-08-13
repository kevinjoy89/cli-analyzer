# non-cli-exclusion Specification

## Purpose

识别并排除非 CLI（GUI 应用及其命令行伴侣、数据目录），确保应用只管理 CLI 工具及其残留。

## Requirements

### Requirement: 厂商排除表

应用 SHALL 维护已知 GUI 产品厂商的排除规则表。每条规则 MUST 包含厂商模式与可选例外列表。匹配 MUST 采用路径片段精确匹配（大小写不敏感）：路径任一目录片段与模式相等即命中；MUST NOT 使用子串匹配（避免 `code` 命中 `opencode`）。排除表 MUST 统一应用于可执行文件发现与数据目录归因两个环节。

#### Scenario: 命中厂商模式的安装目录整体排除

WHEN 扫描发现 PATH 目录 `C:\Program Files (x86)\NetSarang\Xshell 8\` 且排除表包含模式 `netsarang`

THEN 该目录下所有可执行文件（含 Xshell.exe、installanchorservice.exe、xftpcl.exe 等）均不作为工具扫描

#### Scenario: 数据目录命中厂商模式

WHEN 数据根下存在未认领目录 `%APPDATA%\NetSarang Computer\7\Xshell`

THEN 该目录不列为孤儿数据

#### Scenario: 片段匹配不误伤相似名称

WHEN 存在目录 `~/.config/opencode` 而排除表模式包含 `code`

THEN 该目录不受影响（片段 `opencode` 与模式 `code` 不相等）

### Requirement: 纯 CLI 产品例外

厂商模式命中的目录中，属于该厂商独立安装的纯 CLI 产品 SHALL 可通过例外白名单保留；GUI 应用自带的命令行伴侣 MUST NOT 列入例外。判据 MUST 为：该 CLI 是否作为独立产品安装、拥有自己的安装目录、不服务于任何 GUI 应用。

#### Scenario: 例外白名单保留纯 CLI 产品

WHEN 排除表含 `amazon` 且例外列表含 `aws`，PATH 存在 `C:\Program Files\Amazon\AWSCLIV2\aws.exe`

THEN `aws.exe` 仍作为工具扫描

#### Scenario: 命令行伴侣不例外

WHEN 排除表含 `netsarang` 且例外列表为空，PATH 存在 `C:\Program Files (x86)\NetSarang\Xshell 8\xftpcl.exe`

THEN `xftpcl.exe` 不作为工具扫描

### Requirement: 确定性过滤

孤儿数据与可执行文件过滤 MUST 仅使用确定性规则：系统/OS 目录、本应用自身目录、结构性 GUI 信号（macOS `~/Library/Containers/<bundle-id>`、Windows `%LOCALAPPDATA%\Packages\<family>_<hash>`）、厂商排除表。MUST NOT 使用名称形态启发式（如大小写、空格、公司命名风格）做过滤。

#### Scenario: 结构性 GUI 信号目录排除

WHEN 数据根下存在 `~/Library/Containers/com.apple.Safari`（bundle-id 形态）

THEN 该目录不列为孤儿数据

#### Scenario: 无启发式过滤

WHEN 存在未认领目录 `~/.config/WeirdName`（标题大小写、含空格或不符合任何排除规则）

THEN 该目录正常列为孤儿数据（仅凭名称形态不得排除）
