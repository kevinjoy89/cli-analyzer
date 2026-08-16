## Why

用户反馈：明明用 npm 全局安装了 pi agent 和 opencode，软件里却只看到空壳的 `pi.cmd`/`opencode.cmd` 行。根因三层：① classify 不识别 Windows npm 全局 shim（`%APPDATA%\npm\<name>.cmd` 是 node_modules 中包的 bin 入口，却被当作 InstOther 独立工具）；② `reAttributeVendors` 把位于 nodejs 数据目录 `%APPDATA%\npm` 内的 shim 二进制挪进 nodejs，留下空壳行；③ nodejs 规则把整个 `%APPDATA%\npm`（含所有 -g 包，实机 643MB）占为己有，包的真实归属（opencode-ai 511MB、pi-coding-agent 102MB、~/.pi 397MB）完全不可见。

## What Changes

- **classify 识别 npm 全局 shim**：新增 `npmGlobalShim(dir, shimName)`——`<prefix>/<name>.cmd` 优先按普通包 `node_modules/<name>` 解析，其次扫描全部包（普通包 + 作用域包）按 package.json 的 `bin` 字段匹配 shim 名（覆盖 bin 名 ≠ 包名的场景：opencode-ai 的 bin 是 opencode、@earendil-works/pi-coding-agent 的 bin 是 pi）；返回 `Installer: InstNpm` + `InstallRoot = 包目录`，ToolID = shim 名（与用户认知和 curated 规则一致）。仅 .cmd/.bat 形态触发（unix 上 npm 全局是符号链接，EvalSymlinks 后由 npmPkgMatch 处理）。
- **reAttributeVendors 守卫**：InstNpm 且带 installRoot 的工具（npm 全局包）是真实工具，其 shim 二进制即使位于其他工具的数据目录（nodejs 的 %APPDATA%\npm）也保留在自身工具下，不再被吞成空壳行。
- **nodejs 规则移除 `%APPDATA%\npm` 数据目录**：npm 全局包已归属各自工具（安装根 = npm\node_modules\<pkg>），nodejs 不再双重计数；残留检测语义由各包自身工具承担。

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
- `tool-attribution`: npm 全局 shim → npm 包工具（Windows）；npm 全局包不被 reAttribute 吞并

## Impact

- **代码**：`internal/scanner/classify.go`（npmGlobalShim/pkgHasBin）、`internal/scanner/attribute.go`（reAttributeVendors 守卫）、`internal/rules/tools.go`（nodejs 规则去 %APPDATA%\npm）
- **测试**：TestClassifyNpmGlobalShim（作用域/普通包/bin 名≠包名/无 node_modules 保持原行为）、TestReAttributeKeepsNpmGlobal
- **行为**：实机验证——`opencode` v1.18.18（安装根 511MB）、`pi` v0.84.2（安装根 102MB + ~/.pi 397MB）成为完整工具行；nodejs 不再含 opencode.cmd/pi.cmd；`pi.cmd`/`opencode.cmd` 空壳行消失
- **不引入**：unix 行为不变（符号链接路径仍走 npmPkgMatch，包名即工具名）；无 node_modules 的目录保持 InstOther

## 变更记录

- 本变更与 2026-08-14-git-family-merge 同属「工具身份归并」主题，独立记录以便回溯。
