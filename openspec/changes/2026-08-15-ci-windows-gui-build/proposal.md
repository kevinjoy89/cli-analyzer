# CI 补 Windows GUI 构建验证

## Why

Windows 是一等平台（本应用是 Wails Windows 应用，刚完成 windows-test-portability 让 `go test` 在 Windows 全绿），但 CI 的 GUI 构建（`wails build`）只在 `macos-latest` 上验证（`.github/workflows/ci.yml` 的 `wails-build` job）。Windows 的 NSIS 打包、WebView2、图标/清单/资源文件问题只能到发布时才发现。同时，windows-test-portability 变更（5.3 节）自述 trash/cleaner/uninstall 的 **fsync 用例在沙箱失败、真实环境未验证**——CI 测试矩阵虽已加 `windows-latest`，但这些用例在真实 Windows runner 上是否通过尚未确认，存在"埋雷未踩"风险。

## What Changes

- **CI 新增 `wails-build-windows` job**：`windows-latest` 上跑前端构建（embedded assets）+ `wails build`，验证 Windows GUI 完整构建管线；与现有 macOS `wails-build` 并行
- **观察并确认 fsync 用例**：`Test (windows-latest)` job 中 trash/cleaner/uninstall 的 fsync 用例（`FlushFileBuffers` 相关）在真实 runner 的表现；若失败，给受影响测试加 **fsync 可用性探针**（临时文件 `Sync()` 失败即 `t.Skip`，不弱化断言）
- **（可选）NSIS 安装器构建验证**：choco 装 makensis 后 `wails build -nsis`，验证 Windows 安装器产物可构建（若不稳定则标记可选、不阻塞主变更）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯 CI/仓库基础设施 -->

## Impact

- **代码**：`.github/workflows/ci.yml`（新增 job）；条件修改 `internal/trash/trash_test.go`、`internal/cleaner/cleaner_test.go`、`internal/uninstall/*_test.go`（仅当 fsync 用例在真实 Windows 失败时）
- **行为**：Windows GUI 构建回归被 CI 拦截（NSIS/WebView2/资源问题不再留到发布）；fsync 用例在真实 Windows 上的表现被确认（绿或显式跳过）
- **不引入**：不改产品代码语义；不动 release.yml；不新增依赖（choco nsis 仅在可选 job 中）
