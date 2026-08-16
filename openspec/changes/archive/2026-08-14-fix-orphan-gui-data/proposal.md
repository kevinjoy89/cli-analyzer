## Why

Windows 实机扫描的「未认领数据」仍被 GUI 应用数据淹没：`%APPDATA%`/`%LOCALAPPDATA%` 顶层 34 项中 31 项是 GUI 应用数据（PotPlayerMini64、QQEX、The Quark Authors、腾讯电脑管家-全局信息、tabby/termius-updater 各 255–300MB、GameViewer、AweSun、IObit、XnViewMP、Rufus 等）——厂商排除表采用路径片段精确匹配，产品目录名与模式不完全相等即漏网（`PotPlayerMini64` ≠ `potplayer`、`QQEX` ≠ `qq`、`腾讯电脑管家-全局信息` ≠ `tencent`）；`tabby-updater` 等 Squirrel 更新器目录与 `flutter_webview_windows` 等 GUI 运行时目录无任何规则命中；`ConnectedDevicesPlatform`/`PlaceholderTileLogoFolder`/`VoiceAccess` 属 Windows 系统组件却未收录；本应用自身（`CLI Analyzer.exe`，74MB）也因自身排除只匹配 `cli-analyzer` 而漏出。白名单永远列不完，需要确定性、可扩展的排除来源。

## What Changes

- **已安装应用交叉验证（Windows 新增确定性规则）**：未认领目录名与「已安装应用」证据交叉比对——注册表卸载项 DisplayName（HKCU + HKLM 32/64 位）与开始菜单快捷方式名（用户 + 公共）。匹配规则：精确相等；目录名是应用名前缀（`PeaZip` ⊂ `PeaZip 11.0.0 (WIN64)`，目录名长度 ≥ 5）；应用名是目录名前缀且余段以 `-` 开头（`termius-updater` ⊃ `Termius`，应用名长度 ≥ 5，命中 `<App>-updater` 类目录）。这是安装证据而非名称形态启发式；短名（`pc`、`git`、`code`）因长度门槛不参与前缀匹配。
- **应用更新器/运行时目录结构规则**：目录名（小写）以 `-updater` 结尾视为 GUI 应用自动更新器暂存目录（Squirrel/electron-updater 约定 `<App>-updater`），整体跳过；`flutter_webview_windows` 等 GUI 运行时目录入厂商表。
- **自身排除修复**：`isSelfDataDir` 除 `cli-analyzer` 外，匹配应用产品名（`CLI Analyzer`/`CLI Analyzer.exe`——exe 可能被复制或改名运行）与本应用可执行文件基名，`%APPDATA%\CLI Analyzer.exe` 不再作为孤儿。
- **系统组件目录增补**：`ConnectedDevicesPlatform`、`PlaceholderTileLogoFolder`（LocalAppData）、`VoiceAccess`（Roaming）列入系统目录；`SquirrelTemp`（Squirrel.Windows 更新器临时目录）列入 Windows 系统组件；`go-build`（Go 工具链构建缓存，共享基础设施）列入跨平台共享目录
- **厂商表补充**：本次 Windows 实测漏网产品（awesun/sunlogin/oray/iobit/neatdm/notepadnext/potplayermini64/daum/qqex/qarmin/the quark authors/xnviewmp/utforpc/utorrent/gameviewer/rufus/hd tune pro/tabby/termius/pixpin/localsend/peazip/flutter_webview_windows/atlassian），均为 DataOnly 产品级条目。

## Capabilities

### New Capabilities
- `installed-app-crosscheck`（Windows）：已安装应用交叉验证——注册表卸载项 DisplayName 与开始菜单快捷方式名作为孤儿数据排除的确定性证据源

### Modified Capabilities
- `non-cli-exclusion`: 确定性过滤新增「已安装应用交叉验证」与「应用更新器目录」两类规则；系统目录增补 Windows 组件
- `orphan-data`: 排除体系接入上述新规则；自身排除覆盖应用产品名

## Impact

- **代码**：`internal/platform/`（新增 installedapp.go/installedapp_windows.go/installedapp_other.go，systemdirs.go 增补，vendorexclusion.go 增补）、`internal/scanner/scanner.go`（isSelfDataDir + findUnattributed 接入）
- **行为**：Windows 实测未认领 34 → 4（`pc`、`clink` 为真实 CLI 残留，`Backup` 15B 未知保留，`node-addon-native-custom-loader` 为 node 工具链缓存保留）；`CLI Analyzer.exe`、`go-build`(167MB)、`tabby-updater`(300MB)、`termius-updater`(255MB) 等全部消失
- **不引入**：不做名称形态启发式（`MatchInstalledAppName` 的长度门槛与 `-` 后缀约束保证短名/普通名不误伤）；注册表/开始菜单仅读取不写入；macOS/Linux 行为不变（交叉验证桩返回 false）
