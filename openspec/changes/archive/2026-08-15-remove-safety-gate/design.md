# Design: 移除 SAFE/USER 硬门槛——处置决策交给用户

## Context

原设计（add-trash-recycle-bin / add-orphan-data-and-probing 系列）引入两级安全模型：SAFE（可删）与 USER（仅展示，cleaner 硬门槛拒绝）。信任模型是"应用裁决可清理性"，裁决依据是 curated 规则表 + generic 启发式。风险：规则表有限、工具生态无限；`~/.cache/<name>` 一概标 SAFE 对 opencode/mimocode 这类"名为 cache 实为插件目录"的工具是误判；一次误判即摧毁信任。

## Goals / Non-Goals

**Goals**
- 移除 cleaner 对非 SAFE 项的硬拒绝：所有归因目录（安装根除外）都是可处置项
- Tier 降级为信息标签（safe/user → 展示用），JSON 字段名与缓存格式不变（兼容旧缓存/历史）
- 默认处置仍走内置回收站（可恢复）；永久删除始终是显式选择
- `clean --all` 批处理默认只含"清理类"，数据类需显式 `--include-data`（防误操作默认，非门槛）
- uninstall 残留提供"回收站/永久"双选项，路径白名单校验与 trash 版一致

**Non-Goals**
- 不改变完整性守卫（guard/guardSub：绝对路径、无 `..`、禁删系统根、非回收站本身、非当前版本路径）——这是防灾难性错误，不是安全判断
- 不改变孤儿（unattributed）处置路径（仍仅可移入回收站）
- 不引入官方清理命令向导（后续独立 change）
- 不改变扫描/归因/计量逻辑本身

## Decisions

### D1: 门禁拆除的位置与保留物

`internal/cleaner/cleaner.go` 的 `clean()` 中两处 `c.Tier != scanner.TierSafe → Skipped`（整项 + 子项）删除。`guard()`/`guardSub()` 与 `forbiddenRoots`/`windowsForbiddenRoots` 原样保留——平台分文件的系统根保护（unix `/`、`/usr`…；Windows `C:\Windows`、`Program Files`…）是纵深防御，任何来源的删除请求都不能命中。

### D2: 可处置项生成（scanner）

`attribute()` 中"仅 cache 类数据目录自动成为 cleanable"改为"全部归因数据目录（`Kind == "install"` 除外）生成可处置项"，Tier 透传为标签。**按物理路径去重**（平台无关）：macOS 上 XDG 根（`~/.cache`）与 Library 根（`~/Library/Caches`）是不同物理目录，不存在规则级冲突；但防御性按路径去重成本极低。安装根（brew Cellar、node_modules/<pkg>、versions/…）不作为可处置项——删除安装根 = 卸载工具，归卸载流程。

`finalize` 汇总语义：`cleanableBytes` = 全部可处置项之和；`userBytes = footprint − cleanable`（≈ 安装根 + 独立二进制）。字段名不变，前端/i18n 文案同步改。

### D3: 批处理保护性默认（CLI）

`clean --all` 默认只批量选择 kind ∈ {cache, old-version, backup, download}；`--include-data` 追加 {config, data, state, toolchain, logs}。交互式逐项确认路径不变（用户逐项决策本来就明确）。`--all` 未命中任何清理类时输出提示（`cli.cleanAllEmpty`），不静默清 0 项。

### D4: 残留处置双选项（uninstall）

默认 `TrashResidues`（回收站，可恢复）不变；新增 `RemoveResidues`（`os.RemoveAll`，不可恢复）。GUI 绑定 `UninstallDeleteResidues` 与 trash 版共享同一路径白名单（只接受 `Residues()` 返回的路径）；前端永久删除按钮先经 `confirmDialog` 强确认。CLI 增加 `--permanent` 标志，交互确认文案区分（`un.deletePermanentPrompt` vs `un.trashPrompt`）。

### D5: 前端展示

详情面板"可处置项"= 全部 cleanables（kind 标签为信息色块）；数据目录区只显示安装根（`ui.sectionInstallRoot`），避免与可处置项重复。确认弹窗文案改为"移入内置回收站（保留期内可恢复）。处置与否由你决定"。

### D6: 平台差异处理

- 去重键是物理路径（`filepath.Clean` 后比较），与平台根解析无关
- guard 系统根按平台分文件（`forbiddenRoots` + `windowsForbiddenRoots`），本次不改
- 新测试全部使用 `t.TempDir()` 隔离，不断言具体 OS 根路径；`clean --all` 测试通过 `XDG_CACHE_HOME`/`LOCALAPPDATA` 双环境变量隔离缓存根，unix/Windows 均成立

## Risks / Trade-offs

- [用户误删 config/data] → 默认走回收站（可恢复）；GUI 强确认；CLI 批处理默认不含数据类；工具链类保留 `Keep` 字段与 `ui.kind.oldToolchain` 风险文案
- [`clean --all --yes --include-data` 全量删除] → 这是用户显式叠加三个标志的结果，符合"选择权交给用户"；回收站仍是默认去向
- [旧缓存（仅 SAFE 项）在升级后列表"变少"] → 重扫即恢复完整列表；不破坏兼容
- [totals 语义变化影响趋势图/铃铛] → 字段名不变，数值口径统一为"可处置"，文案同步更新

## Migration Plan

无持久化格式变化（Tier 字段保留）。旧 `last-scan.json` 可直接读取；GUI/CLI 重扫后按新语义生成可处置项。已有回收站项不受影响。文档与三语言文案随本 change 同步。
