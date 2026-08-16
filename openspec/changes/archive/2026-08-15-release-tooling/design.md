# Design: 发布脚本固化

## Context

发布流程的检查、版本号、release notes 三步全部依赖人肉操作与临时文件（`.go-toolchain/` 下的 fix-release.js 等）。历史事故：v0.3.7 本地漏跑 gofmt → CI 红；v0.3.8 用 PS 5.1 提交 notes → 中文全部变 `?`。目标是把这三步固化为仓库内脚本，让"发布"可重复、可验证。

## Goals / Non-Goals

**Goals**
- 发布前检查脚本化：一个命令跑完 gofmt/vet/go test/前端测试/tsc/三平台编译/版本一致性
- 版本号更新脚本化：wails.json productVersion 单一来源，写后回读校验
- release notes 提交脚本化：本地自检（CJK/结构）+ 提交后回读校验，杜绝乱码回归
- 清理 `.go-toolchain/` 发布临时文件，`release-process.md` 指向新脚本

**Non-Goals**
- 不自动化 GUI 冒烟（真机人工，保持 MUST）
- 不改 CI / release.yml（脚本服务本地发布操作）
- 不新增第三方依赖（Node 内置 fetch；bash 标准工具）

## Decisions

### D1: check.sh 复用 test-all.sh

`scripts/release/check.sh` 先调用 `./scripts/test-all.sh`（已覆盖 gofmt/vet/go test/前端测试），再补 tsc + 三平台交叉编译 + 可选版本一致性校验（传 tag 参数时 grep wails.json）。避免两份检查逻辑漂移。`set -euo pipefail` 任一失败即退出。

### D2: bump-version.sh 跨平台 sed

macOS 的 BSD sed 与 Linux 的 GNU sed 的 `-i` 语法不同：统一用 `-i.bak`（两者都支持）写备份，成功后删除 `.bak`。版本号用正则 `^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$` 校验（支持四位小补丁 0.3.7.2），非法输入拒绝执行。

### D3: notes.js 的定位与自检

从 `.go-toolchain/fix-release.js` 迁移，修复其硬编码问题（release id 371219438 → 按 tag 查 `GET /releases/tags/<tag>`）：

- notes 源文件约定：`docs/release-notes/<tag>.md`（先写 notes 再提交）
- **提交前自检（拒绝式）**：不含任何 CJK 字符 → 拒绝（双语模板强制项）；缺中英分隔 `---` 或缺下载产物清单（`CLI-Analyzer-`/`checksums.txt`）→ 拒绝
- **提交后回读校验**：PATCH 后从响应 body 检查含 CJK，失败则报错提示（不静默成功）
- `--verify` 模式：只做本地自检，不访问网络（CI/本地都可跑）
- 认证：`GH_PAT` 环境变量（沿用原脚本）

### D4: notes-template.md 双语骨架

`scripts/release/notes-template.md` 提供中英两段结构（标题/变更分节/下载产物逐项/未签名提示），发布者复制到 `docs/release-notes/<tag>.md` 后填充；`---` 分隔中英，满足 notes.js 的结构自检。

### D5: release-process.md 更新

- 第 1 步"本地全量验证" → `./scripts/release/check.sh`
- 第 2 步"版本号" → `./scripts/release/bump-version.sh <v>`（保留 sed 手工路径为 fallback）
- 第 5 步"双语 release notes" → 写 `docs/release-notes/<tag>.md` → `GH_PAT=<token> node scripts/release/notes.js <tag>`；保留血泪教训说明（脚本已内置）
- 第 7 步 GUI 冒烟清单原样保留

## Risks / Trade-offs

- [bash 脚本在 Windows 上不可直接运行] → 文档注明 Git Bash；PowerShell 用户保留手册路径（release-process.md 已有等价命令）
- [notes.js 依赖 GitHub API 稳定性] → 失败时明确报错退出（不静默）；重试即可（与现状一致）
- [check.sh 三平台编译耗时] → 仅发布前运行（本地日常用 test-all.sh）
- [删除 .go-toolchain 临时文件后需要旧脚本] → 逻辑已迁移至 notes.js；git 历史可找回

## Migration Plan

无数据/持久化变化。新脚本先以 `--verify`/dry-run 方式本地验证，再更新 release-process.md；`.go-toolchain/` 临时文件在确认迁移完成后删除。
