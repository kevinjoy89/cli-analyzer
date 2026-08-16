# 仓库卫生：README 漂移、死参数、变更归档

> 变更目录：2026-08-15-repo-hygiene；无能力变更

> **执行环境（必读）**：本仓库在 DSH 沙箱下工作，所有 `pwsh` 命令（git mv / go test / gofmt 等）当前被沙箱拦截（`SetNamedSecurityInfoW failed: grantWrite`）。实施时每一步验证/提交命令经 `pwsh` 执行会弹出授权请求——批准 `workspace-write` 即可放行；授权为会话级。第 3 步归档用 `git mv` 或 `openspec archive`（需 pwsh）。

## 1. README 修复

- [ ] 1.1 `README.md`：截图表格（app-light-en.png/app-dark-en.png）替换为占位注释；Usage 补 `uninstall <tool>` 两条；版本示例 0.3.3 → 0.3.8
- [ ] 1.2 `README.zh-CN.md`：同步（截图 app-light.png/app-dark.png 占位、uninstall 两条、版本示例）
- [ ] 1.3 验证：`grep -rn "docs/screenshots" README.md README.zh-CN.md` 无输出；`grep -n "uninstall <tool>" README.md` 有输出
- [ ] 1.4 提交：`docs: fix broken screenshots, add uninstall to CLI list, refresh version example`

## 2. 移除 -full 死参数

- [ ] 2.1 `internal/cli/scan.go`：删 `full := fs.Bool(...)`（:21）；`Options{Full: *full, ...}` 改 `Options{NoCache: *noCache, ToolFilter: filters}`（:42）
- [ ] 2.2 `internal/scanner/types.go`：Options 删 `Full bool` 字段与注释
- [ ] 2.3 `internal/scanner/scanner.go`：`:109` `Options{Full: opts.Full}` 改 `Options{}`
- [ ] 2.4 验证：`gofmt -l .` 空 + `go vet ./...` + `go test ./internal/scanner/ ./internal/cli/` 全绿 + `grep -rn "\.Full\|Full:" --include="*.go" internal/` 无输出
- [ ] 2.5 提交：`refactor: remove dead -full flag (unattributed dirs are always measured)`

## 3. OpenSpec 变更归档

- [ ] 3.1 归档 5 个已完成变更（优先 `openspec archive <slug>`，否则 `git mv` 到 `openspec/changes/archive/`）：2026-08-14-fix-orphan-gui-data / windows-test-portability / git-family-merge / npm-global-shim-attribution / cleanup-ui-tweaks
- [ ] 3.2 验证：`ls openspec/changes/` 为空（无活跃变更；本批其余 4 个 change 在执行时处于进行中，不归档）
- [ ] 3.3 提交：`chore(openspec): archive 5 completed changes`
