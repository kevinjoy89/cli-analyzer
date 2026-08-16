## Why

产品信任模型存在结构性风险：应用承诺"判定哪些目录可安全清理"，但判定依赖手工 curated 规则表 + generic 名称启发式。工具生态无限增长、新目录形态不断出现（opencode/mimocode 的 `~/.cache` 实为插件安装目录、codex-runtimes 等），规则永远追不上生态；一次误判（把插件目录标成 SAFE 可删）就足以永久失去用户信任。维护成本无限、信任上限有限。

解法不是继续扩充规则表，而是**退一步**：应用从"安全裁判"变为"归因展示 + 执行用户选择"。所有归因目录只打类型标签（缓存/配置/数据/旧版本/工具链…），不再区分"可删/不可删"；删除与否由用户决定。

## What Changes

- **cleaner 拆除两级硬门槛**：删除 `cleaner.go` 中 `Tier != SAFE → Skipped` 的两个分支；保留完整性守卫（guard：绝对路径、无 `..`、禁删系统根、非回收站本身、非当前版本路径）——守卫防灾难性错误，不是替用户做判断
- **scanner 语义降级**：Tier 从门槛变为信息标签；全部归因数据目录（安装根除外）生成可处置项（按物理路径去重，平台无关）；`cleanableBytes` 语义调整为"可处置项合计"，`userBytes = footprint − cleanable`（≈ 安装根 + 独立二进制）；子项 breakdown 对所有可处置项计算
- **uninstall 残留双选项**：默认仍走内置回收站（可恢复）；新增 `RemoveResidues` 永久删除变体（GUI 强确认 + CLI `--permanent`），与 trash 版相同的路径白名单校验
- **CLI 批处理保护性默认**：`clean --all` 默认只选"清理类"（cache/old-version/backup/download）；config/data/state/toolchain 需 `--include-data` 或逐项指定 id——防误操作的默认值，不是门槛
- **前端**：详情面板展示全部可处置项（kind 标签为信息），安装根单独展示（删安装根 = 卸载，走卸载流程）；确认文案从"仅 SAFE 可清理"改为"移入回收站，处置由你决定"
- **i18n/文档**：三语言键集同步更新（`cln.userSkipped` 移除，新增 `--include-data`/永久删除相关文案）；README 安全模型叙述重写为处置模型

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities

- **tool-attribution**（`openspec/specs/tool-attribution/spec.md`）：Tier 不再参与清理门禁，仅作展示标签
- **trash-recycle-bin**（`openspec/specs/trash-recycle-bin/spec.md`）：回收站仍是默认删除去向；永久删除始终是显式选择
- **tool-uninstall**（`openspec/specs/tool-uninstall/spec.md`）：残留处置默认回收站，新增用户显式选择的永久删除
- **non-cli-exclusion / orphan-data**：不受影响（未认领数据仍仅可移入回收站）

## Impact

- **代码**：`internal/cleaner/cleaner.go`（拆门槛）、`internal/scanner/attribute.go`（可处置项生成 + totals 语义）、`internal/scanner/types.go`（Tier 注释）、`internal/cli/clean.go`（白名单 + `--include-data`）、`internal/cli/output.go`、`internal/cli/uninstall.go`（`--permanent`）、`internal/uninstall/residue.go`（`RemoveResidues`）、`gui/service.go`（`UninstallDeleteResidues` 绑定）、`frontend/wailsjs/*`（绑定同步）、`frontend/src/main.ts`（详情面板 + 残留永久删除按钮）
- **i18n**：三个 locale 键集一致（345 键），新增/修改约 30 个文案
- **测试**：cleaner 两个"USER 必拒"用例改为"USER 可删"；`TestFinalize` totals 断言更新；新增 `clean --all` 白名单/`--include-data`/数据类提示、`RemoveResidues`、全部数据目录可处置用例
- **文档**：README.md / README.zh-CN.md 安全模型 → 处置模型；本 change 记录设计
- **不引入**：不改变孤儿处置路径；不改变完整性守卫语义；不新增依赖；不改变 JSON 字段名（缓存/历史兼容）
