## Context

现状（详见 proposal.md - Why）：扫描器已按安装来源分类每个工具（brew/npm/pipx/cargo/go/pyenv/versioned/other），规则表持有每工具的数据目录规则（`dd(Root, Sub, Tier, Kind)`），trash 包提供可恢复的回收站移入。本设计在其上叠加"卸载"能力：官方命令 + 残留双源检测 + 残留回收站清理，全程可反悔。

## Goals / Non-Goals

**Goals:**
- 用户无需知道卸载命令：工具按来源给出官方命令，可代跑。
- 残留检测覆盖规则表目录与扫描归因目录（双源），卸载后一次串联完成。
- USER 级数据（配置/凭证）的唯一删除路径是内置回收站（可恢复）；不新增永久删除路径。
- CLI/GUI 双入口共享同一 core，全部文案本地化。

**Non-Goals:**
- 不做整包"强制卸载"（手动删可执行文件+全部足迹）：包管理器一致性由官方卸载保证，避免共享文件/DB 不一致风险面。
- 不做依赖图分析（无法静态得知 npm/pip 依赖关系）；以黑名单 + 官方卸载委托规避级联风险。
- 不清理共享文件（`.npmrc`、shell 别名等）——残留仅限工具专属数据目录。
- 不做卸载后系统级残留扫描（launchd 服务、注册表项等）——超出文件归因能力，列为提示而非删除。

## Decisions

### D1. 官方卸载命令表（按安装来源）

`internal/uninstall` 维护来源 → 命令模板映射：

| 来源 | 命令 |
|---|---|
| brew | `brew uninstall <formula>`（formula 名取工具别名规则或当前版本名） |
| npm | `npm uninstall -g <pkg>`（pkg 从二进制 real path 的 node_modules 解析） |
| pipx | `pipx uninstall <pkg>` |
| cargo | `cargo uninstall <crate>` |
| go | 无包管理器：提示 `rm <GOPATH>/bin/<bin>`（仅提示，不走强卸） |
| pyenv | 提示在对应 pyenv 版本内卸载（`pip uninstall`/`pipx`），仅提示 |
| versioned / other | 无标准命令：直接进入残留检测（本工具的价值点在残留清理） |

- 命令拼接所需参数（formula/pkg/crate 名）由扫描器的归类信息解析；解析不到时仅展示命令模板 + 提示人工补全。
- 纯提示来源（go/pyenv/versioned/other）：不提供代跑，仅展示建议。

### D2. 代跑执行

- GUI「执行官方卸载」确认后执行：`exec.Command(bin, args...)`，stdout/stderr 合并流式经事件（`uninstall:output`）推给前端；完成后推 `uninstall:done`（成功/失败+exit code）。
- 加超时（默认 5 分钟）防卡死（如 sudo 等待密码）；超时/失败 → 明确提示"可复制命令自行执行"，并继续残留检测（spec: 卸载失败仍继续）。
- 复用 updater 的经验教训：事件在 macOS WKWebView 不可靠 → 输出改用轮询 `GetUninstallOutput()` 或一次性结果返回（输出量小，可用事件 + 完成事件兜底；若实测不可靠则切轮询，同 update 的进度方案）。

### D3. 残留检测（双源）

```
残留源 = 规则表 dd() 目录（按当前平台 roots 解析出绝对路径）
       ∪ 卸载前扫描快照中归因到该工具的 dataDirs
过滤: 路径仍存在 → 残留项 {path, tier, kind, bytes}
标注: tier==USER（config/history/凭证）→ "含登录凭证"标红
```

- 规则表目录是确定性基础（即使工具从未被扫过）；快照目录兜住规则表外的临时/自定义目录。
- 不主动扫描未知残留（unattributed 目录不归因给具体工具，不删）。
- 二进制消失与否不影响残留检测（残留只查数据目录）。

### D4. 残留清理：回收站硬约束

- 复用 `trash.Trash()`：残留项移入内置回收站（同文件系统瞬时、可恢复），保留期/到期清除沿用现有配置。
- 这是唯一允许触碰 USER 数据的路径，且**不提供绕过**（无 `--permanent` 变体）。`--yes` 仅跳过交互确认，回收站约束不变（spec: 不允许永久删除残留）。
- 移入失败（如跨文件系统、Windows 进程占用）→ 列出失败项并提示原因，不静默丢弃。

### D5. 系统关键工具黑名单

- 硬编码黑名单：python/python3/node/npm/git/docker/go/brew/bun/yarn/pnpm/bash/zsh/cli-analyzer 自身 等基础设施。
- 命中 → GUI 无卸载入口（或点击即拒绝+说明）；CLI `uninstall` 返回错误与原因。
- 规则表可扩展（未来新增基础设施工具时补表）。

### D6. 交互模型

- GUI 残留列表：默认全选；USER/凭证项红色标注；一次确认，确认文案点明"配置类数据将被删除（移入回收站，保留期内可恢复）"。
- CLI：
  - `cli-analyzer uninstall <tool>`：显示官方命令 → 询问是否代跑（仅可代跑来源）→ 代跑后残留检测 → 询问是否清理残留（默认全选）→ 执行。
  - `cli-analyzer uninstall <tool> --residue`：仅列出残留（含级别），不删除。
  - `cli-analyzer uninstall <tool> --yes`：跳过交互（代跑确认 + 残留清理确认），回收站约束不变。
  - 退出码：0 成功 / 1 错误 / 2 被黑名单拒绝或无该工具。
- 文本输出走 i18n；`--json` 供脚本（契约保护：键名英文不变）。

### D7. GUI 流程（详情页）

- 详情页工具头部新增「卸载」按钮（danger 样式），点击：
  1. 确认弹窗：官方命令（复制按钮）+ 占用摘要 + （黑名单工具 → 直接拒绝说明）
  2. 「执行官方卸载」→ 输出区流式显示（或"复制命令自行执行"）
  3. 自动进入残留检测 → 残留列表弹窗（全选 + 凭证标红）→ 确认 → 移入回收站
  4. 完成后：toast 摘要 + 自动 rescan（刷新列表）+ 回收站状态栏更新
- 弹窗状态机：confirm → running(可选) → residue → done/failed，每步可取消。

### D8. CLI 子命令接入

- `main.go` 分发加 `uninstall`；`internal/cli/run.go` usage 更新。
- 纯逻辑在 `internal/uninstall`，CLI 只做参数解析与输出（沿用现有子命令模式）。

### D9. i18n

- 新键域 `un.*`：`un.confirmTitle`、`un.officialCmd`、`un.runOfficial`、`un.copyCmd`、`un.residueTitle`、`un.residueNone`、`un.residueCredential`、`un.trashConfirm`、`un.blockedSystem`、`un.done`、`un.failed` 等，三语（zh-CN/zh-TW/en），parity 测试兜底。

## Risks / Trade-offs

- 代跑命令可能卡在 sudo/交互提示 → 超时 + 明确回退"复制命令自行执行"；不尝试注入密码。
- 残留检测可能漏（规则表外的自定义目录且无扫描快照）→ 双源已最大化覆盖；提示用户"可用 `scan --full` 查看未归因目录"。
- 凭证误删 → 回收站可恢复 + 标红 + 确认文案点破；恢复面板可还原。
- Windows 残留目录被进程占用 → trash 移入失败时明确报错（不静默），提示关闭相关进程后重试。
- 命令表与工具演进脱节（新包管理器出现）→ 表集中维护 + other/versioned 兜底（纯残留清理）。

## Migration Plan

1. 纯增量：新子命令 + 新按钮 + 新包，不改既有 clean/trash 行为与规格。
2. 配置无变更；回收站复用现有保留期/到期配置。
3. 回滚：revert 提交即可，无破坏性变更。
4. 发布：随下一版本，语言文件同步加新键。

## Open Questions

无阻塞项。以下留待后续：
- 卸载后系统级残留（launchd/注册表/环境变量）扫描——超出文件归因范围，v2 候选。
- 依赖重叠警告（其他工具共享同一包根的启发式提示）——当前由黑名单+官方卸载规避。
