## 1. 编目与语义 kind

- [x] 1.1 在 `internal/rules` 新增 AI 语义 kind 常量与 tier 映射（session / session-db / index / runtime / plugins / artifacts / logs / telemetry / checkpoint / workspace-data）
- [x] 1.2 新增 AI 编目表（类别字段：agent / ai-ide）：pi、copilot、zcode（含 ZCode.app）、deepseek-harness（dsh + DeepSeek Harness.app）、antigravity（agy CLI + Antigravity.app + Antigravity Tools.app）、kimi-code（kimi CLI + Kimi Code.app）、workbuddy（WorkBuddy.app，腾讯）、u1s1（u1s1.app，pi 生态发行版）、mimocode（mimo CLI，opencode fork）、VS Code、gemini（含桌面端目录）、Trae CN（已卸载残留）、MarsCode（已卸载残留）；codex 条目补桌面端数据目录（AppSupport/Codex，承载于 ChatGPT.app）；每条注释记录实测日期与占用
- [x] 1.3 为现有 claude / codex / opencode 条目补充 AI 语义 kind（transcripts/projects → session；logs_2.sqlite → session-db；.ckg 类 index；packages/bin → runtime 等）
- [x] 1.4 `rules_test.go`：每条编目的路径展开、tier 与 kind 断言（macOS 为主，Windows 镜像路径断言结构）

## 2. 扫描与归因

- [x] 2.1 `internal/scanner`：归因结果携带 AI kind；kind 标签文案走 i18n 管道
- [x] 2.2 session-db 目录不进入 cleanable 集合；树下钻遇到 session-db 子项时展示「不支持直接清理」说明
- [x] 2.3 已卸载 AI 工具判定：数据目录存在 + 可执行文件缺失 + /Applications 无对应应用 → 归因为编目条目并标注「已卸载」（复用 orphan-data 发现，结果提升到工具列表）
- [x] 2.4 AI IDE 目录豁免：编目条目的 AppSupport 归因先于厂商排除表判定
- [x] 2.5 归因测试：opencode.db → session-db 且不可清理；`.marscode/.ckg` → index(TierSafe)；Trae CN 残留 → 已卸载条目；`~/Library/Application Support/Code` → workspace-data

## 3. 展示与 i18n

- [x] 3.1 新 kind 标签文案：zh-CN / zh-TW / en 三语补齐
- [x] 3.2 「已卸载」标注文案与工具详情页展示（三语）
- [x] 3.3 工具列表展示类别字段（agent / ai-ide），沿用现有列表管道，不做新分组视图

## 4. 验证

- [x] 4.1 本机实机扫描验收：19G 总量口径对齐（opencode 7.5G / marscode 3.9G / zcode 2.4G / Trae 残留 1.2G / Code 1.8G 等均带语义标签呈现）
- [x] 4.2 dry-run 验收：session-db 不出现在清理计划；index/runtime 项可入回收站；孤儿列表不再包含 Trae CN / MarsCode / .gemini
- [x] 4.3 回归：non-cli-exclusion 现有用例（Xshell / mihomo / iterm2 等）全部通过
