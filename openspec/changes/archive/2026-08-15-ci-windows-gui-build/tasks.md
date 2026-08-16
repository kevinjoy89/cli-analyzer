# CI 补 Windows GUI 构建验证

> 变更目录：2026-08-15-ci-windows-gui-build；无能力变更（CI 基础设施）

## 1. 新增 wails-build-windows job

- [x] 1.1 `.github/workflows/ci.yml` 追加 `wails-build-windows` job（镜像现有 macOS wails-build）：windows-latest / checkout / Go（go-version-file + cache）/ Node 22（npm ci + npm run build）/ `go install github.com/wailsapp/wails/v2/cmd/wails@v2.13.0` / `wails build`
- [x] 1.2 提交：`ci: add Windows Wails GUI build job`

## 2. 观察 CI 并处置 fsync 用例

- [x] 2.1 推送分支触发 CI，观察：`Test (windows-latest)` 的 trash/cleaner/uninstall fsync 用例；`Wails GUI build (Windows)` job
  > 结果（用户确认，2026-08-15）：**CI 全绿**——`Test (windows-latest)` 通过（fsync 用例在真实 Windows runner 上正常，无需探针）；`Wails GUI build (Windows)` 通过（新增 job 首次验证成功）。
- [x] 2.2 若 fsync 用例在真实 runner 失败：给 `internal/trash/trash_test.go`、`internal/cleaner/cleaner_test.go`、`internal/uninstall/*_test.go` 受影响用例加 `probeFSync` 探针 + `t.Skip`（跳过不弱化断言）
  > 未触发：真实 runner 上 fsync 用例通过，无需探针。
- [x] 2.3 若 wails-build-windows 失败：按 D1 风险预案处置（如 `-skipbindings`），记录原因
  > 未触发：Windows GUI 构建成功。
- [x] 2.4 提交（如有修复）：`test: skip fsync cases when environment lacks fsync support`
  > 未触发：无修复提交。

## 3. （可选）NSIS 安装器构建验证

- [ ] 3.1 在 wails-build-windows job 追加：`choco install nsis -y --no-progress` + `wails build -nsis`（处理 PATH 刷新问题：绝对路径 makensis 或重启 shell）
  > **未执行（可选，明确跳过）**：NSIS 安装器产物由发布流程的 `release.yml` 构建，CI 已有 `wails build` 验证核心 GUI 管线；NSIS 步骤的 choco/PATH 脆弱性（设计 D3）使其增量价值低于 CI 稳定性成本。后续如需可在独立 job 中补充。
- [ ] 3.2 若稳定 → 保留并提交 `ci: verify NSIS installer build on Windows`；若脆弱 → 标记可选（独立 job 或 if: always()），记录原因不阻塞
  > 未执行：与 3.1 一致跳过，理由同上。
