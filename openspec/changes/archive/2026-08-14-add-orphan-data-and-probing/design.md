# Design: 孤儿数据 + 非 CLI 排除体系 + 健康探测

## Context

现状（参见 proposal.md）：`findUnattributed` 后端已实现但仅 CLI `--full` 可用、GUI 未接入；`isVendorInstallDir` 用子串匹配且无例外；版本列对大多数工具为 `—`。红线：USER 级残留唯一处置路径是内置回收站。GUI 每次启动触发扫描。

## Goals / Non-Goals

**Goals**
- 单一排除表贯穿 exe 发现与数据归因，语义统一
- 孤儿数据 GUI 展示与回收站处置，CLI/GUI 结果一致
- 探测不阻塞、可缓存、失败静默

**Non-Goals**
- 体检报告/建议中心（已否决）
- 一键清理全部（已否决）
- 永久删除路径（红线不变）
- 过时检测/更新源查询（依赖被砍的探测-报告链路，不做）

## Decisions

### D1: 排除表结构与匹配语义

```go
type VendorExclusion struct {
    Pattern string   // 路径片段，小写精确匹配
    Allow   []string // 纯 CLI 产品名（剥扩展名后小写精确匹配）
}
var vendorExclusions = []VendorExclusion{
    {"microsoft", []string{"az"}},
    {"amazon", []string{"aws"}},
    {"google", []string{"gcloud", "gsutil", "bq"}},
    {"netsarang", nil}, // 无例外：命令行伴侣一律拦
    // …预置 GUI 厂商（adobe/tencent/discord/slack/steam 等，allow 为空）
}
```

匹配：把路径按分隔符拆成片段，任一片段小写等于 Pattern 即命中；命中后若（exe 名或数据目录名）剥扩展名 ∈ Allow 则放行。替代方案（子串匹配）因 `code` 命中 `opencode` 被否决。

**位置**：`internal/platform`（被 scanner 与 uninstall 复用）；替换现有 `isVendorInstallDir`。

### D2: 结构性 GUI 信号（确定性过滤）

- macOS：`~/Library/Containers/<bundle-id>`——顶层目录名匹配 `^[a-z]+\.[a-z0-9.-]+$`（bundle-id 形态）即排除
- Windows：`%LOCALAPPDATA%\Packages\<family>_<hash>`——目录名匹配 `^.+_[0-9a-f]{8,}$` 即排除

两者都是系统约定而非启发式（spec 已锁定）。Linux 无对应约定。

### D3: 孤儿数据管线

```
scanner.Scan: Unattributed = findUnattributed(...)   ← 默认执行（GUI Options{} 即默认）
  过滤层（findUnattributed 内）:
    本应用自身 → 结构信号（macOS Containers bundle-id / Windows UWP 包族
    + Packages 容器）→ 厂商排除表 DataOnly 语境（目录名精确判例）→ 认领集合
    （含各工具 cleanable 规则覆盖的顶层目录，如 codex 的 codex-runtimes）
GUI: 工具面板"未认领数据"标签页（与工具列表并列；按数据根分组，条目含路径/大小，
      勾选/全选 + 二次确认后批量移入回收站；顶部汇总卡展示总占用；无孤儿时不显示标签）
处置: 新绑定 OrphanTrash（trash.Trash，USER 级可恢复）
体积: 扫描内并行计体积（spec 要求大小进扫描结果；磁盘 Sizer 并行 WalkAll，
      排除表过滤先行可显著缩小待走目录。实测：过滤后本机 210→200 目录、
      37.5GB→11.6GB；扫描总时长 ~21s 与工具数据 walk 基线相当，非孤儿引入）
```

`findUnattributed` 现有实现已含认领集合与 WalkAll；改动点：默认启用（去掉 `opts.Full` 门控）、加过滤层（含 cleanable 认领）、GUI 渲染与处置绑定。CLI `--full` 语义保留为兼容，但默认输出含孤儿字段。

### D4: 探测编排与缓存

```
探测: 依次 --version / -V / --help，取首个 0 退出且有输出的 stdout/stderr
超时: 3s，进程组终止（unix Setpgid + kill(-pgid)；windows 依赖 CREATE_NO_WINDOW + TerminateProcess）
并行: 信号量限 N=8 worker；扫描完成后后台跑，进度经事件推 GUI
缓存键: (real, size, mtime) → 版本/描述；存 CacheRoot/versions.json
解码: windows 下先尝试 UTF-8，失败用 GBK 解码（golang.org/x/text/encoding/simplifiedchinese），再剥离控制字符
降级: 任何失败 → version 保持原值，无错误事件
```

**GUI 状态**：列表行版本列"探测中…"→ 完成后更新；缓存命中时直接显示。探测结果并入 `ScanResult.Tools[].Version`（JSON 契约不变，仅填充既有字段——spec 已锁定）。

### D5: 探测与排除的独立性

探测只作用于已通过排除体系的工具列表，两者解耦：排除表改动不影响探测模块。

## Risks / Trade-offs

- [排除表漏掉未预置 GUI 厂商] → 清单按用户反馈增长（同 NetSarang 模式）；孤儿数据全走回收站，误判可恢复
- [探测跑挂某些工具（联网检查等）] → 3s 超时杀进程组 + 缓存降频；个别工具可后续加"跳过探测"名单
- [孤儿体积惰性计算导致首次展开慢] → 仅展开时计算；结果随扫描缓存持久化
- [结构性信号误排除极少数合法目录] → 均为系统约定形态，概率极低且回收站兜底
- [GUI 启动即扫描 + 探测叠加耗时] → 探测后台并行、缓存命中即跳；孤儿体积惰性

## Migration Plan

- 部署：随常规发版；扫描缓存格式向后兼容（孤儿/版本字段为增量，旧缓存可被新代码重扫覆盖）
- 回滚：旧版本二进制直接替换；无数据迁移
- 缓存目录残留：旧探测缓存（若存在）位于 CacheRoot，可随缓存清理语义处置

## Open Questions

无（设计决策均已在上文 resolve，且不改变 spec 契约）
