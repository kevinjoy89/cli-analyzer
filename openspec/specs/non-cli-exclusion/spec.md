# non-cli-exclusion Specification

## Purpose

识别并排除非 CLI（GUI 应用及其命令行伴侣、数据目录），确保应用只管理 CLI 工具及其残留。

## Requirements

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

### Requirement: 纯 CLI 产品例外

厂商模式命中的目录中，属于该厂商独立安装的纯 CLI 产品 SHALL 可通过例外白名单保留；GUI 应用自带的命令行伴侣 MUST NOT 列入例外。判据 MUST 为：该 CLI 是否作为独立产品安装、拥有自己的安装目录、不服务于任何 GUI 应用。

#### Scenario: 例外白名单保留纯 CLI 产品

WHEN 排除表含 `amazon` 且例外列表含 `aws`，PATH 存在 `C:\Program Files\Amazon\AWSCLIV2\aws.exe`

THEN `aws.exe` 仍作为工具扫描

#### Scenario: 命令行伴侣不例外

WHEN 排除表含 `netsarang` 且例外列表为空，PATH 存在 `C:\Program Files (x86)\NetSarang\Xshell 8\xftpcl.exe`

THEN `xftpcl.exe` 不作为工具扫描

### Requirement: 确定性过滤

孤儿数据与可执行文件过滤 MUST 仅使用确定性规则：系统/OS 目录、本应用自身目录、结构性 GUI 信号（macOS `~/Library/Containers/<bundle-id>`、Windows `%LOCALAPPDATA%\Packages\<family>_<hash>`）、系统/共享结构目录、应用更新器目录、已安装应用交叉验证（Windows）、厂商排除表。系统/共享结构目录 SHALL 包含跨平台共享运行时目录（`.mono`、`configstore`、`simple-update-notifier`、`IsolatedStorage`、`.isolated-storage`、`man`、`go-build`（Go 工具链构建缓存））与 Windows 系统组件目录（`%LOCALAPPDATA%\Programs`（GUI 应用安装根）、`Temp`、`CrashDumps`、`D3DSCache`、`lxss`（WSL 根）、`Comms`、`ConnectedDevicesPlatform`（设备同步平台）、`PlaceholderTileLogoFolder`（磁贴占位）及 `%APPDATA%\VoiceAccess`（Windows 语音访问组件））。应用更新器目录 SHALL 指目录名（小写）以 `-updater` 结尾的目录（Squirrel/electron-updater 约定 `<App>-updater`，如 `tabby-updater`、`termius-updater`）——GUI 应用自动更新暂存目录，非任何工具的数据残留。已安装应用交叉验证 SHALL 仅在 Windows 生效：未认领数据根目录名与已安装应用证据（注册表卸载项 DisplayName——HKCU 与 HKLM 的 32/64 位视图；开始菜单快捷方式基名——用户与公共两级，仅取文件名不解析链接）交叉比对；匹配规则 MUST 为：名称精确相等；目录名是应用名前缀且目录名长度 ≥ 5；或应用名是目录名前缀、应用名长度 ≥ 5 且目录名余段以 `-` 开头。长度门槛与 `-` 后缀约束 MUST 防止短名（`pc`、`git`、`code`）与普通复合名（`gitea`、`notepadnext`）误伤。交叉验证基于安装证据（注册表/快捷方式只读查询），不是名称形态启发式。MUST NOT 使用名称形态启发式（如大小写、空格、公司命名风格）做过滤。

#### Scenario: 结构性 GUI 信号目录排除

WHEN 数据根下存在 `~/Library/Containers/com.apple.Safari`（bundle-id 形态）

THEN 该目录不列为孤儿数据

#### Scenario: 系统/共享结构目录排除

WHEN 数据根下存在 `~/.config/.mono`（Mono 运行时共享状态）、`~/.config/configstore`（node 生态共享配置）、`%LOCALAPPDATA%\Programs\SomeApp`（GUI 应用安装根）、`%LOCALAPPDATA%\Temp`、`%LOCALAPPDATA%\CrashDumps`、`%LOCALAPPDATA%\ConnectedDevicesPlatform`、`%APPDATA%\VoiceAccess`

THEN 这些目录不列为孤儿数据

#### Scenario: 更新器目录排除

WHEN 数据根下存在未认领目录 `%LOCALAPPDATA%\tabby-updater`（小写以 `-updater` 结尾）

THEN 该目录不列为孤儿数据（GUI 应用自动更新暂存）

#### Scenario: 已安装应用数据目录排除

WHEN Windows 上未认领目录 `%APPDATA%\PeaZip` 且注册表卸载项含 DisplayName `PeaZip 11.0.0 (WIN64)`

THEN 该目录不列为孤儿数据（目录名是应用名前缀）

#### Scenario: 应用名前缀且余段带连字符排除

WHEN 数据根下存在未认领目录 `%APPDATA%\腾讯电脑管家-全局信息` 且注册表卸载项含 DisplayName `腾讯电脑管家`

THEN 该目录不列为孤儿数据（应用名是目录名前缀且余段 `-全局信息` 以 `-` 开头）

#### Scenario: 短名目录不误伤

WHEN 未认领目录为 `pc`（2 字符）或 `gitea` 而注册表含应用名 `git`（3 字符，且 `gitea` 余段不以 `-` 开头）

THEN 该目录仍为孤儿数据候选（长度门槛与 `-` 后缀约束生效）

#### Scenario: 无启发式过滤

WHEN 存在未认领目录 `~/.config/WeirdName`（标题大小写、含空格或不符合任何排除规则）

THEN 该目录正常列为孤儿数据（仅凭名称形态不得排除）
