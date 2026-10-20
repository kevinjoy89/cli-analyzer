## 1. 应用包测量

- [x] 1.1 `platform.AppBundlePaths(apps)`：darwin 返回实际存在的 .app 路径（/Applications、~/Applications、/System/Applications），其他平台返回空
- [x] 1.2 `scanner`：AI 条目（category 非空且有 Apps）在 metadata 阶段为存在的应用包追加 Kind install 归因 + measure；包级变量注入便于测试
- [x] 1.3 scanner 测试：注入临时 .app 路径 → 安装占用包含应用包、footprint 相应增长；应用缺失时不归因

## 2. 验证

- [x] 2.1 go test 全绿（`internal/scanner` `internal/platform` `internal/rules` 均通过）；实机重扫：workbuddy/zcode/code 等安装占用 = 应用包实际大小，trae-cn/marscode 维持 —
  - 备注：1.1–1.3 与 `TestAIAppBundleInstallSizing` 在本轮之前已由先前会话落地，本次仅核实与跑测。
  - **实机重扫已确认**（2026-10-20）：workbuddy `/Applications/WorkBuddy.app` 920MB、zcode `ZCode.app` 736MB、code `Visual Studio Code.app` 866MB、kimi `Kimi Code.app` 296MB、agy `Antigravity.app` 466MB + `Antigravity Tools.app` 52MB、codex 安装根 91MB + `ChatGPT.app` 1680MB、dsh 596MB + `DeepSeek Harness.app` 990MB——应用包均已作为 `kind: install` 归因计入安装占用；trae-cn / marscode 无应用包，维持不归因。
