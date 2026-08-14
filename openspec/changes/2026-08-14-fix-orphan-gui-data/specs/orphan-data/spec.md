## MODIFIED Requirements

### Requirement: 排除体系应用于孤儿数据

孤儿数据候选 MUST 经过非 CLI 排除体系过滤：系统/OS 目录、本应用自身目录（含本应用可执行文件同名目录——Wails 产品名 `CLI Analyzer` 的 exe 出现在 `%APPDATA%` 顶层时同样视为自身）、结构性 GUI 信号目录、系统/共享结构目录（跨平台共享运行时目录如 `.mono`/`configstore`/`go-build`/`man`，Windows 系统组件目录如 `%LOCALAPPDATA%\Programs`/`Temp`/`CrashDumps`/`ConnectedDevicesPlatform`/`PlaceholderTileLogoFolder`、`%APPDATA%\VoiceAccess`）、应用更新器目录（`<App>-updater` 形态）、Windows 已安装应用交叉验证（注册表卸载项 DisplayName / 开始菜单快捷方式名匹配）、命中厂商排除表的目录 MUST NOT 列为孤儿数据。

#### Scenario: 更新器目录排除

WHEN 数据根下存在未认领目录 `%LOCALAPPDATA%\tabby-updater`、`%LOCALAPPDATA%\termius-updater`

THEN 不列为孤儿数据（GUI 应用自动更新暂存）

#### Scenario: 已安装应用数据目录排除

WHEN Windows 上未认领目录 `%APPDATA%\PeaZip` 与已安装应用 DisplayName `PeaZip 11.0.0 (WIN64)` 前缀匹配

THEN 不列为孤儿数据

#### Scenario: 本应用可执行文件同名目录排除

WHEN 数据根下存在未认领目录 `%APPDATA%\CLI Analyzer.exe`（本应用可执行文件基名）

THEN 不列为孤儿数据（自身目录）
