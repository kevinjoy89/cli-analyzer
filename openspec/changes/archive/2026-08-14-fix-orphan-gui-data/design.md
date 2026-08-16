# Design: 未认领数据 GUI 泄漏修复（已安装应用交叉验证 + 结构规则 + 自身排除）

## Context

Windows 实机（%APPDATA%/%LOCALAPPDATA%）孤儿扫描 34 项中 31 项是 GUI 应用数据（详见 proposal.md）。根因是排除体系只有「厂商表精确片段匹配」一种数据源：产品目录名与模式不完全相等即漏（`PotPlayerMini64`≠`potplayer`、`QQEX`≠`qq`、`The Quark Authors`≠`quark`、`腾讯电脑管家-全局信息`≠`tencent`），且不存在「已安装应用」维度的判定。白名单按机器补条目无法收敛。

## Goals / Non-Goals

**Goals**
- Windows 上未认领目录与「已安装应用」证据（注册表卸载项 + 开始菜单快捷方式）交叉验证，确定性排除 GUI 应用数据目录
- 应用更新器目录（`<App>-updater`）与 GUI 运行时目录（`flutter_webview_windows`）结构性排除
- 修复自身排除（`CLI Analyzer.exe` 74MB 漏出）
- Windows 系统组件目录增补（ConnectedDevicesPlatform/PlaceholderTileLogoFolder/VoiceAccess）

**Non-Goals**
- 不引入名称形态启发式（spec 红线）；`MatchInstalledAppName` 用长度门槛 + `-` 后缀约束保证短名/普通名不误伤
- 不读写注册表/开始菜单以外的系统状态；注册表只读
- 不改变孤儿处置路径（仍仅可移入回收站）
- macOS/Linux 行为不变（交叉验证桩返回 false；macOS GUI 数据在 Application Support，本就不是孤儿来源）

## Decisions

### D1: 已安装应用证据源（Windows）

`loadInstalledAppNames()`（sync.Once 缓存）收集：
- 注册表卸载项 DisplayName：`HKCU` 与 `HKLM` 下 `Software\Microsoft\Windows\CurrentVersion\Uninstall`，HKLM 另含 `Software\WOW6432Node\...`（32 位视图）；逐子键读 `DisplayName`
- 开始菜单快捷方式基名：`%APPDATA%\Microsoft\Windows\Start Menu\Programs` 与 `%ProgramData%\Microsoft\Windows\Start Menu\Programs` 递归枚举 `*.lnk`（只取文件名，不解析链接目标——避免 COM/二进制解析）

实测命中率（本机 34 项）：注册表前缀命中 LocalSend/PeaZip/Termius/Tabby/Notepad Next/PixPin/腾讯电脑管家；快捷方式补充 VoiceAccess；二者合计约 12 项。短名目录（`pc`、`clink`）与未安装应用残留（`Backup` 15B）不受影响，保持孤儿候选。

### D2: 匹配规则（MatchInstalledAppName，纯函数、全平台可测）

对目录名 `n` 与每个应用名 `a`（均小写化、去空白）：
1. `n == a` → 命中（精确；无长度门槛）
2. `len(n) >= 5 && strings.HasPrefix(a, n)` → 命中（目录名是应用名前缀——应用名常为 `目录名 + 版本/描述`：`PeaZip` ⊂ `PeaZip 11.0.0 (WIN64)`；`IObit` ⊂ `IObit Uninstaller`）
3. `len(a) >= 5 && strings.HasPrefix(n, a) && strings.HasPrefix(n[len(a):], "-")` → 命中（应用名是目录名前缀且余段以 `-` 开头——`<App>-updater` 类：`termius-updater` ⊃ `Termius`；`腾讯电脑管家-全局信息` ⊃ `腾讯电脑管家`）

门槛设计：规则 2/3 的 5 字符门槛挡住 `pc`/`git`/`go`/`code` 等短名；规则 3 的 `-` 后缀约束挡住 `gitea` ⊃ `git`、`notepadnext` ⊃ `notepad` 类误伤（`NotepadNext` 由厂商表 `notepadnext` 精确覆盖）。`clink` 无卸载项/快捷方式 → 保留为孤儿（真实 CLI 工具）。

### D3: 更新器目录结构规则（IsUpdaterDir）

`strings.HasSuffix(strings.ToLower(name), "-updater")` → 排除。Squirrel/electron-updater 约定 `<App>-updater` 是 GUI 应用自动更新暂存目录（tabby-updater 300MB、termius-updater 255MB、qmlauncher-updater），不是任何工具的数据残留。归为结构性规则（与 `packages`/`Programs` 同类），非名称形态启发式；全平台生效（unix XDG 根下同名目录同样不可能是 CLI 数据）。

### D4: 自身排除修复

`isSelfDataDir` 在 `cli-analyzer` 精确匹配之外，取 `os.Executable()` 基名（含扩展名 `CLI Analyzer.exe` 与去扩展名 `CLI Analyzer` 两种形态）做大小写不敏感匹配。Wails 产品名 `CLI Analyzer` 的 exe 出现在 AppData 顶层时不再自我归因。`go test` 下 os.Executable 为测试二进制名，测试夹具目录名不会与之冲突。

### D5: 系统目录与厂商表增补

- `windowsSystemDataDirs` += `connecteddevicesplatform`、`placeholdertilelogofolder`（LocalAppData）
- `systemDataDirs` += `voiceaccess`（Windows 语音访问组件；在 Roaming 根出现，故放跨平台表使其对 AppData 也生效）、`go-build`（Go 工具链构建缓存，`~/.cache/go-build` / `%LocalAppData%\go-build`，实机 167MB，与 configstore 同类共享基础设施）
- 厂商表（DataOnly）：awesun/sunlogin/oray/iobit/neatdm/neat download manager/notepadnext/notepad next/potplayermini64/daum/qqex/qarmin/the quark authors/xnviewmp/xnview/utforpc/utorrent/gameviewer/rufus/hd tune pro/hdtunepro/tabby/termius/pixpin/localsend/peazip/flutter_webview_windows

### D6: 测试策略（CI 仅 ubuntu/macos）

- 匹配器 `MatchInstalledAppName` 为纯函数 → 全平台测试
- Windows 加载逻辑与 `IsInstalledAppDataDir` 用 `installedAppNamesOverride` 注入假名单测试（`//go:build windows`）
- `IsUpdaterDir`/系统目录/厂商表 → platform 层全平台测试
- scanner 孤儿测试在 Windows 上跳过 XDG 根用例（Windows 无 XDG 语义），Windows 语义由 platform 层覆盖；新增的 updater 目录用例跨平台运行（unix 走 XDG config，Windows 走隔离的 %APPDATA%）；本机（Windows）全量 `go test ./internal/...` 手动回归

## Risks / Trade-offs

- [注册表 DisplayName 前缀误伤同名 CLI 数据目录] → 5 字符门槛 + 语义（应用显示名以目录名开头意味着目录即该应用的常见数据名）；孤儿仅展示（USER 级、只进回收站），不破坏数据
- [`<App>-updater` 规则误伤未来 CLI 目录名] → 罕见；同样仅影响孤儿展示
- [注册表读取失败（权限/视图）] → 失败静默降级，仅损失该证据源，厂商表兜底
- [性能] → sync.Once 缓存，一次扫描仅枚举一次（百级键 + 两级快捷方式目录，毫秒级）

## Migration Plan

无持久化格式变化。孤儿过滤为纯展示语义，重扫即生效；旧缓存随 `--refresh`/GUI 重扫自然更新。实机验证（以 `CLI Analyzer.exe` 命名的二进制运行）：未认领 34 → 4（`pc`/`Backup`/`clink` 为真实 CLI 残留或未知小目录，`node-addon-native-custom-loader` 为 node 工具链缓存，均保留）。
