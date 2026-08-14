# Design: 工具归因细化 + 回收站展示契约

## Context

现状（参见 proposal.md）：这批行为改动已在工作区实现、与孤儿数据/探测同批开发但超出该变更范围，本变更负责归档。红线不变：USER 级残留唯一处置路径是内置回收站，不新增任何删除路径。相关契约见 specs（tool-attribution / trash-recycle-bin）。

## Goals / Non-Goals

**Goals**
- Windows 等布局下 Node.js 运行时家族合并为单一工具，版本探测取运行时版本
- 别名契约修正：curated 别名进 JSON、寄居二进制不冒充别名、family 字段驱动「包含工具」展示
- 可清理子项精确类型（logs 等）贯穿扫描 → 清理 → 回收站
- 回收站列表信息密度与类型标注（汇总条、配色标签、孤儿标识、SVG 图标）

**Non-Goals**
- 新增/修改清理与删除路径（红线不变）
- 改动既有 CLI JSON 键结构（`family`、`SubEntry.kind` 为新增可选字段）
- 家族概念推广到其他运行时（python 等暂不合并）

## Decisions

### D1: 家族判定：命令名表 + 布局感知，brew 例外

`nodejsFamily` 固定为 node/npm/npx/corepack/node-gyp（yarn/pnpm 等独立分发的包管理器不并入）。`classify` 内按匹配顺序判定：brew Cellar 命中优先返回公式名（`Cellar/node` 已合并命令，改名会破坏 `brew uninstall node`）；npmPkgMatch 命中 node_modules 布局（nvm/fnm）时归家族；兜底按命令名归家族（同目录布局：Windows 官方安装器 / nvm-windows / Volta / scoop / unix 散放）。入口名归一化 `normEntryName` 剥 `.exe/.cmd/.bat/.com` 后比较。
备选：仅按"目录含 node.exe"判定 → 共享 bin 目录误并；仅按命令名判定 → node_modules 布局漏并。选定表 + 布局组合。

### D2: 安装根判定：目录内必须存在 node/node.exe

`nodejsInstallRoot` 仅当目录本身含 node 或 node.exe 时返回该目录为安装根，否则返回空（unix 散放的 npm 脚本按文件计大小）。避免 `~/.local/bin` 等共享 bin 目录整体计入 nodejs 安装占用。`probeOrder` 在 finalize 前把 node 排到 Binaries[0]，版本探测（取 Binaries[0]）得到运行时版本而非 corepack 版本。

### D3: 别名契约三处修正

- **curated 别名并入**：`attribute` 中非家族工具经 `ruleTable.Lookup` 把规则别名（claude→claude-code、pip→pip3、hf 等）并入 aliases；家族工具排除——其 curated 别名就是合并命令清单，并入会把未安装的 corepack/node-gyp 也算进「包含工具」
- **寄居二进制不冒充别名**：`reAttributeVendors` 不再把宿主捆绑的 rg/fd 加进 aliases（它们仍在二进制列表逐条可见）
- **前端展示**：`family` 非空 →「包含工具」；普通工具 →「别名」，且 >3 条不展示（pyenv shims 推入几十个命令名，既非别名也非包含工具，展示属噪音）

### D4: 子项精确类型：只认有把握的类别

`subKind(parentKind, name)`：`_logs` 或以 `.log` 结尾 → `logs`，其余继承父项（宁缺毋滥，不靠猜测扩大分类）。`SubEntry.Kind` 进 JSON；cleaner 移入回收站时用子项精确类型、旧缓存无字段回退父项；`trash.List` 读取时 `refineKind` 自愈（仅 `cache`→`logs` 这一可明确识别的组合，升级前写入的旧条目）。
备选：按目录名猜类型（dev/old/dist…）→ 误判风险高，否决。

### D5: 回收站展示：汇总条 + 类型标签 + 孤儿标识

列表头部 `trashSummary`（项数 · 总大小 · 最早到期）；每行 ktag 类型标签配色（KIND_TONE：cache=绿、logs=琥珀、data/download=蓝、其余灰；孤儿=蓝）；孤儿条目以本地化「未认领数据」展示（后端内部 tool id "orphan" 不暴露）；恢复/清除为线性 SVG 图标按钮（与卸载复制按钮风格统一，替代 emoji，跨平台渲染一致）；空态清空汇总。清空回收站按钮移至头部 ghost-danger，避免误触主操作区。

### D6: kind/root 本地化映射表 + parity 测试

`labels.ts` 维护内部标识 → i18n 键映射（KIND_LABEL_KEY / ROOT_LABEL_KEY），未收录标识回退原始值（新类型不显示空白）；`labels.test.ts` 校验映射表每个键在三语字典均存在，防两端漂移。后端字典新增 `ui.kindLabel.*`、`ui.root.*`、`ui.aliases`、`ui.bundledTools`、`ui.installer.nodejs`、`tool.desc.nodejs` 等键。

### D7: nodejs 规则配套

curated 规则 `nodejs`（installer=nodejs）：DataDirs 含 `%AppData%\npm`（Windows npm 全局前缀，卸载残留检测用）；Cleanables 含 `~/.npm` 与 `%LocalAppData%\npm-cache`（cache 级，可安全清理）。`uninstall` blocklist 加 nodejs（家族合并工具不可直接卸载）；`ProbeSafeInstaller` 加 `InstNodejs`（家族工具可安全执行版本探测）。

## Risks / Trade-offs

- [Windows 扫描结果结构性变化（node/npm/npx 独立成行 → nodejs）] → 家族工具保留完整二进制列表与数据目录，卸载走 blocklist 保护；brew/nvm 用户不受影响（不参与合并）
- [curated 别名并入使 aliases 变长、展示变化] → 前端 >3 条限制 + 单测覆盖 claude/pip/hf
- [子项类型纠正改变回收站标签（cache→logs）] → 仅明确组合（`_logs`/`*.log`），且是更准确的展示，无行为破坏
- [旧扫描缓存缺 `SubEntry.kind`] → cleaner 回退父项类型，不报错

## Migration Plan

无 schema 迁移：`family`、`SubEntry.kind` 均为新增可选 JSON 字段，CLI 契约向后兼容；旧缓存子项缺 kind 时回退父项；旧回收站条目读取时自动自愈类型。可回滚：行为改动集中于归因与展示，无持久化格式变更。
