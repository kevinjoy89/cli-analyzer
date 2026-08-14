## Why

工作区积累了一批与既有变更（孤儿数据/探测）同批开发、但超出其能力范围的细节调整：Node.js 运行时家族在 Windows 等布局下被扫成多个互不相干的小工具，规则表别名从未进入 JSON、寄居二进制冒充别名、可清理子项类型一律继承父项（`~/.npm/_logs` 被标成 cache），回收站列表信息密度低且无类型标注。这些行为改动需要一个独立变更归档。

## What Changes

- **Node.js 运行时家族合并**：node/npm/npx/corepack/node-gyp 归并为单一工具 `nodejs`（新安装来源 `nodejs`、JSON 新增 `family` 字段），适用于同目录布局（Windows 官方安装器 / nvm-windows / Volta / scoop、unix 独立放置）与 node_modules 布局（nvm/fnm）；brew 公式 node 保持独立；版本探测主二进制固定为 node
- **别名契约修正**：规则表 curated 别名（claude→claude-code、pip→pip3、hf 等）并入 JSON aliases；家族合并工具的别名用归一化命令名（Windows 剥 `.exe/.cmd` 扩展名）；寄居二进制（kimi 自带的 rg/fd 等）不再冒充别名，仅出现在二进制列表；前端按 `family` 区分「包含工具」与「别名」，别名过多不展示
- **可清理子项精确类型**：`SubEntry` 新增 `kind` 字段，可识别子项（`_logs`/`*.log` → logs）使用精确类型而非继承父项；清理入库与回收站展示使用该精确类型，旧缓存无字段时回退父项
- **回收站展示契约**：列表头部汇总条（项数 · 总大小 · 最早到期）、每项本地化类型标签（颜色区分：缓存绿/日志琥珀/数据蓝/其余灰）、孤儿数据条目以「未认领数据」标识展示（内部 `orphan` 标识不暴露）、恢复/清除图标按钮；读取时按扫描器一致规则自愈旧条目的错误类型（cache→logs）
- **类型与数据根本地化映射**：前端新增 kind/root 内部标识 → 本地化标签映射表（labels.ts），未收录标识回退原始值；parity 测试守护映射表与三语字典不漂移

## Capabilities

### New Capabilities
- `tool-attribution`: 工具身份归因契约——运行时家族合并、别名与包含工具区分、寄居二进制归属、可清理子项精确类型分类

### Modified Capabilities
- `trash-recycle-bin`: 回收站列表展示契约（汇总条、类型标签、孤儿标识、图标操作）与旧条目类型自愈

## Impact

- **代码**：`internal/scanner/`（classify.go 家族判定与安装根、attribute.go 别名合并与 subKind、scanner.go 归一化别名、types.go `family`/`SubEntry.kind`）、`internal/rules/`（nodejs 规则：AppData/npm 数据目录、`~/.npm` 与 `%LocalAppData%\npm-cache` 清理项、元数据）、`internal/cleaner/`（子项精确类型入库）、`internal/trash/`（List 展示字段与 refineKind 自愈）、`internal/uninstall/`（nodejs 加入卸载黑名单）、`frontend/`（main.ts 标签页/回收站渲染、labels.ts 映射表、index.html/style.css 回收站布局与 SVG 图标）、i18n 三语键（`ui.kindLabel.*`、`ui.root.*`、`ui.aliases`、`ui.bundledTools`、`ui.installer.nodejs`、`tool.desc.nodejs` 等）
- **行为**：Windows 扫描不再出现 node/npm/npx 各自成行；claude/kimi 等工具的 aliases 补齐；回收站列表展示类型标签与汇总；`~/.npm/_logs` 等日志条目在清理/回收站中以 logs 归类
- **不引入**：新的扫描/清理/删除路径（回收站红线不变）；不改变 CLI JSON 既有键结构（`family`、`SubEntry.kind` 均为新增可选字段）
