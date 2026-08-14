## MODIFIED Requirements

### Requirement: 厂商排除表

应用 SHALL 维护已知 GUI 产品厂商的排除规则表。每条规则 MUST 包含厂商模式与可选例外列表。匹配 MUST 采用路径片段精确匹配（大小写不敏感）：路径任一目录片段与模式相等即命中；MUST NOT 使用子串匹配（避免 `code` 命中 `opencode`）。排除表 MUST 统一应用于可执行文件发现与数据目录归因两个环节；产品级短名条目（DataOnly）MUST 仅应用于数据目录归因环节（孤儿过滤），不得拦截可执行文件发现——短名（如 `code`、`amd`）可能是用户的真实 PATH 目录，拦截会误伤其中的工具。代理/网络类 CLI 核心（clash、v2ray、mihomo 等独立二进制）MUST NOT 列入排除表，仅其 GUI 客户端（clash-verge、v2rayN 等）可列入。

#### Scenario: 命中厂商模式的安装目录整体排除

WHEN 扫描发现 PATH 目录 `C:\Program Files (x86)\NetSarang\Xshell 8\` 且排除表包含模式 `netsarang`

THEN 该目录下所有可执行文件（含 Xshell.exe、installanchorservice.exe、xftpcl.exe 等）均不作为工具扫描

#### Scenario: 数据目录命中厂商模式

WHEN 数据根下存在未认领目录 `%APPDATA%\NetSarang Computer\7\Xshell`

THEN 该目录不列为孤儿数据

#### Scenario: 片段匹配不误伤相似名称

WHEN 存在目录 `~/.config/opencode` 而排除表模式包含 `code`

THEN 该目录不受影响（片段 `opencode` 与模式 `code` 不相等）

#### Scenario: DataOnly 条目不拦截 exe 发现

WHEN 排除表含 DataOnly 模式 `iterm2` 且 PATH 存在用户自建目录 `~/bin/iterm2` 内含可执行脚本

THEN 该目录下的可执行文件仍作为工具扫描，但数据根下未认领目录 `~/.config/iterm2` 不列为孤儿数据

#### Scenario: CLI 代理核心不排除

WHEN 数据根下存在未认领目录 `~/.config/mihomo`（mihomo 核心为独立 CLI 二进制）

THEN 该目录仍为孤儿数据候选（仅 GUI 客户端目录 clash-verge/v2rayN 等被排除）

### Requirement: 确定性过滤

孤儿数据与可执行文件过滤 MUST 仅使用确定性规则：系统/OS 目录、本应用自身目录、结构性 GUI 信号（macOS `~/Library/Containers/<bundle-id>`、Windows `%LOCALAPPDATA%\Packages\<family>_<hash>`）、系统/共享结构目录、厂商排除表。系统/共享结构目录 SHALL 包含跨平台共享运行时目录（`.mono`、`configstore`、`simple-update-notifier`、`IsolatedStorage`、`.isolated-storage`、`man`）与 Windows 系统组件目录（`%LOCALAPPDATA%\Programs`（GUI 应用安装根）、`Temp`、`CrashDumps`、`D3DSCache`、`lxss`（WSL 根）、`Comms`）。MUST NOT 使用名称形态启发式（如大小写、空格、公司命名风格）做过滤。

#### Scenario: 结构性 GUI 信号目录排除

WHEN 数据根下存在 `~/Library/Containers/com.apple.Safari`（bundle-id 形态）

THEN 该目录不列为孤儿数据

#### Scenario: 系统/共享结构目录排除

WHEN 数据根下存在 `~/.config/.mono`（Mono 运行时共享状态）、`~/.config/configstore`（node 生态共享配置）、`%LOCALAPPDATA%\Programs\SomeApp`（GUI 应用安装根）、`%LOCALAPPDATA%\Temp`、`%LOCALAPPDATA%\CrashDumps`

THEN 这些目录不列为孤儿数据

#### Scenario: 无启发式过滤

WHEN 存在未认领目录 `~/.config/WeirdName`（标题大小写、含空格或不符合任何排除规则）

THEN 该目录正常列为孤儿数据（仅凭名称形态不得排除）
