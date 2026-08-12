## 1. i18n 基础设施

- [x] 1.1 新建 `internal/i18n/i18n.go`：`go:embed ../../frontend/src/locales/*.json` 加载三份语言文件，`SetLocale`/`ActiveLocale`（非法值回退 zh-CN）
- [x] 1.2 实现 `T(key string, args ...map[string]any) string`：`{name}` 占位符插值；缺失键返回键名本身（便于发现漏翻）
- [x] 1.3 实现复数选择：键后缀 `_one`/`_other`，en 按 `n==1` 选单数，zh-CN/zh-TW 恒 `_other`（文件内同值）
- [x] 1.4 实现 `DetectSystem() string` 平台探测：macOS `defaults read -g AppleLanguages`、Windows 系统 UI 语言（syscall）、Linux LC_ALL/LC_MESSAGES/LANG；`zh-Hans*`→zh-CN、`zh-Hant*`/`zh-TW`→zh-TW、`en*`→en，其余空
- [x] 1.5 实现 `Resolve(explicit string) string`：显式语言直接返回，auto 走 DetectSystem，探测失败回退 zh-CN
- [x] 1.6 i18n 单元测试：插值、复数、缺失键、locale 切换、Resolve 各分支

## 2. 语言文件与 parity 测试

- [x] 2.1 创建语言文件 zh-CN.json：迁移现有全部中文文案（键集即基准，默认行为不变）
- [x] 2.2 创建 zh-TW.json：按设计 D12 术语表真翻译（非简转繁）
- [x] 2.3 创建 en.json：英文翻译（插值/复数占位符合英文语法）
- [x] 2.4 parity 测试：三文件键集完全一致（无缺失/多余/空值），任一语言不同步即失败
- [x] 2.5 确认单来源：语言文件位于 `internal/i18n/locales/`，Go 经 `go:embed` 嵌入，前端经 GUI 绑定 `GetTranslations` 获取同一份字典（go:embed 不允许 `..` 跨目录，故不放在 frontend/ 下）

## 3. 配置扩展

- [x] 3.1 `internal/config/config.go` 新增 `Language string`（`auto`|`zh-CN`|`zh-TW`|`en`，默认 auto）；normalize 兜底非法值
- [x] 3.2 config 测试：默认 auto、旧配置兼容、保存回读
- [x] 3.3 前端首选项面板新增语言下拉（「跟随系统」「简体中文」「繁體中文」「English」，选项文案随当前语言）；保存写入 config

## 4. 前端本地化

- [x] 4.1 新建 `frontend/src/lib/i18n.ts`：导入三份 JSON、`setLocale`/`t(key, vars)`、复数规则、`formatDate(locale)`；导出当前 locale 状态
- [x] 4.2 `index.html` 静态标签加 `data-i18n`（含 `data-i18n-title` 属性变体）；`init()` 统一扫描替换；Windows 自绘菜单（menuBar）同步处理
- [x] 4.3 `main.ts` 全部用户可见模板字符串改为 `t()` 调用（约 111 行：按钮/弹窗/toast/首选项/趋势/回收站/关于/更新弹窗/状态栏/铃铛）
- [x] 4.4 语言切换即时生效：切换后重渲染已打开界面；`SetLanguage` 同步 Go 侧
- [x] 4.5 `fmtTime`/趋势日期按 locale 格式化（`toLocaleDateString`）

## 5. Go GUI 与原生菜单

- [x] 5.1 `gui/service.go` 新增 `SetLanguage(locale string)`（调 `i18n.SetLocale`）与 `GetLanguage() string` 绑定
- [x] 5.2 `main.go` `buildMenu()` 菜单标签改 `i18n.T`（首选项/检查更新/About/帮助等）；About 弹窗文案取当前 locale
- [x] 5.3 GUI 启动解析：config → auto 时 DetectSystem；前端 `init()` 用 `navigator.language` 细化并 SetLanguage 握手
- [x] 5.4 `gui/service.go` 内错误/toast 字符串改 `i18n.T`（约 8 处）

## 6. CLI 本地化

- [x] 6.1 `internal/cli/run.go` usage 与版本输出改 `i18n.T`；CLI 启动时 Resolve+SetLocale
- [x] 6.2 `internal/cli/output.go` 表格表头/合计/统计行改 `i18n.T`
- [x] 6.3 `internal/cli/` 各子命令（scan/clean/cache/trash/trends/update）提示与错误改 `i18n.T`
- [x] 6.4 JSON 契约保护：`--json` 分支不接触 `T()`；新增测试断言三种语言下 `scan --json` 与 `update check --json` 输出字节一致

## 7. 后端错误本地化

- [x] 7.1 `internal/trash/` 错误串改 `i18n.T`（约 10 处，含插值参数）
- [x] 7.2 `internal/scanner/`、`internal/cleaner/` 错误串改 `i18n.T`
- [x] 7.3 `internal/updater/` 错误串改 `i18n.T`（check/download/checksum/source/open 等）
- [x] 7.4 `internal/history/`、`internal/disk/` 等其余包错误串改 `i18n.T`
- [x] 7.5 回归：默认 zh-CN 下既有测试（断言中文串的 4 个测试文件）全部保持通过

## 8. 工具描述本地化

- [x] 8.1 `internal/rules/metadata.go` 描述字段改为 `tool.desc.<name>` 键引用（URL 与结构不变）
- [x] 8.2 28 条描述 × 3 语言写入语言文件（按 D12 术语表）
- [x] 8.3 GUI 详情与 CLI 显示端经 `i18n.T` 取描述；无描述工具返回空（行为不变）

## 9. 端到端验证

- [x] 9.1 三语言走查：zh-CN / zh-TW / en 各过一遍 GUI 关键路径（扫描/清理/回收站/趋势/更新/首选项）
- [x] 9.2 语言切换即时性：切换后未重启，前端文案与后端错误均已换语言；macOS 原生菜单保持旧语言并提示
- [x] 9.3 CLI：三语言下 `scan` 表头与提示正确；`--json` 输出与语言无关
- [x] 9.4 回归：默认 zh-CN 下全量 go test + 前端 tsc/vitest 通过
- [x] 9.5 更新 README（语言支持说明）
