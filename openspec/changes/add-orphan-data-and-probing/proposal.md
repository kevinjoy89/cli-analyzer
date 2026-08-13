## Why

工具已掌握大量 CLI 工具数据，但存在两个缺口：**死掉/未上 PATH 的 CLI 工具残留**（数据目录无人认领、GUI 完全不展示）和**核心视图信息密度低**（版本列对大多数工具显示 `—`）。同时扫描范围界定模糊：非 CLI（GUI 应用及其数据）会混入结果，且既有厂商过滤是子串匹配、无例外机制，既可能误伤（`code` 命中 `opencode`）也无法保留 GUI 厂商旗下的纯 CLI 产品（`gcloud`、`aws`、`az`）。

## What Changes

- **非 CLI 排除体系**：新增厂商排除表（路径片段精确匹配 + 纯 CLI 产品例外白名单），统一应用于 exe 发现与数据目录归因两环节；替换现有 `isVendorInstallDir` 的子串匹配
- **孤儿数据目录**：把后端已存在的 `findUnattributed`（现仅 CLI `--full` 可用）接入 GUI，工具列表底部新增"未归属数据"小节；数据经排除体系过滤后仅以 USER 级展示，只能移入回收站
- **健康探测**：对扫描出的工具后台探测 `--version`/`-V`/`--help`，填充版本列并提取一句话描述；带超时杀进程、结果缓存、失败降级 `—`；Windows GBK 输出转码
- **范围收紧**：全应用只管理 CLI 工具及其残留；GUI 应用、其命令行伴侣、其数据目录一律排除

## Capabilities

### New Capabilities
- `non-cli-exclusion`: 非 CLI（GUI 应用）识别与排除规则——厂商排除表、片段精确匹配、纯 CLI 例外白名单，贯穿 exe 发现与数据目录归因
- `orphan-data`: 未归属数据目录的发现、过滤、展示与回收站处置（GUI + CLI）
- `tool-probing`: 工具版本/描述探测——命令编排、超时与缓存、失败降级、Windows 编码处理

### Modified Capabilities
<!-- 现有 spec 无扫描/发现能力定义；本变更不修改既有 capability 的行为契约 -->

## Impact

- **代码**：`internal/scanner/`（discover.go 发现过滤重构、scanner.go 孤儿管线、新增探测模块）、`internal/platform/`（排除表与片段匹配、探测辅助）、`frontend/src/main.ts`（孤儿小节、探测状态渲染）、i18n 三语键、CLI `scan` 输出（孤儿与探测结果进 JSON 契约）
- **行为**：Windows/macOS/Linux 扫描结果变化（GUI 数据与伴侣进一步排除）；版本列不再大面积 `—`
- **不引入**：体检报告/建议中心（已否决）、一键清理全部（已否决）、永久删除路径（红线不变，USER 级残留仅回收站）
