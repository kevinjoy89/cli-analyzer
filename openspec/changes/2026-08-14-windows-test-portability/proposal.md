## Why

多开发者维护下，CI 只跑 ubuntu/macos，Windows 开发者本地 `go test` 的失败被静默放过。实测 Windows 上 `go test ./internal/...` 有 7 个包共 23 个失败，分四类：真实生产 bug（路径按 `/` 硬切分，Windows 反斜杠路径不识别）、unix 专属语义测试（XDG 根、`~/.local/bin` PATH 增强、dpkg 探测）、测试夹具未平台化（`/a/b` 字面量、依赖 HOME 环境变量而 Windows 用 USERPROFILE）、以及本会话沙箱伪影（fsync 被沙箱拒绝，真实开发机无此问题）。

## What Changes

- **生产修复（真实 bug）**：`classify.go` 的 `versionedMatch`/`npmPkgMatch`/`pipxMatch` 与 `attribute.go` 的 `toolchainVersion` 改为分隔符无关（同时识别 `/` 与 `\`）——Windows 上 versioned/npm/pipx 安装与 rustup toolchain 推断此前完全不生效
- **测试平台化**：路径字面量改 `filepath.Join`（TestUnder/TestSumMaximal/TestFootprintOf）；HOME 依赖用例补 `USERPROFILE`（pyenv/rustup derive、uninstall fakeRoots、cli setupUninstallEnv）；uninstall 残留用例按平台选择 config 根（unix XDGConfig / Windows AppData）；TestCacheRoundTrip 与 cli setupUninstallEnv 隔离 LOCALAPPDATA（避免 Windows 上读写真实缓存）
- **unix 专属语义显式跳过**：TestRootXDGEnv、TestPathDirsAugmentUserDirs、TestResolveCleanableGlob、TestProbeDPKGManaged、TestResolveCommandViaAugmentedPath——带原因注释的 `t.Skip`；TestPathDirsDedup 改为平台分支断言（Windows 用 `;` 分隔 + %WINDIR% 系统目录）
- **结构性保障**：CI 测试矩阵加入 `windows-latest`（本应用是 Wails Windows 应用，Windows 是一等平台）；`.gitattributes` 增加 `*.go text eol=lf`（gofmt 需要 LF，避免 CRLF 让 `gofmt -l` 全库误报，保证各平台检出一致）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯测试/CI/仓库基础设施 -->

## Impact

- **代码**：`internal/scanner/classify.go`、`internal/scanner/attribute.go`（分隔符无关）；`internal/scanner/classify_test.go`、`core_test.go`、`internal/platform/platform_test.go`、`internal/rules/rules_test.go`、`internal/updater/updater_test.go`、`internal/uninstall/uninstall_test.go`、`internal/cli/uninstall_test.go`（平台化/跳过）
- **基础设施**：`.github/workflows/ci.yml`（+windows-latest）、`.gitattributes`（*.go eol=lf）
- **行为**：Windows 上 `go test ./internal/...` 除沙箱 fsync 伪影外全绿；Windows 上 versioned/npm/pipx/rustup 归因修复（生产行为改善）；CI 三平台矩阵保证未来回归被立即发现
- **不引入**：不改变孤儿/排除/清理等产品语义；不引入新的依赖；unix 专属测试在 Windows 显式跳过而非弱化断言
