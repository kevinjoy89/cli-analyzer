## Why

用户想卸载某个 CLI 工具时常常不知道正确的卸载命令（brew uninstall / npm uninstall -g / pipx uninstall…），且官方卸载后往往残留配置、缓存等数据目录。本工具已能识别每个工具的安装来源并归因其全部数据目录——自然可以进一步提供"官方卸载 + 残留检测 + 残留清理"的完整卸载流程，且残留清理走内置回收站（可恢复），延续工具"safely"的核心承诺。

## What Changes

- 新增 `internal/uninstall/` 包：官方卸载命令表（按安装来源）、残留检测（双源：规则表数据目录 + 卸载前扫描快照归因目录）、残留入回收站编排。
- CLI 新子命令 `uninstall`：`uninstall <tool>`（显示官方命令 + 可选代跑）、`uninstall <tool> --residue`（检测残留）、`--yes` 跳过交互（USER 残留仍走回收站、可恢复）。
- GUI：工具详情页新增「卸载」按钮，流程自动串联：确认弹窗（展示官方命令 + 占用摘要）→ 显示命令/代跑 → 残留检测 → 残留列表（凭证类标红，全选默认）→ 一次确认 → 移入回收站。
- 安全约束：
  - 系统关键工具（python/node/git/docker/go/brew 等）与 cli-analyzer 自身 → 拒绝卸载。
  - 残留清理是唯一允许触碰 USER 级数据的操作，硬约束为**必须移入内置回收站**（可恢复），到期按现有配置清除；`--yes` 不豁免。
  - 共享可执行文件（多个工具指向同一 real path）不参与强卸（本范围不做整包强卸，无此风险面）。
- 卸载动作自动串联残留检测与清理，非拆散命令。

## Capabilities

### New Capabilities
- `tool-uninstall`: 官方卸载、残留检测与残留回收站清理。覆盖：安装来源 → 官方命令映射、代跑执行与输出、残留双源检测、凭证类标注、USER 数据回收站硬约束、系统工具黑名单、CLI/GUI 双入口与自动串联流程。

### Modified Capabilities
<!-- 无：clean/trash 的既有需求不变；残留清理复用 trash 能力，不修改其规格。 -->

## Impact

- 新增包：`internal/uninstall/`（命令表 + 残留检测 + 编排，纯逻辑可单测）。
- 修改：`internal/cli/`（新子命令 + usage）、`gui/service.go`（新绑定）、`main.go`（分发）、前端 `main.ts`/`index.html`（详情页卸载按钮 + 卸载流程弹窗）、`internal/scanner/`（暴露卸载前快照所需的数据目录信息，若不足）。
- 复用：`internal/trash`（残留移入回收站）、`internal/platform`（各平台 roots）、`internal/rules`（数据目录规则）、`internal/i18n`（全部文案本地化，含新键）。
- 测试：命令表正确性、残留检测双源、回收站约束（USER 不直删）、黑名单、CLI 退出码与 JSON。
- 无新增第三方依赖。
