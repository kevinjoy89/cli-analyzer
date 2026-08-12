## Purpose

为应用引入语言国际化：所有用户可见文案按当前语言渲染，默认跟随系统语言并可在设置中手动选择，第一阶段支持简体中文、繁体中文、英文。语言解析、切换与翻译完整性均有明确行为约束，且不破坏 CLI 机器可读输出的契约。

## ADDED Requirements

### Requirement: 语言解析

系统 SHALL 按以下优先级解析当前语言：用户显式配置的语言 → 系统语言探测 → 回退简体中文。系统探测 SHALL 覆盖 macOS（AppleLanguages）、Windows（系统 UI 语言）、Linux（LC_ALL/LC_MESSAGES/LANG 环境变量）。GUI 场景下，系统探测 SHALL 结合 WebView 的 `navigator.language` 细化结果。

#### Scenario: 显式配置优先
- **WHEN** 用户在设置中选择「繁體中文」
- **THEN** 应用所有界面与输出使用繁体中文，忽略系统语言

#### Scenario: 跟随系统
- **WHEN** 用户未手动设置语言，且系统语言为英文
- **THEN** 应用使用英文界面；系统语言不在支持列表（如日文）时回退简体中文

#### Scenario: 探测失败回退
- **WHEN** 平台探测无法确定系统语言
- **THEN** 应用使用简体中文，且行为与现状一致

### Requirement: 语言设置

系统 SHALL 在首选项面板提供语言选择，选项为「跟随系统」「简体中文」「繁體中文」「English」。所选语言 SHALL 持久化到配置文件，后续启动生效。默认值为「跟随系统」。

#### Scenario: 保存语言选择
- **WHEN** 用户在首选项选择「English」并保存
- **THEN** 配置文件中记录 `language: "en"`，重启后仍为英文

#### Scenario: 恢复跟随系统
- **WHEN** 用户将语言改回「跟随系统」
- **THEN** 配置文件记录 `language: "auto"`，应用重新按系统语言解析

### Requirement: 语言切换生效

系统 SHALL 在 GUI 内切换语言后即时更新所有前端界面文案，且 SHALL 同步后端用于错误提示与弹窗的语言。macOS 原生菜单栏标签 SHALL 在下次启动时生效（原生菜单启动时构建、无法运行时重建）；该限制 SHALL 在首选项保存时提示用户。

#### Scenario: 前端即时切换
- **WHEN** 用户在首选项选择新语言并保存
- **THEN** 前端所有文案立即切换为新语言，无需重启

#### Scenario: 原生菜单延迟生效
- **WHEN** 用户在 macOS 上切换语言并保存
- **THEN** 提示「部分原生菜单将在重启后生效」，菜单标签保持旧语言直到下次启动

### Requirement: GUI 本地化

系统 SHALL 将所有 GUI 用户可见文案（按钮、弹窗、提示、首选项、趋势、回收站、关于、更新弹窗、状态栏、静态 HTML 标签）按当前语言渲染。日期时间 SHALL 按当前语言区域格式化。

#### Scenario: 静态标签本地化
- **WHEN** 界面语言为 English 且界面加载
- **THEN** 静态按钮（如"重新扫描"）显示为英文对应文案

#### Scenario: 动态文案本地化
- **WHEN** 界面语言为 English 且触发清理确认弹窗
- **THEN** 弹窗文案与插值参数（数量、大小）均按英文语法渲染

### Requirement: CLI 本地化与机器契约保护

系统 SHALL 将 CLI 人类可读输出（usage、表格表头、提示、进度、错误摘要）按当前语言渲染。CLI 的 `--json` 输出 SHALL 保持键名与值结构完全不变（机器契约），仅语言无关。CLI 当前语言 SHALL 由配置与系统探测解析（无 WebView 可用）。

#### Scenario: 人类输出本地化
- **WHEN** 当前语言为 English 且执行 `cli-analyzer scan`
- **THEN** 表头与提示以英文显示

#### Scenario: JSON 契约不变
- **WHEN** 当前语言为任意支持语言且执行 `cli-analyzer scan --json`
- **THEN** JSON 键名与数据字段值与此前版本完全一致，不随语言变化

### Requirement: 后端错误本地化

系统 SHALL 将后端各包（scanner、cleaner、trash、updater、history 等）返回给用户的错误与提示字符串按当前语言渲染。默认语言为简体中文时，错误文案 SHALL 与现状逐字一致。

#### Scenario: 错误按当前语言显示
- **WHEN** 当前语言为 English 且回收站恢复失败
- **THEN** 错误提示以英文显示（含原路径等插值参数）

#### Scenario: 默认语言下行为不变
- **WHEN** 用户未设置语言且系统为简体中文环境
- **THEN** 后端错误文案与未国际化版本逐字一致

### Requirement: 工具描述本地化

系统 SHALL 将规则表中的工具描述（约 28 条）按当前语言渲染。工具名称与专有名词（如 "Anthropic"、"Homebrew"）SHALL 保持原文不翻译。

#### Scenario: 描述按语言显示
- **WHEN** 界面语言为 English 且查看 npm 工具详情
- **THEN** 描述显示为英文对应文案，工具名 "npm" 保持原样

### Requirement: 翻译完整性

系统 SHALL 保证所有支持语言的翻译文件键集完全一致。系统 SHALL 在构建期测试中校验三语言键集对齐，任一语言缺失或多余键 SHALL 使测试失败。

#### Scenario: 键集校验
- **WHEN** 某语言文件新增了翻译键而未同步其他语言
- **THEN** 完整性测试失败，阻止合入

### Requirement: 插值与复数

系统 SHALL 支持翻译键的插值参数（如数量、大小、路径）。复数规则 SHALL 按语言区分：英文按 `n==1` 使用单数形式，中文不区分单复数。

#### Scenario: 英文复数
- **WHEN** 英文界面显示 "1 tool" 与 "5 tools"
- **THEN** 分别使用单数与复数形式

#### Scenario: 中文无复数
- **WHEN** 中文界面显示数量
- **THEN** 不因数量变化改变句式
