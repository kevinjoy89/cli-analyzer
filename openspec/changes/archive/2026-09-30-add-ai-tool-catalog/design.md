## Context

产品从「通用 CLI 磁盘分析」重新定位为「AI 编码工具磁盘管家」的第一步：先支持本机实测存在的 AI 工具。编目沿用 `internal/rules/tools.go` 的 `dd()` 模式（目录规则 + tier + kind），AI 语义通过扩展 kind 集合实现，不引入新层级。

### 本机实测第一批清单（2026-09-30，macOS）

| 工具 | 类别 | 状态 | 关键目录与语义（实测占用） |
|---|---|---|---|
| Claude Code 2.1.281 | agent | 已装（编目已存在，补语义） | `~/.claude`：transcripts 39M(session)、projects 17M(session)、plugins 10M(plugins)、telemetry 7.6M(telemetry)、file-history(checkpoint)、shell-snapshots、cache、backups |
| Codex CLI 0.156.1 | agent | 已装（CLI + ChatGPT.app 内桌面端） | `~/.codex`：plugins 316M(runtime)、packages 311M(runtime)、sessions 176M(session)、archived_sessions 25M(session)、logs_2.sqlite 93M(session-db)、thread_history_1.sqlite 61M(session-db)、computer-use 70M(artifacts)；桌面端（bundle id com.openai.codex = ChatGPT.app）：AppSupport `Codex` 125M（Chromium 数据）、`com.openai.codex` |
| opencode v2.0.16 | agent | 已装（补语义） | `~/.local/share/opencode`：opencode.db 7.1G(session-db)、snapshot 158M(index)、bin 123M(runtime)、storage 94M(session)、log 59M；`~/.opencode`(runtime)；AppSupport `ai.opencode.desktop` 1.1G、`@opencode-ai`；`~/.config/opencode`、`~/.omo`(config) |
| ZCode | agent | 已装（CLI + ZCode.app 桌面端） | `~/.zcode`：cli/agents 1.2G（语义实现时逐目录确认）、cli/db 478M(session-db)、artifacts 188M(artifacts)、plugins 186M(plugins)、exec 100M(artifacts)、rollout 67M(session)、computer-use 249M(artifacts)、image-cache(cache)；AppSupport ZCode 61M |
| DeepSeek Harness（dsh 0.1.7-alpha.2） | agent | 已装（dsh CLI + DeepSeek Harness.app 桌面端） | `~/.dsh`：dsh-runtimes 361M(runtime)、sessions 84M(session)、profiles 27M(config)、storages 18M、attachments 16M(artifacts)；AppSupport `@deepseek-ai` 9.8M（桌面端）；`~/Library/Logs/DeepSeek Harness` |
| Antigravity（agy） | agent | 已装（agy CLI + Antigravity.app 桌面端 + Antigravity Tools.app 伴侣） | AppSupport `Antigravity` 113M（Electron：Cache/Code Cache/GPUCache 等）；`com.lbjlaq.antigravity-tools`；agy CLI 数据目录实现时确认（Open Questions） |
| Kimi Code（kimi） | agent | 已装（kimi CLI + Kimi Code.app 桌面端） | `~/.kimi-code` 386M：bin 353M(runtime)、sessions 28M(session)、search-index(index)、cache；AppSupport `kimi-code-app` 24M（桌面端） |
| WorkBuddy | agent | 已装（WorkBuddy.app 桌面端，腾讯 com.tencent.workbuddy.mac，Electron；CLI 未见） | `~/.workbuddy` 3.0G：binaries 998M(runtime)、logs 696M(logs)、plugins 302M(plugins)、blobs 296M(artifacts)、traces 266M(logs)、projects 204M（会话/工作区数据，实现时确认）、app 105M；AppSupport 仅 28K |
| u1s1 | agent | 已装（u1s1.app 桌面端，pi 生态发行版，主二进制 pi-agent-desktop） | `~/.u1s1`：agent/npm 461M(runtime)、agent/bin 6.7M、agent/sessions 5.1M(session) |
| mimocode | agent | 已装（mimo CLI，opencode fork） | `~/.mimocode`：bin 96M(runtime)、node_modules 56M(runtime)；`~/.local/share/mimocode`：mimocode.db 145M+wal 18M(session-db)、snapshot(index)、log；`~/.config/mimocode`：node_modules 56M(runtime)；`~/.cache/mimocode` 4.7M(cache) |
| pi | agent | 已装（brew） | `~/.pi`：agent/npm 375M(runtime)、agent/sessions 175M(session)、context-mode 27M(cache) |
| GitHub Copilot CLI | agent | 已装 | `~/.copilot` 84K(config) |
| Gemini | agent | CLI 已卸载，桌面端（Gemini 2.app）在 | `~/.gemini` 563M（孤儿归因为 gemini 条目）；AppSupport `Gemini 2` 12M、`cloud-code` |
| VS Code（+AI 扩展伴侣） | AI IDE | 已装 | AppSupport `Code` 1.8G：User/workspaceStorage 1.3G(workspace-data)、CachedData(cache)；`JoyCoder` 22M、`CodeBuddyExtension` 760K |
| Trae CN | AI IDE | **已卸载** | `~/.trae-cn` 348M + AppSupport `Trae CN` 844M（User 289M、CachedData 197M、logs 170M、ModularData 169M） |
| MarsCode | AI IDE | **已卸载** | `~/.marscode` 3.9G：`.ckg` 代码知识图谱索引(index) |

注：Antigravity.app 与 Kimi Code.app 均为 Electron 应用（AppSupport 为 Chromium 数据 / Resources 为 app.asar），非 VS Code fork——按 agent 分类；「Antigravity IDE」（VS Code fork 形态的 AI IDE）本机未安装。机器合计约 26GB（点目录约 21.8G + AppSupport 约 4.0G），其中孤儿残留（Trae CN + MarsCode + gemini CLI 数据）约 5.7G；WorkBuddy 的 logs 696M + traces 266M 是单体最大的可回收日志堆积。

## Goals / Non-Goals

- Goals：第一批编目与 AI 语义归因；session-db 识别与报告；已卸载 AI 工具从孤儿列表提升为有语义条目；AI IDE 目录豁免排除表。
- Non-Goals：session-db 的 vacuum/按会话删行（后续 change）；非本机工具编目（Cursor / Windsurf / Goose 等）；跨机器平台路径的逐一定制（目录规则按现有跨平台模式写，Windows 对应 `%APPDATA%`/`%LOCALAPPDATA%` 镜像 macOS 结构，不逐家验证）；产品叙事/README 重写。

## Decisions

### D1. AI 语义是 kind 扩展，不是新体系
沿用现有「标注 + tier + 用户裁量」模型。新 kind 只影响标签文案与 tier 默认值，不影响清理管道。这样 GUI/CLI 展示、trash、dry-run 全部零改动复用。

### D2. session-db 只报告、不可清理
opencode.db 是单个 7.1G SQLite，包含全部会话历史；`rm` 等于毁掉 resume 能力，行级清理需要理解 schema。v1 将 session-db 排除在 cleanable 之外，树下钻进入时展示说明（「会话数据库——包含会话历史，暂不支持直接清理」）。这是安全红线，也是与竞品 `rm -rf` 式清理的差异点。

### D3. 已卸载 AI 工具 = 编目条目 + 「已卸载」标注，而非孤儿
orphan-data 现有逻辑判定「数据根顶层目录未被任何已扫描工具认领」→ 孤儿。AI 编目后，Trae CN / MarsCode / gemini 数据会被认领。判定规则：编目条目的数据目录存在 且 可执行文件未发现 且 macOS `/Applications`（Windows 注册表卸载项，Linux 惯例路径）无对应应用 → 条目标注「已卸载」，数据全部按 kind 语义展示（index/runtime → 可回收；session → 用户数据）。孤儿列表因此只剩真正的无名残留。

### D4. 排除豁免用「优先级」表达，不改排除表数据
厂商排除表保持不动。新增一条优先级规则：AI 编目条目的目录归因在排除表判定之前完成，命中即豁免。这样非 AI 的 GUI 产品（CleanMyMac、Xshell 等）照旧被排除，现有测试不受影响。

### D5. 第一批范围以本机实测为准
只编目实测存在的工具，每条目录规则都有真实占用佐证；未实测的平台分支写保守默认（TierUser），实现时确认。ZCode `cli/agents` 1.2G 的语义在实现时逐目录确认后再定 kind。

### D6. 分类只有 agent / ai-ide 两类，桌面端归入 agent 条目
zcode、dsh 与 claude / codex / opencode / pi / copilot 同为「围绕模型的 agent harness（CLI 形态）」，不单设 harness 类（此前把 zcode 单列是分类标准不一致，已修正）。ZCode.app、OpenCode.app、DeepSeek Harness.app 等桌面端与对应 CLI 归入同一条目，避免同一工具被拆成两行；DeepSeek Harness 即范例：dsh CLI 数据 `~/.dsh` 504M + 桌面端 `@deepseek-ai` 9.8M 归属同一条目。通用 AI 助手桌面端（ChatGPT.app、kimi-desktop 358M）与配置伴侣（CC Switch）不属于编码工具，v1 不编目。

## Risks / Trade-offs

- 会话数据误删代价高 → session/checkpoint 一律 TierUser，默认走内置回收站，dry-run 保持。
- 各家目录 schema 迭代快（如 opencode 正从平面文件迁往 SQLite）→ 编目是纯数据表，新增/修正成本低；编目注释里记录实测日期与占用，便于复核。
- VS Code 是通用 IDE 不是纯 AI 工具 → 以「AI IDE（宿主）」类别编目，kind 聚焦 AI 相关与通用可回收数据（workspace-data/cache），不把整个 Code 目录标成 AI 占用。

## Migration Plan

纯新增能力 + 一条豁免规则，无数据迁移。旧扫描结果缓存按现有 scan-change-detection 机制失效重扫。

## Open Questions

- ZCode `cli/agents` 的构成（实现时确认，可能是各 agent 会话工作目录）。
- agy（Antigravity CLI）的数据目录位置：AppSupport/Antigravity 113M 为 Electron 桌面端数据，agy CLI 是否共用该目录待确认。
- u1s1 与 pi 的关系：u1s1.app 是 pi 生态的桌面发行版（主二进制 pi-agent-desktop），其 CLI 形态是否直接复用 pi 条目（`~/.pi`）还是仅用 `~/.u1s1`，编目归属实现时确认。
- `~/.gemini` 在 Gemini 2 桌面端仍在用时是否算「已卸载」（桌面端与 CLI 是否共用数据目录）——实现时以目录写入活动为准。
