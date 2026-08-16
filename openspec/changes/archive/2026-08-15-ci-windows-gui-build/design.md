# Design: CI 补 Windows GUI 构建验证

## Context

CI 现有两个 job：`test`（三平台 go test + 前端测试 + vet + build）与 `wails-build`（仅 macOS）。Windows GUI 构建从未被 CI 验证。同时 windows-test-portability 的 fsync 用例在本地沙箱失败、真实环境"未验证"——`Test (windows-latest)` job 上线后这些用例的表现是未知数。

## Goals / Non-Goals

**Goals**
- `windows-latest` 上验证 `wails build` 完整管线（前端构建 → embedded assets → GUI 编译）
- 确认（或显式跳过）trash/cleaner/uninstall 的 fsync 用例在真实 Windows runner 上的表现
- 保持与现有 job 的结构一致（actions 版本、Go/Node 缓存设置）

**Non-Goals**
- 不改产品代码语义（fsync 探针只影响测试跳过，不动断言）
- 不动 release.yml 发布管线
- 不引入新的第三方服务

## Decisions

### D1: 独立 job（而非并入 test 矩阵）

新增 `wails-build-windows` job，结构镜像现有 `wails-build`（macOS）：checkout → Go → Node（npm ci + build）→ `go install wails@v2.13.0` → `wails build`。与 macOS job 并行跑。原因：wails 构建需要 frontend/dist（git-ignored），必须在同一 job 内先 `npm run build`；test 矩阵 job 已做同样的事但**不跑 wails build**，且 test 矩阵加 wails 步骤会让"测试失败 vs 构建失败"混在一起。

### D2: fsync 用例的条件跳过（仅当真实失败）

`Test (windows-latest)` 上线后先观察：若 fsync 用例通过 → 无需任何改动；若失败（真实 runner 也可能因环境限制拒绝 FlushFileBuffers）→ 加探针：

```go
// probeFSync 报告当前环境是否支持 fsync（沙箱/某些 CI 容器会拒绝）
func probeFSync(t *testing.T) bool {
    t.Helper()
    f, err := os.CreateTemp(t.TempDir(), "fsync-probe")
    if err != nil {
        return false
    }
    defer f.Close()
    return f.Sync() == nil
}
```

用例开头 `if !probeFSync(t) { t.Skip("fsync not supported in this environment") }`——跳过而非弱化断言，保证真实支持 fsync 的环境仍全量验证。

### D3: NSIS 构建为可选步骤

`wails build -nsis` 需要 makensis（choco install nsis）。choco 安装后 PATH 刷新在同一 step 内可能不生效，需处理（绝对路径或重启 shell）；若不稳定，标记可选、不阻塞主变更——NSIS 的增量价值（验证安装器产物）小于其 CI 脆弱性。

## Risks / Trade-offs

- [wails build 在 Windows runner 上需额外环境] → 与 macOS job 对齐的最小依赖（Go/Node/Wails CLI）；若因 WebView2 加载器下载等网络问题失败，先 `-skipbindings`（绑定已提交）并记录原因
- [fsync 用例真实失败 → 需要探针] → D2 探针方案已备好，跳过不弱化断言
- [NSIS 步骤脆弱] → 可选标记，失败不红整个 job（用 `if: always()` 或独立 job 评估）

## Migration Plan

无数据/持久化变化。推送分支观察 CI 两个结果（windows 测试矩阵 + wails-build-windows），按 D2/D3 决定是否追加修复提交。
