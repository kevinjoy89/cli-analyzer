## Why

通用 CLI 磁盘分析对 AI 编码工具的呈现过于粗糙：目录只有 cache/config/data/install 四类标注，无法区分「可再生代码索引」「捆绑运行时」与「用户会话数据」——而这正是 AI 工具链（agent harness / AI IDE）占用的三种主要形态，处置策略截然不同。本机实测（2026-09-30）AI 工具链合计约 26GB：opencode 单个 SQLite 会话库 7.1G、WorkBuddy 3.0G（其中 logs 696M + traces 266M 为纯日志堆积）、MarsCode 残留索引 3.9G、ZCode 2.4G、Trae CN 残留 1.2G、VS Code workspaceStorage 1.3G、codex 1.2G、dsh 运行时与会话 0.5G、pi 578M、mimocode 0.4G、Kimi Code 0.4G、u1s1 0.5G。其中约 5G（Trae CN / MarsCode）来自**已卸载**应用的孤儿数据。产品重新定位的第一步：把本机实际存在的 AI 工具作为第一批一等公民编目对象，用 AI 语义分类替代通用标注。

## What Changes

- 新增 AI 工具编目（`internal/rules/`）：第一批覆盖本机实测存在的 16 个对象，按类别（agent / ai-ide）编目。agent 指 agent harness——以 CLI 或桌面端为形态（zcode、dsh 与 claude/codex/opencode/pi/copilot 同类），其桌面端应用（ZCode.app、OpenCode.app、DeepSeek Harness.app、WorkBuddy.app、u1s1.app 等）归入同一工具条目；每条编目含目录规则 + AI 语义 kind + tier（与现有 `dd()` 模式一致）：
  - **agent**：claude（已有条目，加 AI 语义标注）、codex（已有；桌面端承载于 ChatGPT.app，AppSupport/Codex 归入该条目）、opencode（已有，含 OpenCode.app 桌面端）、pi（新增，brew 安装）、copilot（GitHub Copilot CLI，新增）、zcode（新增，含 ZCode.app 桌面端）、deepseek-harness（新增：dsh CLI 0.1.7-alpha.2，brew 安装 + DeepSeek Harness.app 桌面端）、antigravity（新增：agy CLI + Antigravity.app 桌面端 + Antigravity Tools.app 伴侣；「Antigravity IDE」为另一产品，本机未安装）、kimi-code（新增：kimi CLI + Kimi Code.app 桌面端，均经实证为 agent 形态而非 VS Code fork）、workbuddy（新增：腾讯 agent 桌面端 com.tencent.workbuddy.mac）、u1s1（新增：pi 生态桌面发行版 io.u1s1.app）、mimocode（新增：opencode fork，mimo CLI）
  - **ai-ide**（GUI）：VS Code（含 AI 扩展伴侣 JoyCoder / CodeBuddyExtension）；已卸载残留 Trae CN、MarsCode
  - **已卸载残留**（孤儿归因）：Gemini CLI 数据（.gemini，CLI 已卸载，Gemini 2.app 桌面端在用）、Trae CN、MarsCode
- 发现但明确排除在 v1 之外：kimi-desktop（358M，Kimi 助手聊天桌面端，非编码工具）、CC Switch（AI 工具配置切换器）——留待后续批次决策；ChatGPT.app 仅将 Codex 相关数据目录纳入 codex 条目，其聊天数据不纳入。
- 新增 AI 语义 kind 集合：`session`（会话/转录）、`session-db`（会话数据库）、`index`（可再生代码索引/知识图谱）、`runtime`（捆绑运行时/npm 包）、`plugins`、`artifacts`（agent 产物/exec/上传）、`logs`、`telemetry`、`checkpoint`（文件历史/回滚点）、`workspace-data`（IDE 工作区存储）。每个 kind 映射到现有 tier 体系（session/checkpoint → TierUser；index/runtime/logs/telemetry → TierSafe；其余 TierUser）。
- `session-db` 仅识别与报告（v1）：SQLite 会话库（opencode.db 7.1G、zcode db 478M、codex 日志库）MUST NOT 列为可直接清理项，展示占用与含义说明；语义级清理（vacuum/按会话删行）留给后续 change。
- 已卸载 AI 工具的孤儿归因：编目中某 AI 工具的数据目录存在、但其可执行文件与 GUI 应用均缺失时，SHALL 归因为该工具条目（标注「已卸载」），而非落入无语义的孤儿数据列表。复用 orphan-data 的发现逻辑，仅改变展示归属。
- non-cli-exclusion 修改：AI 编目工具优先于厂商排除表——AI IDE 的数据目录（如 `~/Library/Application Support/Trae CN`、`Code`）不再因「GUI 应用」被排除归因；排除表对非 AI 的 GUI 产品照旧生效。
- i18n：全部新 kind 标签与「已卸载」文案补齐 zh-CN / zh-TW / en。

## Capabilities

### New Capabilities
- `ai-tool-catalog`: AI 编码工具（agent harness / AI IDE）的编目、语义归因与已卸载残留识别。覆盖：第一批编目清单（本机实测 16 个对象）、AI 语义 kind 与 tier 映射、session-db 仅报告、已卸载 AI 工具孤儿归因、AI IDE 目录归因豁免排除表。

### Modified Capabilities
- `non-cli-exclusion`: 新增「AI 编目工具数据目录豁免」要求——AI 编目工具的数据目录归因 MUST NOT 被厂商排除表拦截；厂商排除表对非 AI 的 GUI 产品照旧生效。

## Impact

- 修改包：`internal/rules/`（AI 编目与 kind 常量，tools.go 现有 claude/codex/opencode 条目补充 AI 语义标注）、`internal/scanner/`（归因时携带 AI kind；已卸载 AI 工具的孤儿归因）、`internal/i18n/`（新文案）。
- 展示层：GUI 与 CLI 沿用现有 kind 标签管道展示新 kind；工具列表新增类别（agent / ai-ide）字段展示，v1 不做独立分组视图。
- 复用：现有 tier 体系、orphan-data 发现逻辑、`dd()` 编目模式、trash/dry-run 安全机制（session/index/runtime 的清理走现有管道）。
- 无新增第三方依赖；无网络行为。
- 后续（不在本 change）：session-db 的 vacuum 与按会话清理、更多 AI 工具编目（Cursor / Windsurf / Goose 等）、产品叙事与 README 重写。
