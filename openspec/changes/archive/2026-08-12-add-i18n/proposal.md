## Why

应用目前所有用户可见文案（GUI、CLI、原生菜单、后端错误、工具描述）都硬编码为简体中文。面向更广用户群体，需要语言国际化：默认跟随系统语言，也支持在设置中手动选择。第一阶段支持简体中文、繁体中文、英文三种语言。

## What Changes

- 新增 `internal/i18n/` 包：语言解析（显式配置 → 平台探测 → 回退 zh-CN）、`T(key, args)` 翻译函数、复数规则、`go:embed` 加载语言文件。
- 语言文件单一来源：`frontend/src/locales/{zh-CN,zh-TW,en}.json`，前端（vite 打包）与 Go（go:embed）双端共享同一份内容。
- 配置扩展：`config.json` 新增 `language`（`auto` | `zh-CN` | `zh-TW` | `en`），默认 `auto`（跟随系统）；首选项面板新增语言下拉。
- 切换生效方式：GUI 前端即时重渲染 + `SetLanguage` 绑定同步 Go 侧；macOS 原生菜单标签（启动时构建，无法热重建）下次启动生效。
- 后端错误字符串全部接入 i18n（方案 A：各包内直接调 `i18n.T`，默认 locale 为 zh-CN 保证既有测试断言不变）。
- 规则表工具描述（`internal/rules/metadata.go` 约 28 条）翻译为三种语言。
- CLI 人类可读输出按当前语言渲染；`--json` 输出键名与值保持英文不变（机器契约，脚本不破）。
- 前端零依赖实现：小型 `t()` 函数 + 最小复数规则（en: n==1 单数）+ `data-i18n` 静态 HTML 扫描替换。
- parity 测试：三个语言文件键集全对齐，防止漏翻。

## Capabilities

### New Capabilities
- `localization`: 语言解析与翻译基础设施。覆盖：语言解析链（配置/平台探测/回退）、GUI 即时切换与 Go 侧同步、CLI 本地化与 JSON 契约保护、后端错误本地化、工具描述本地化、配置项与设置入口、复数与日期格式化、翻译完整性校验。

### Modified Capabilities
<!-- 无：既有 capability（trash-recycle-bin、usage-trends、app-update）的需求不变，仅实现层面本地化。 -->

## Impact

- 新增包：`internal/i18n/`（解析器 + T() + 复数 + embed 加载）；新增 `frontend/src/locales/*.json`（3 个语言文件）。
- 前端：`main.ts`（111 行中文 → `t()` 调用）、`index.html`（静态标签 → `data-i18n`）、`style.css`（4 处）。
- Go：`main.go`（原生菜单/About）、`gui/service.go`（错误/toast + `SetLanguage`/`GetLanguage` 绑定）、`internal/cli/*`（usage/表头/提示）、后端包（scanner/cleaner/trash/updater/history 等错误串接入 i18n）、`internal/rules/metadata.go`（描述数据结构改为按语言取）。
- 配置：`internal/config/` 新增 `language` 字段与 normalize 兜底。
- 测试：既有断言中文串的 4 个测试文件在默认 zh-CN 下保持通过；新增 i18n 单元测试与 parity 测试。
- 无新增第三方依赖（保持零依赖风格）。
