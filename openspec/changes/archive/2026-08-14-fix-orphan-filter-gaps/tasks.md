## 1. 孤儿候选与认领语义修复

- [x] 1.1 findUnattributed 仅列目录：DirEntry Type 位判定 + 符号链接跟随（指向目录保留、指向文件跳过）
- [x] 1.2 认领集合大小写不敏感（aliasSet 与 claimTopLevel 两侧 ToLower 归一）
- [x] 1.3 dataDirs 参与认领（顶层名异于工具名时不再误报）
- [x] 1.4 单测：文件/symlink 过滤、大小写认领（Claude/claude）、dataDirs 认领（oh-my-opencode）

## 2. 系统/共享结构目录排除

- [x] 2.1 platform.IsSystemDataDir：跨平台共享目录（.mono/configstore/simple-update-notifier/IsolatedStorage/.isolated-storage/man）
- [x] 2.2 Windows 系统组件（LocalAppData 下 Programs/Temp/CrashDumps/D3DSCache/lxss/Comms），按 root 区分
- [x] 2.3 findUnattributed 接入 IsSystemDataDir
- [x] 2.4 单测：各 root/名称组合判定 + 孤儿过滤集成

## 3. 排除表扩充

- [x] 3.1 浏览器/邮件（mozilla/firefox/bravesoftware/vivaldi/opera/opera software/thunderbird/foxmail）
- [x] 3.2 聊天会议办公（dingtalk/feishu/lark/wecom/signal/whatsapp/webex/wemeet/wps/kingsoft/onlyoffice）
- [x] 3.3 网盘下载/音乐视频（baidunetdisk/aliyundrive/quark/xunlei/dropbox/megasync/spotify/kugou/kuwo/netease/bilibili/douyin/kuaishou/iqiyi/youku/potplayer/vlc/kodi/obs studio/obs-studio/streamlabs）
- [x] 3.4 编辑器工具/游戏/VPN/密码/输入法（iterm2/raycast/alfred/karabiner/joplin/joplin-desktop/axure/geany/notepad/jgit/epic games/battlenet/riot/ubisoft/gog/roblox/minecraft/unity/blender/nordvpn/expressvpn/surfshark/protonvpn/clash-verge/clash verge/v2rayn/v2ray-n/1password/bitwarden/lastpass/sogou/iflytek/huorong/360）
- [x] 3.5 驱动/硬件厂商双向 + 短厂商 DataOnly（nvidia/nvidia corporation/intel/realtek/logitech/razer/samsung/huawei/xiaomi/lenovo/canon/epson/brother/synaptics/elgato/corsair/steelseries + amd/hp/dell/asus/acer/msi/baidu DataOnly）
- [x] 3.6 DataOnly 语义：数据语境拦、exe 语境不拦；代理 CLI 核心（clash/v2ray/mihomo）不排除
- [x] 3.7 单测：DataOnly 双语境、扩充条目命中、双向厂商 exe 拦截、CLI 核心保留

## 4. 归因规则与回归

- [x] 4.1 opencode 规则 + oh-my-opencode 数据目录
- [x] 4.2 macOS 实测：孤儿 25 → 7（.DS_Store/iterm2/raycast/joplin/karabiner/Axure/geany/.mono/configstore/man 消失；剩余均为未上 PATH 的 CLI 残留）
- [x] 4.3 全量回归：go vet + go test ./... + 三平台交叉编译
