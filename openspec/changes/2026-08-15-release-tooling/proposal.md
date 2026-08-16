# 发布脚本固化

## Why

发布流程依赖手工操作与散落临时文件：`.go-toolchain/`（git-ignored 的临时工具链目录）里堆着 `fix-release.js`、`verify-body.txt`、`current-body.txt`、`release-notes-v0.3.8.md` 等发布临时产物；`docs/release-process.md` 的检查清单（gofmt/vet/tsc/三平台编译/版本一致性）全靠人肉逐条执行——v0.3.7 曾因本地漏跑 gofmt 导致 CI 直接红。版本号更新（wails.json productVersion）靠手工 sed，双语 release notes 的 UTF-8/CJK 自检靠记忆（v0.3.8 曾用 PS 5.1 把中文全部变 `?`）。这些应固化为仓库内正式脚本。

## What Changes

- **`scripts/release/check.sh`（新）**：发布前全量检查——调用 `scripts/test-all.sh`（gofmt + vet + go test + 前端测试）+ `npx tsc --noEmit` + 三平台交叉编译 + （传参时）校验 wails.json productVersion 与 tag 一致
- **`scripts/release/bump-version.sh`（新）**：更新 wails.json 的 productVersion（版本号格式校验；`-i.bak` 兼容 BSD/GNU sed；写后回读校验）
- **`scripts/release/notes.js`（新）+ `notes-template.md`（新）**：从 `.go-toolchain/fix-release.js` 迁移——按 tag 定位 release（不再硬编码 release id），提交双语 notes 前强制本地自检（含 CJK + 中英分隔 `---` + 下载产物清单），提交后回读校验 body 含 CJK（v0.3.8 血泪教训）；`--verify` 模式仅本地自检不提交
- **`docs/release-process.md`（修改）**：手工命令替换为脚本调用（check/bump/notes），GUI 冒烟清单保持原样
- **清理**：删除 `.go-toolchain/` 下的发布临时文件（fix-release.js / verify-body.txt / current-body.txt / release-notes-v0.3.8.md）

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯仓库工具/流程 -->

## Impact

- **代码**：`scripts/release/`（新增 4 个文件）；`docs/release-process.md`；删除 `.go-toolchain/` 4 个临时文件
- **行为**：发布操作从"人肉清单 + 临时脚本"变为"仓库内脚本一键执行"；`notes.js` 提交后回读校验防中文乱码回归
- **不引入**：不新增依赖（check.sh 复用既有 test-all.sh；notes.js 用 Node 内置 fetch）；不改变 CI（release.yml 不动）；Windows 开发者用 Git Bash 运行 bash 脚本，PowerShell 用户保留手册路径
