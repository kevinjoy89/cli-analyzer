## Why

未认领数据（孤儿）过滤存在系统性漏洞：`.DS_Store` 等文件被列为孤儿（ReadDir 未检查目录类型）、已安装 GUI 应用（iTerm2/Raycast/Joplin 等）与共享基础设施目录（.mono/configstore/man/`%LocalAppData%\Programs`）漏网、认领匹配大小写敏感（Windows 上 PATH `claude` 与目录 `Claude` 失配）、dataDirs 顶层名异于工具名时不被认领（opencode 的 `oh-my-opencode`）。排除表对高频 GUI 产品（浏览器/聊天/办公/网盘/游戏平台/VPN/驱动）覆盖不足。

## What Changes

- **仅目录列为孤儿候选**：跳过 `.DS_Store`、`.localized` 等普通文件与指向文件的符号链接；指向目录的符号链接仍为合法候选
- **认领语义修正**：认领匹配大小写不敏感（Windows PATH 名 vs 数据目录名）；工具的 dataDirs 参与认领（顶层名异于工具名时不再误报孤儿）
- **系统/共享结构目录排除**：跨平台共享运行时目录（`.mono`、`configstore`、`simple-update-notifier`、`IsolatedStorage`、`.isolated-storage`、`man`）与 Windows 系统组件目录（`%LocalAppData%\Programs`（GUI 安装根）、`Temp`、`CrashDumps`、`D3DSCache`、`lxss`（WSL）、`Comms`）整体跳过
- **排除表扩充**：新增 ~90 条高频 GUI 产品/厂商条目——浏览器（mozilla/firefox/bravesoftware/vivaldi/opera…）、聊天会议（dingtalk/feishu/lark/wecom/signal/whatsapp/webex/wemeet…）、办公网盘（wps/kingsoft/onlyoffice/baidunetdisk/aliyundrive/quark/xunlei/dropbox…）、音乐视频（spotify/netease/bilibili/douyin/kuaishou/iqiyi/potplayer/vlc/kodi/obs-studio…）、编辑器工具（iterm2/raycast/alfred/karabiner/joplin/axure/geany…）、游戏平台（epic games/battlenet/riot/ubisoft/gog/roblox/minecraft/unity/blender…）、VPN 客户端（nordvpn/expressvpn/surfshark/protonvpn/clash-verge/v2rayn…）、密码管理（1password/bitwarden/lastpass）、输入法安全（sogou/iflytek/huorong/360）、驱动硬件厂商（nvidia/intel/realtek/logitech/razer/samsung/huawei…）；**产品级短名条目为 DataOnly**（仅数据语境拦截，不拦 exe 发现，避免误伤真实 PATH 目录）；代理核心 CLI（clash/v2ray/mihomo 本体）不排除
- **归因规则补齐**：opencode 数据目录规则增加 `oh-my-opencode`（插件框架配置归属 opencode）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
- `non-cli-exclusion`: 排除表引入 DataOnly 产品级条目语义（仅数据归因语境）；新增系统/共享结构目录判定（跨平台共享目录 + Windows 系统组件目录）
- `orphan-data`: 孤儿候选仅限目录（文件/symlink 过滤）；认领大小写不敏感且包含 dataDirs；系统/共享结构目录纳入排除体系

## Impact

- **代码**：`internal/platform/`（vendorexclusion.go 排除表扩充、新增 systemdirs.go 系统目录判定）、`internal/scanner/scanner.go`（findUnattributed：isDirEntry 检查、EqualFold 认领、dataDirs 认领、IsSystemDataDir 接入）、`internal/rules/tools.go`（opencode 规则 +oh-my-opencode）
- **行为**：macOS 实测孤儿 25 → 7（.DS_Store/iterm2/raycast 177MB/joplin/karabiner/Axure/geany/.mono/configstore/man 等全部消失）；Windows 的 `%LocalAppData%\Programs`、`Temp`、`CrashDumps` 等系统目录与大批 GUI 产品目录不再统计
- **不引入**：不做名称形态启发式（大小写/空格/公司命名风格仍不参与判定）；代理/网络 CLI 核心（clash/v2ray/mihomo）仍为孤儿候选（未上 PATH 的 CLI 残留语义不变）
