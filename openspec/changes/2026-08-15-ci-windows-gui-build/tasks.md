# CI 补 Windows GUI 构建验证

> 变更目录：2026-08-15-ci-windows-gui-build；无能力变更（CI 基础设施）

> **执行环境（必读）**：本仓库在 DSH 沙箱下工作，所有 `pwsh` 命令（git push / gh 等）当前被沙箱拦截（`SetNamedSecurityInfoW failed: grantWrite`）。实施时提交/推送命令经 `pwsh` 执行会弹出授权请求——批准 `workspace-write` 即可放行；授权为会话级。本变更的验证主体在 **GitHub Actions**（推送后观察 CI run），本地仅改 YAML + 提交。

## 1. 新增 wails-build-windows job

- [ ] 1.1 `.github/workflows/ci.yml` 追加 `wails-build-windows` job（镜像现有 macOS wails-build）：windows-latest / checkout / Go（go-version-file + cache）/ Node 22（npm ci + npm run build）/ `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` / `wails build`
- [ ] 1.2 提交：`ci: add Windows Wails GUI build job`

## 2. 观察 CI 并处置 fsync 用例

- [ ] 2.1 推送分支触发 CI，观察：`Test (windows-latest)` 的 trash/cleaner/uninstall fsync 用例；`Wails GUI build (Windows)` job
- [ ] 2.2 若 fsync 用例在真实 runner 失败：给 `internal/trash/trash_test.go`、`internal/cleaner/cleaner_test.go`、`internal/uninstall/*_test.go` 受影响用例加 `probeFSync` 探针 + `t.Skip`（跳过不弱化断言）
- [ ] 2.3 若 wails-build-windows 失败：按 D1 风险预案处置（如 `-skipbindings`），记录原因
- [ ] 2.4 提交（如有修复）：`test: skip fsync cases when environment lacks fsync support`

## 3. （可选）NSIS 安装器构建验证

- [ ] 3.1 在 wails-build-windows job 追加：`choco install nsis -y --no-progress` + `wails build -nsis`（处理 PATH 刷新问题：绝对路径 makensis 或重启 shell）
- [ ] 3.2 若稳定 → 保留并提交 `ci: verify NSIS installer build on Windows`；若脆弱 → 标记可选（独立 job 或 if: always()），记录原因不阻塞
