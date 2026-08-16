## MODIFIED Requirements

### Requirement: 确定性过滤

孤儿数据与可执行文件过滤 MUST 仅使用确定性规则：系统/OS 目录、本应用自身目录、结构性 GUI 信号（macOS `~/Library/Containers/<bundle-id>`、Windows `%LOCALAPPDATA%\Packages\<family>_<hash>`）、系统/共享结构目录、应用更新器目录、已安装应用交叉验证（Windows）、厂商排除表。系统/共享结构目录 SHALL 包含跨平台共享运行时目录（`.mono`、`configstore`、`simple-update-notifier`、`IsolatedStorage`、`.isolated-storage`、`man`、`go-build`（Go 工具链构建缓存））与 Windows 系统组件目录（`%LOCALAPPDATA%\Programs`（GUI 应用安装根）、`Temp`、`CrashDumps`、`D3DSCache`、`lxss`（WSL 根）、`Comms`、`ConnectedDevicesPlatform`（设备同步平台）、`PlaceholderTileLogoFolder`（磁贴占位）、以及 `%APPDATA%\VoiceAccess`（Windows 语音访问组件））。应用更新器目录 SHALL 指目录名（小写）以 `-updater` 结尾的目录（Squirrel/electron-updater 约定 `<App>-updater`，如 `tabby-updater`、`termius-updater`）——GUI 应用自动更新暂存，不是任何工具的数据残留。已安装应用交叉验证 SHALL 仅在 Windows 生效：未认领数据根目录名与已安装应用证据（注册表卸载项 DisplayName——HKCU 与 HKLM 的 32/64 位视图，及开始菜单快捷方式基名——用户与公共两级）交叉比对；匹配规则 MUST 为：名称精确相等；目录名是应用名前缀且目录名长度 ≥ 5；或应用名是目录名前缀、应用名长度 ≥ 5 且目录名余段以 `-` 开头。长度门槛与 `-` 后缀约束 MUST 防止短名（`pc`、`git`、`code`）与普通复合名（`gitea`、`notepadnext`）误伤。交叉验证是安装证据（注册表/快捷方式只读查询），不是名称形态启发式。MUST NOT 使用名称形态启发式（如大小写、空格、公司命名风格）做过滤。

#### Scenario: 已安装应用数据目录排除

WHEN Windows 上未认领目录 `%APPDATA%\PeaZip` 且注册表卸载项含 DisplayName `PeaZip 11.0.0 (WIN64)`

THEN 该目录不列为孤儿数据（目录名是应用名前缀）

#### Scenario: 更新器目录排除

WHEN 数据根下存在未认领目录 `%LOCALAPPDATA%\tabby-updater`（小写以 `-updater` 结尾）

THEN 该目录不列为孤儿数据（GUI 应用自动更新暂存）

#### Scenario: 应用名前缀且余段带连字符排除

WHEN 数据根下存在未认领目录 `%APPDATA%\腾讯电脑管家-全局信息` 且注册表卸载项含 DisplayName `腾讯电脑管家`

THEN 该目录不列为孤儿数据（应用名是目录名前缀且余段 `-全局信息` 以 `-` 开头）

#### Scenario: 短名目录不误伤

WHEN 未认领目录为 `pc`（2 字符）或 `gitea` 而注册表含应用名 `git`（3 字符，`gitea` 余段不以 `-` 开头）

THEN 该目录仍为孤儿数据候选（长度门槛与 `-` 后缀约束生效）
