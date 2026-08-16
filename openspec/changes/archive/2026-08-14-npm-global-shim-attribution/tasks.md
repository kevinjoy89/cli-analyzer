## 1. classify 识别 npm 全局 shim

- [x] 1.1 `npmGlobalShim(dir, shimName)`：普通包 node_modules/<name> 直查；否则扫描普通包与作用域包，按 package.json bin 字段匹配 shim 名（覆盖 bin 名≠包名：opencode-ai→opencode、@earendil-works/pi-coding-agent→pi）
- [x] 1.2 `pkgHasBin`：bin 为对象查键；bin 为字符串按 npm 约定（bin 名=包名最后一段）
- [x] 1.3 classify 接入：仅 .cmd/.bat shim 触发，ToolID=shim 名、Installer=InstNpm、InstallRoot=包目录
- [x] 1.4 单测：TestClassifyNpmGlobalShim（作用域/普通/bin≠包名/无 node_modules 保持 InstOther/.exe 不触发）

## 2. reAttribute 守卫

- [x] 2.1 InstNpm + installRoot 的工具（npm 全局包）的二进制不被挪进宿主工具
- [x] 2.2 单测：TestReAttributeKeepsNpmGlobal（pi shim 不被 nodejs 吞并）

## 3. nodejs 规则去重

- [x] 3.1 nodejs 规则移除 `dd(AppData, "npm")`（npm 全局包归属各自工具，避免双重计数）

## 4. 回归与交付

- [x] 4.1 gofmt + go vet + go test ./internal/scanner/ ./internal/rules/ ./internal/platform/（全绿）
- [x] 4.2 端到端：`opencode` v1.18.18（安装根 511MB）、`pi` v0.84.2（安装根 102MB + ~/.pi 397MB）完整成行；nodejs 无 shim 残留；空壳行消失
- [x] 4.3 重建 NSIS 安装器 + 便携 zip，更新 SHA256
