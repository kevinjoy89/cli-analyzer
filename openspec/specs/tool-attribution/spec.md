# tool-attribution Specification

## Purpose

工具身份的归因与展示契约：将随运行时一起分发的命令家族合并为单一工具，明确别名与包含工具、寄居二进制的边界，并为可清理子项提供精确类型分类。

## Requirements

### Requirement: Node.js 运行时家族合并

随 Node.js 运行时一起分发的命令（node、npm、npx、corepack、node-gyp）SHALL 归并为单一工具 `nodejs`，而非各自成行。合并 SHALL 适用于两类布局：同一安装目录（Windows 官方安装器 / nvm-windows / Volta / scoop 及 unix 独立放置，目录内存在 node 或 node.exe）与 npm global node_modules 布局（nvm/fnm 等）。brew 公式 node 等具有独立身份的安装 SHALL 保持原名，不参与合并。合并工具的主二进制 SHALL 固定为 node（版本探测取运行时版本而非 corepack 等附属命令）。合并工具的安装来源 SHALL 作为已知 CLI 生态处理，可安全执行探测。

#### Scenario: 同目录布局合并

- **WHEN** PATH 中同一目录同时含 `node.exe`、`npm.cmd`、`npx.cmd`、`corepack.cmd`（Windows 官方安装器 / nvm-windows / Volta / scoop）
- **THEN** 扫描结果出现单一工具 `nodejs`，四个命令均为其二进制，该目录计为安装根

#### Scenario: node_modules 布局合并

- **WHEN** nvm/fnm 布局中 node、npm、npx 为独立的 node_modules 包
- **THEN** 归并为单一工具 `nodejs`

#### Scenario: brew 公式保持独立

- **WHEN** 二进制来自 `Cellar/node`（brew 公式 node 已含 npm/npx/corepack）
- **THEN** 工具保持 `node` 原名，不参与家族合并

#### Scenario: 共享 bin 目录不整体计为安装根

- **WHEN** 某目录含 npm 脚本但不存在 node/node.exe（如 `~/.local/bin`）
- **THEN** 该目录不整体计为 nodejs 安装根，仅按文件计大小

#### Scenario: 探测取运行时版本

- **WHEN** 对合并工具 `nodejs` 执行版本探测
- **THEN** 探测 `node --version` 得到 Node 运行时版本，而非 corepack/npm 的版本

### Requirement: 别名与包含工具契约

工具 JSON 的 aliases 字段 SHALL 同时包含 PATH 入口推断的别名与规则表 curated 别名（claude→claude-code、pip→pip3、hf 等）。家族合并工具的 aliases SHALL 为其包含命令的归一化名（Windows 剥 `.exe/.cmd/.bat/.com` 扩展名）。宿主工具捆绑的寄居二进制（如 kimi 自带的 rg/fd）MUST NOT 进入 aliases，仅出现在该工具的二进制列表。前端 SHALL 按 `family` 字段区分展示：家族合并工具的 aliases 标注为「包含工具」，普通工具标注为「别名」；普通工具别名过多（>3 条）时 SHALL 不展示该行，避免 shims 噪音。

#### Scenario: curated 别名并入 JSON

- **WHEN** 规则表声明 claude→claude-code 且 claude 二进制在 PATH
- **THEN** 该工具 JSON aliases 含 claude-code

#### Scenario: 家族别名归一化

- **WHEN** Windows 上 nodejs 家族含 `corepack.cmd`
- **THEN** aliases 展示 `corepack` 而非 `corepack.cmd`

#### Scenario: 寄居二进制不冒充别名

- **WHEN** 宿主工具 kimi 安装目录内捆绑 rg/fd
- **THEN** rg/fd 不出现在 kimi 的 aliases 中，仅出现在其二进制列表

### Requirement: 可清理子项精确类型

可清理项的直属子项 SHALL 携带自身类型：路径名可明确识别为日志（`_logs` 或 `*.log` 结尾）时 MUST 使用精确类型 `logs`，其余子项继承父项类型。清理移入回收站与回收站展示 SHALL 使用子项的精确类型；子项无类型字段（旧扫描缓存）时 SHALL 回退父项类型。

#### Scenario: 日志子项精确归类

- **WHEN** `~/.npm` 的直属子项为 `~/.npm/_logs`（父项类型 cache）
- **THEN** 该子项类型为 logs，清理入库与回收站展示均以 logs 归类

#### Scenario: 子项类型回退

- **WHEN** 旧扫描缓存中的子项无类型字段
- **THEN** 清理时回退使用父项类型，不报错
