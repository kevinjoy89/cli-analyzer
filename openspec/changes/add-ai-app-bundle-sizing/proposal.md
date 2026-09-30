## Why

AI 编目条目的桌面端形态（Electron 应用包）装在 /Applications，不在扫描测量范围——WorkBuddy 明明装着且应用包数百 MB，「安装占用」却显示 0 B（现为 —）。用户视角：装着的工具安装占用不可能是 0。编目条目已声明 Apps（桌面端宿主应用名），缺的只是把应用包纳入测量。

## What Changes

- AI 编目条目（category 非空）声明的桌面端应用，其 .app 包（/Applications、~/Applications、/System/Applications 中实际存在者）SHALL 作为 Kind install 的归因目录计入：安装占用列 = CLI 安装根 + 桌面端应用包，总占用相应包含应用包。
- 应用包缺失（已卸载）时不归因——与「已卸载」标注口径一致。
- 仅 macOS 实现；Windows/Linux 的桌面端安装包测量留待后续。
- 后端注入点：应用包路径解析做成 scanner 包级变量，便于测试替换。

## Capabilities

### New Capabilities
<!-- 无 -->

### Modified Capabilities
- `ai-tool-catalog`: 新增「桌面端应用包计入安装占用」要求。

## Impact

- `internal/platform/appbundle_darwin.go`（AppBundlePaths）、`internal/scanner/attribute.go`（metadata 阶段追加 install 归因）。
- 实机预期：workbuddy 安装占用 = WorkBuddy.app 实际大小；zcode/code/kimi/opencode/dsh/gemini/u1s1/agy 同理；trae-cn/marscode 维持 —（应用包不存在）。
- 前端无改动（"—" 逻辑已按 install 根存在性判定）。
