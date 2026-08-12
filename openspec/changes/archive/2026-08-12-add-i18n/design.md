## Context

现状（详见 proposal.md - Why）：全部用户可见文案硬编码简体中文，分布在前端（main.ts ~111 行、index.html ~34 行、style.css 4 处）、Go 原生菜单/About、CLI 输出、后端各包错误串（~15-20 处）与规则表工具描述（28 条）。前端为单文件零依赖架构，无现成 i18n 框架。语言决策已在探索阶段锁定：全量一次完成（GUI + CLI + 原生菜单 + 后端错误 + 工具描述），切换即时生效（原生菜单除外）。

## Goals / Non-Goals

**Goals:**
- 单一语言文件来源，前端（vite）与 Go（go:embed）双端共享，杜绝双份翻译漂移。
- 默认语言下行为与现状逐字一致（回归保护：现有测试断言继续通过）。
- CLI `--json` 机器契约零破坏。
- 零第三方依赖（延续项目风格）。

**Non-Goals:**
- 不翻译注释、代码标识符、工具名/专有名词、SAFE/USER 分级标签、单位（KB/MB/GB）、应用名 "CLI Analyzer"。
- 不做运行时重建 macOS 原生菜单（Wails 不支持）——标签下次启动生效。
- 不引入富文本/ICU 消息格式（`{name}` 插值 + 最小复数足够）。
- 不做 RTL（三种语言均非 RTL）。

## Decisions

### D1. 语言文件单一来源：`internal/i18n/locales/*.json`，Go 嵌入 + 绑定下发

`internal/i18n/locales/{zh-CN,zh-TW,en}.json`，扁平键 `Record<string,string>`。Go 经 `go:embed locales/*.json` 嵌入二进制；前端不直接 import JSON，而是在 `init()` 经 `GetTranslations(locale)` 绑定拿到同一份字典后自行渲染（t() 读内存字典，语言切换即时）。

- 理由：零重复。`go:embed` 不允许 `..` 跨目录，无法从 `internal/i18n` 嵌入 `frontend/src/locales`；若前端直接 import 仓库其他目录的 JSON，vite dev server 的 `fs.allow` 需要额外配置。由 Go 嵌入并经绑定下发，单进程内字典天然一致，且避免双端解析同一份文件的机制分叉。
- 键命名：按域分组前缀，如 `ui.rescan`、`menu.checkUpdates`、`err.trashRestore`、`tool.desc.npm`、`cli.scan.header`。

### D2. 语言解析链（config → 平台探测 → 回退 zh-CN）

```
explicit(config.language) ──┬─ 非 auto → 直接用
                            └─ auto ──▶ DetectSystem() 平台探测
                                         ├─ macOS:   defaults read -g AppleLanguages 首项
                                         ├─ Windows: GetUserDefaultUILanguage (syscall)
                                         ├─ Linux:   LC_ALL / LC_MESSAGES / LANG
                                         └─ 失败/不支持 → zh-CN
```

- `internal/i18n/DetectSystem()` 返回 `zh-CN | zh-TW | en | ""`（映射：`zh-Hans*`→zh-CN、`zh-Hant*`/`zh-TW`→zh-TW、`en*`→en，其余空）。
- GUI 细化：WebView 的 `navigator.language` 比 Go 环境探测更贴近真实系统 UI 语言。前端在 `init()` 解析出生效语言后调 `SetLanguage(locale)` 绑定，Go 侧 `i18n.SetLocale` 同步——后端错误/弹窗/菜单随之一致。
- CLI 无 WebView：直接 config + DetectSystem。
- 替代：仅靠 `navigator.language`（GUI 可靠但 CLI 无解）或仅靠 Go 探测（macOS GUI 用户常无 LANG，会误回退 zh-CN）。双通道互补。

### D3. 后端错误本地化：方案 A（包内直接 `i18n.T`）

各后端包在错误创建处调用 `i18n.T("err.<domain>.<name>", {"path": p})`，占位符 `{path}` 插值。全局 active locale 在启动时设置（GUI 经 SetLanguage 握手，CLI 经 Resolve）。

- 默认 locale 为 zh-CN 且翻译内容与现状逐字一致 → cleaner/trash/updater 等 4 个断言中文串的测试文件无需改动。
- 备选（方案 B）：类型化错误码 + 边界翻译，后端无语言概念——架构更纯，但 45 处错误返回需重构为结构化类型、测试改写为断言错误码，改造面翻倍。单进程单语言场景下收益为理论值，否决；记录于本文档供未来参考。
- 注意：`internal/i18n` 依赖 `internal/platform`（探测）；config 依赖 platform；后端包依赖 i18n——无循环依赖。i18n 不依赖 config（解析职责在调用方，i18n 只持 locale 状态与翻译）。

### D4. 切换生效：前端即时 + 原生菜单下次启动（决策①b）

- 前端：语言切换 → 立即以新 locale 重渲染（`t()` 读取响应式 locale 状态）+ 保存 config + `SetLanguage` 同步 Go 侧（此后后端错误/运行时弹窗即新语言）。
- About 弹窗是运行时生成的 → 即时切换（含 macOS 原生 MessageDialog，其文案取当前 locale）。
- 唯一延迟项：macOS 原生菜单标签（`buildMenu()` 启动时构建）。保存时 toast 提示「原生菜单将在重启后生效」。
- Windows/Linux：自绘 HTML 菜单走 `data-i18n` → 即时生效，无延迟项。

### D5. CLI `--json` 机器契约保护

`--json` 输出的键名与数据字段值（tier/kind 等）保持英文与现状完全一致，不经 i18n。仅人类可读输出（非 json 模式）走 `i18n.T`。

- 实现约束：JSON 序列化路径（`scan.go`、`update.go` 等的 `json.Marshal` 分支）不得接触 `T()`；表头/提示/错误摘要（`output.go`、各 `runXxx`）使用 `T()`。
- 测试：新增断言 `scan --json` 与 `update check --json` 输出在三种语言下字节一致。

### D6. 插值与最小复数

- 插值：`T(key, map[string]any{"n": 5, "path": "/x"})`，占位符 `{n}`；缺失参数保留原样，多余参数忽略。
- 复数：键后缀 `_one` / `_other`。`T` 按 locale 规则选：en 用 `n==1 ? _one : _other`；zh-CN/zh-TW 恒用 `_other`（文件内两键同值，保持键集对齐）。
- 前端 `lib/i18n.ts` 与 Go `i18n.T` 实现同一规则（双端行为一致性由测试对齐）。

### D7. 前端零依赖改造

- `frontend/src/lib/i18n.ts`：导入三份 JSON，导出 `setLocale(locale)`、`t(key, vars)`、`locale` 状态。
- 静态 HTML：`index.html` 标签加 `data-i18n="key"`（title 等属性用 `data-i18n-title`），`init()` 时统一扫描替换；`menuBar`（Windows 自绘菜单）同步处理。
- `main.ts` 全部模板字符串改为 `t()` 调用；动态渲染函数在语言切换后重跑（`renderAll()` 或对已开弹窗重渲染）。

### D8. 配置扩展

`config.json` 新增 `language` 字段：`"auto" | "zh-CN" | "zh-TW" | "en"`，默认 `"auto"`。normalize 兜底非法值 → auto。旧配置无此字段 → auto（跟随系统），向后兼容。首选项面板新增语言下拉（选项文案本身随当前语言显示）。

### D9. 工具描述：改为 i18n 键

`internal/rules/metadata.go` 的描述字段改为键引用（`tool.desc.<name>`），URL 与结构不变；显示端（GUI 详情 / CLI）经 `i18n.T` 取词。28 条描述进入语言文件，与其余字符串同源、同 parity 校验。

- 理由：描述是字符串而非结构数据，放入语言文件后一处维护、一个测试兜底；避免 metadata 里出现 3 语言 × 28 的结构化字段。
- 无描述的工具（动态发现）按缺失键处理 → 返回空，UI 不显示描述（现状行为）。

### D10. 日期本地化（GUI）

前端 `fmtTime`/趋势日期改用 `toLocaleDateString(locale)`（或 `toLocaleString`）按当前语言区域格式化。Go CLI 侧日期输出保持现状（机器可读 ISO 风格，不属于 GUI 场景，spec 已限定 GUI）。

### D11. 翻译完整性测试（parity）

`internal/i18n/i18n_test.go`：加载三份文件，断言键集完全一致（无缺失、无多余、无空值）。CLI JSON 契约测试（D5）双端行为一致性测试（D6）一并纳入。

### D12. zh-TW 术语表（防机械转换陷阱）

繁体中文采用**真翻译**而非简转繁。关键术语对照（写入设计备忘，翻译时遵循）：

| 简体 | 繁体 | 英文 |
|---|---|---|
| 首选项 | 偏好設定 | Preferences |
| 扫描 | 掃描 | Scan |
| 清理 | 清理 | Clean |
| 回收站 | 資源回收筒 | Trash |
| 恢复 | 還原 | Restore |
| 可清理 | 可清理 | Cleanable |
| 占用 | 佔用 | Usage |
| 趋势 | 趨勢 | Trends |
| 提醒 | 提醒 | Reminder |
| 更新 | 更新 | Update |
| 忽略该版本 | 略過此版本 | Ignore this version |
| 内置 | 內建 | Built-in |

## Risks / Trade-offs

- main.ts 111 行机械改造漏译 → 系统性逐行替换 + parity 测试（只保证键对齐，不保证覆盖）+ 验收时三种语言各过一遍关键路径。
- 首次启动 macOS 菜单语言可能与前端不一致（探测路径不同）→ macOS 用 AppleLanguages 对齐；残余差异下次启动消失，属决策①b 已知行为。
- 后端包引入 i18n 全局状态 → 单进程单 locale，启动期一次性设置，无并发写；测试可用 `t.SetLocale` 隔离。
- zh-TW 翻译质量 → D12 术语表约束；英文文案由翻译产出并人工复查。
- `data-i18n` 与动态渲染双路径可能漏掉新增字符串 → 约定：新字符串必须走语言文件，代码评审检查点。

## Migration Plan

1. 配置向后兼容：旧 config.json 无 `language` → normalize 为 auto，无迁移动作。
2. 行为回归：默认 zh-CN 下所有文案与现状逐字一致；既有测试（含断言中文串）保持绿色。
3. 回滚：i18n 为纯增量改造，config 无破坏性变更；回滚 = revert 提交。
4. 发布：随下一版本（v0.3.1+）发布，语言文件嵌入二进制，无外部依赖。

## Open Questions

无阻塞项。以下留待后续，不影响本方案落地：
- CLI 日期/数字按 locale 格式化（当前保持 ISO，v2 可议）。
- 更多语言（日/韩/法…）只需新增语言文件 + 探测映射，机制已就绪。
