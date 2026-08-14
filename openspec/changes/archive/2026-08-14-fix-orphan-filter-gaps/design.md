# Design: 孤儿过滤漏洞修复 + GUI 排除扩充

## Context

现状（参见 proposal.md）：实测 macOS 孤儿 25 项，其中 18 项是误报——文件、共享基础设施、已安装 GUI 应用。排除表覆盖不足且无"系统结构目录"概念。红线：孤儿只显示不删除（USER 级、仅可移入回收站），本变更不触碰处置路径。

## Goals / Non-Goals

**Goals**
- 孤儿候选仅限目录（修 `.DS_Store` 类误报）
- 认领语义修正：大小写不敏感 + dataDirs 参与认领
- 系统/共享结构目录确定性排除（跨平台 + Windows）
- 排除表扩充高频 GUI 产品，产品级短名 DataOnly 不误伤 exe 发现

**Non-Goals**
- 名称形态启发式（spec 红线不变）
- 代理/网络 CLI 核心（clash/v2ray/mihomo）不排除（它们是 CLI 残留语义的一部分）
- 处置路径变化（不引入删除/清理新路径）

## Decisions

### D1: 文件条目用 DirEntry 类型位判定

`os.ReadDir` 返回的条目用 `e.Type().IsDir()` 判定目录；符号链接（`Type()&ModeSymlink`）再 `os.Stat` 跟随：指向目录的链接是数据目录（保留为候选），指向文件的链接不是。不用 `os.Stat` 全量跟随（避免对每个条目一次额外 stat；普通文件/目录占绝大多数，Type 位直接判定零开销）。实测 `.DS_Store`（4KB 文件）因此消失。

### D2: 认领集合归一化小写

claimed map 的键统一 `strings.ToLower`（aliasSet 与 claimTopLevel 两侧同时归一）。Windows 上 PATH 名 `claude` vs 目录 `Claude` 的失配因此修复。macOS 大小写敏感目录极少同名异写，归一化无副作用（覆盖更广比更窄安全——孤儿误报比误认领危害大）。

### D3: dataDirs 参与认领

原认领 = aliasSet + cleanables；dataDirs 顶层名通常等于工具名（被 aliasSet 覆盖），但 opencode 的 `~/.config/oh-my-opencode` 是例外。新增 dataDirs 循环（与 cleanables 相同的 claimTopLevel 逻辑）。备选：给规则表加别名——会让"包含工具/别名"展示污染，否决。

### D4: 系统/共享结构目录（IsSystemDataDir）

新增 `platform.IsSystemDataDir(root, name)`：跨平台共享目录（`.mono`、`configstore`、`simple-update-notifier`、`IsolatedStorage`、`.isolated-storage`、`man`——node/.NET/系统数据库类共享基础设施）+ `LocalAppData` 下的 Windows 系统组件（`Programs`/`Temp`/`CrashDumps`/`D3DSCache`/`lxss`/`Comms`）。按 root 区分（`Programs` 在 XDG 下不拦）。这是确定性清单（与厂商表同类），不是启发式。删除这些目录危险（运行时共享状态）或无效（系统组件），整体跳过。

### D5: 排除表扩充与 DataOnly 语义

新增 ~90 条高频 GUI 产品/厂商。三类：
- **双向厂商**（microsooft/google 同款）：驱动硬件厂商（nvidia/intel/realtek/logitech/razer/samsung/huawei/xiaomi/lenovo/canon/epson/brother/synaptics/elgato/corsair/steelseries）——目录下为系统组件，不应作为工具扫描或探测
- **产品级 DataOnly**：短名/产品名（iterm2/raycast/mozilla/firefox/dingtalk/spotify/1password/amd/hp/dell…）——只拦孤儿过滤，不拦 exe 发现（短名可能是真实 PATH 目录；且 probe 有 ProbeSafeInstaller 门控兜底，InstOther 不执行）
- **不列入**：代理 CLI 核心（clash/v2ray/mihomo/openvpn）——独立二进制的 CLI 工具；只拦其 GUI 客户端（clash-verge/v2rayN）
含空格目录名单独成条（`nvidia corporation`、`epic games`、`obs studio`、`opera software`——片段匹配是整目录名精确相等，空格不会被拆分）。

### D6: opencode 归因规则

opencode 规则增加 `dd(XDGConfig, "oh-my-opencode")`——插件框架配置归属 opencode（其插件/主题由 opencode 加载），配合 D3 的 dataDirs 认领闭环。

## Risks / Trade-offs

- [排除表条目永远列不完] → 结构性修复（文件/系统目录/认领）砍掉大头；产品条目按"高频 + 高确信目录名"持续维护；Windows 实测如有漏网目录可继续补
- [DataOnly 条目误伤真实 CLI 数据目录（如 iterm2 未来出 CLI）] → 罕见；DataOnly 只影响孤儿展示（USER 级、仅回收站），不阻断任何功能
- [双向厂商条目拦掉厂商目录下的真实工具] → 驱动/硬件目录下的工具均为系统组件（nvidia-smi 等），符合"不执行未知来源二进制"精神；若用户有真实需求可加 Allow 例外
- [大小写归一化误认领两个异写目录] → mac 上极少；误认领（少报孤儿）比误报孤儿安全

## Migration Plan

无持久化格式变化。旧扫描缓存随 `--refresh`/GUI 重扫自然更新；孤儿过滤是纯展示层语义，行为即时生效。
