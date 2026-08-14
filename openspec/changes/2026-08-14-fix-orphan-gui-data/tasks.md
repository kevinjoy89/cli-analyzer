## 1. 已安装应用交叉验证（Windows）

- [x] 1.1 `internal/platform/installedapp.go`：纯匹配器 `MatchInstalledAppName(name, apps)`——精确 / 目录名前缀（≥5）/ 应用名前缀且余段 `-` 开头（≥5）
- [x] 1.2 `internal/platform/installedapp_windows.go`：`loadInstalledAppNames()` 收集注册表卸载项 DisplayName（HKCU + HKLM + WOW6432Node）与开始菜单快捷方式基名（用户 + 公共，递归 .lnk）；sync.Once 缓存；`installedAppNamesOverride` 测试注入钩子
- [x] 1.3 `internal/platform/installedapp_other.go`：非 Windows 桩返回 false
- [x] 1.4 单测：匹配器全平台用例（含 pc/gitea/opencode 不误伤、termius-updater/腾讯电脑管家-全局信息 命中）；Windows 注入用例 + 真实加载冒烟

## 2. 更新器/运行时目录结构规则

- [x] 2.1 `platform.IsUpdaterDir(name)`：小写 `-updater` 后缀判定
- [x] 2.2 单测：updater 目录各平台命中；普通目录不受影响

## 3. 系统目录与自身排除

- [x] 3.1 `windowsSystemDataDirs` += connecteddevicesplatform / placeholdertilelogofolder / squirreltemp；`systemDataDirs` += voiceaccess / go-build
- [x] 3.2 `scanner.isSelfDataDir` 匹配 `os.Executable()` 基名（含/不含扩展名）与应用产品名（CLI Analyzer / CLI Analyzer.exe），保留 `cli-analyzer` 精确匹配
- [x] 3.3 单测：系统目录判定用例扩充；isSelfDataDir 用例

## 4. 厂商表补充

- [x] 4.1 vendorexclusion.go DataOnly 增补：awesun/sunlogin/oray/iobit/neatdm/neat download manager/notepadnext/notepad next/potplayermini64/daum/qqex/qarmin/the quark authors/xnviewmp/xnview/utforpc/utorrent/gameviewer/rufus/hd tune pro/hdtunepro/tabby/termius/pixpin/localsend/peazip/flutter_webview_windows/atlassian
- [x] 4.2 单测：新增条目孤儿语境命中；不拦 exe 发现（DataOnly 语义回归）

## 5. scanner 接入与回归

- [x] 5.1 `findUnattributed` 接入 `IsUpdaterDir` + `IsInstalledAppDataDir`
- [x] 5.2 Windows 孤儿测试隔离：`isolateXDGRoots` 同步隔离 APPDATA/LOCALAPPDATA；XDG 语义用例在 Windows 上 skip（无 XDG 根）；新增 updater 用例跨平台运行
- [x] 5.3 本机 Windows 全量回归：gofmt 合规 + go vet ./internal/... + go test ./internal/...（新增用例全过；遗留的 Windows 环境性失败由后续变更 2026-08-14-windows-test-portability 统一修复：真实 bug 修生产、unix 专属语义显式跳过、CI 增加 windows-latest）；孤儿过滤实机验证 34 → 4（pc、clink、Backup、node-addon-native-custom-loader 保留）
