# 发布脚本固化

> 变更目录：2026-08-15-release-tooling；无能力变更（仓库工具/流程）

> **执行环境（必读）**：本仓库在 DSH 沙箱下工作，所有 `pwsh` 命令（bash / node / git 等）当前被沙箱拦截（`SetNamedSecurityInfoW failed: grantWrite`）。实施时每一步验证/提交命令经 `pwsh` 执行会弹出授权请求——批准 `workspace-write` 即可放行；授权为会话级。本变更含**删除文件**操作（第 3.4 步删除 `.go-toolchain/` 临时文件，需 `pwsh Remove-Item`，同样需要授权）。

## 1. check.sh

- [ ] 1.1 创建 `scripts/release/check.sh`：调用 `./scripts/test-all.sh` + `(cd frontend && npx tsc --noEmit)` + 三平台 `GOOS=windows|linux|darwin go build ./...` + （可选 tag 参数）grep 校验 wails.json productVersion
- [ ] 1.2 验证：`chmod +x`；本地 `./scripts/release/check.sh` 全部通过
- [ ] 1.3 提交：`chore(release): add pre-release check script (checklist automation)`

## 2. bump-version.sh

- [ ] 2.1 创建 `scripts/release/bump-version.sh`：版本号正则校验；`sed -i.bak` 改 wails.json productVersion；写后回读 grep 校验；删 .bak
- [ ] 2.2 验证：`./scripts/release/bump-version.sh 0.3.9` 改版 → 再改回 `0.3.8` 还原；非法版本号（如 `abc`）拒绝
- [ ] 2.3 提交：`chore(release): add version bump script (single source in wails.json)`

## 3. notes.js + 模板

- [ ] 3.1 创建 `scripts/release/notes-template.md`：中英双语骨架（标题/变更分节/下载产物逐项/未签名提示/`---` 分隔）
- [ ] 3.2 创建 `scripts/release/notes.js`：按 tag 查 release id（GET /releases/tags/<tag>）；提交前自检（CJK + `---` + 下载产物清单，任一不满足拒绝）；PATCH body（draft=false）；提交后回读校验 body 含 CJK；`--verify` 仅本地自检；GH_PAT 认证
- [ ] 3.3 验证：用模板复制 `docs/release-notes/v0.3.9.md` 跑 `node scripts/release/notes.js v0.3.9 --verify` 通过；删除测试文件
- [ ] 3.4 删除 `.go-toolchain/fix-release.js`、`verify-body.txt`、`current-body.txt`、`release-notes-v0.3.8.md`
- [ ] 3.5 提交：`chore(release): formalize release notes submit/verify script (migrate from .go-toolchain)`

## 4. release-process.md 更新

- [ ] 4.1 更新 `docs/release-process.md`：第 1 步 → check.sh；第 2 步 → bump-version.sh；第 5 步 → notes.js 流程（含 --verify）；GUI 冒烟清单保留
- [ ] 4.2 验证：`grep -n "scripts/release" docs/release-process.md` 三处引用齐全
- [ ] 4.3 提交：`docs: point release runbook at the new scripts`
