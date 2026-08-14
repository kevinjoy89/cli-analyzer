## 1. Node.js 运行时家族合并

- [x] 1.1 scanner 家族判定：`nodejsFamily` 命令表 + `normEntryName` 归一化（剥 .exe/.cmd/.bat/.com），classify 三处接入（空 real 兜底、npmPkgMatch 的 node_modules 布局、InstOther 兜底），brew Cellar 保持公式名
- [x] 1.2 `nodejsInstallRoot`：目录内含 node/node.exe 才返回安装根，否则空（共享 bin 目录不整体计入）；新 installer 类型 `InstNodejs` 并加入 `ProbeSafeInstaller`
- [x] 1.3 `probeOrder`：nodejs 合并工具把 node 排到 Binaries[0]，版本探测取运行时版本；Tool JSON 新增 `family` 可选字段（omitempty）
- [x] 1.4 规则配套：curated 规则 nodejs（installer=nodejs、DataDirs `%AppData%\npm`、Cleanables `~/.npm` + `%LocalAppData%\npm-cache`）、metadata（homepage/desc）、uninstall blocklist 加 nodejs、i18n `tool.desc.nodejs`/`ui.installer.nodejs`
- [x] 1.5 单测：同目录布局合并（Windows 官方安装器形态）、node_modules 布局合并、brew 不合并、共享 bin 不计安装根、别名归一化（corepack.cmd → corepack）、探测取 node 版本

## 2. 别名契约修正

- [x] 2.1 `attribute` 并入 curated 别名（claude→claude-code、pip→pip3、hf 等），家族工具排除；`reAttributeVendors` 不再把寄居二进制加为别名
- [x] 2.2 前端展示：family 非空 →「包含工具」，普通工具 →「别名」，>3 条且非 family 不展示；i18n `ui.aliases`/`ui.bundledTools`
- [x] 2.3 单测：curated 别名进 JSON、寄居二进制不冒充别名、家族别名归一化、前端别名展示分支

## 3. 可清理子项精确类型

- [x] 3.1 `subKind`：`_logs`/`*.log` 结尾 → logs，其余继承父项；`SubEntry.Kind` 进 JSON；扫描子项生成处填 kind
- [x] 3.2 cleaner 入库用子项精确类型，旧缓存无字段回退父项；trash `refineKind` 自愈（仅 cache→logs）
- [x] 3.3 前端 `kindLabel` 本地化标签（详情行/子行/确认弹窗/回收站行统一），子项与父项类型不同时标注精确类型
- [x] 3.4 单测：日志子项归类、旧缓存回退、refineKind 自愈、cleaner 入库类型

## 4. 回收站展示契约

- [x] 4.1 列表头部汇总条（项数 · 总大小 · 最早到期），空态清空汇总；清空回收站按钮移至头部 ghost-danger
- [x] 4.2 行式布局：本地化类型标签 + KIND_TONE 配色（cache=绿/logs=琥珀/data+download=蓝/其余灰）、路径、大小、来源·时间、恢复/清除 SVG 图标按钮；孤儿条目显示「未认领数据」不暴露内部标识
- [x] 4.3 SVG 图标统一（回收站按钮/铃铛/恢复/清除替换 emoji）；i18n `ui.trashSummary` 等新键
- [x] 4.4 单测/手动验证：三语渲染、孤儿标识、空态、类型配色、自愈生效

## 5. kind/root 本地化映射与回归

- [x] 5.1 `labels.ts`：KIND_LABEL_KEY / ROOT_LABEL_KEY 映射表 + 回退原始值；`labels.test.ts` parity 校验三语字典不漂移
- [x] 5.2 全量回归：Go 单测 + 前端 TS/test + i18n parity + 三平台交叉编译；Windows 扫描验证 nodejs 合并与回收站展示
