## Why

工具目前没有版本更新机制，用户只能手动上 GitHub Releases 页面查看、下载并安装新版本。对于一款以"安全、省心"为核心卖点的工具，这割裂了体验：用户被提醒"磁盘又被占满了"的同时，却要自己去找新版安装包。我们需要一个内置的更新流程：自动检查、提示下载、下载完交给用户手动安装——把"发现新版本"这件事从用户手里接过来，同时把"安装"这个高风险动作留在系统原生流程里。

## What Changes

- 新增 `internal/updater/` 包：semver 版本比较、按平台选择 release asset、带进度的 HTTP 下载、SHA256 校验。
- 新增 `internal/buildinfo/` 包：构建期通过 `-ldflags -X` 注入 `Version` 与 `InstallSource`，替代目前散落在 `internal/cli/run.go`、`gui/service.go`、`wails.json` 三处的手写版本号（单一来源）。
- GUI：启动时后台静默检查更新（默认开启，可在首选项中关闭）；发现新版弹窗提示（含当前/最新版本号），提供 [下载] [忽略该版本]；下载过程显示进度条（可取消）；下载完成校验通过后提示 [立即安装]——点击后先打开安装包再退出应用，安装由用户手动完成。Help 菜单新增「检查更新…」手动触发入口。
- CLI：新增 `cli-analyzer update check --json` 子命令，供终端/脚本使用。
- 发布流程：`release.yml` 中 Linux/Windows 各构建两次（`-ldflags` 注入 InstallSource 区分 deb/tarball、installer/portable），并新增一步生成 `checksums.txt` 校验和文件。
- 版本号从 tag 派生注入，`cli-analyzer version` 顺带打印安装来源（如 `cli-analyzer 0.2.4 (linux, deb)`）。

## Capabilities

### New Capabilities
- `app-update`: 版本检查、下载与安装引导。覆盖：检查时机与开关、限流缓存、版本比较规则（跳过 draft/pre-release）、按平台/安装来源选择产物、下载进度与取消、SHA256 校验、安装引导（打开安装包并退出）、安装来源标识与兜底。

### Modified Capabilities
<!-- 无：既有 capability（trash-recycle-bin、usage-trends）的需求不变。 -->

## Impact

- 新增包：`internal/updater/`（纯逻辑，可单测）、`internal/buildinfo/`。
- 修改包：`internal/config/`（新增 `checkUpdates` 开关与 `lastCheckAt` 缓存）、`internal/cli/`（新子命令 + 版本单一来源）、`gui/service.go`（新增绑定 + 版本来源）、`main.go`（Help 菜单）。
- 前端：`frontend/src/main.ts` 更新弹窗、进度条、首选项开关、忽略版本记忆。
- 发布：`.github/workflows/release.yml`（每平台双构建 + checksums 步骤）；`wails.json` 的 `productVersion` 不再作为版本唯一来源（保留用于安装包元数据）。
- 无新增第三方依赖（GitHub Releases API 走 net/http，无 SDK 需要）。
- 国内网络：GitHub 不可达时静默失败，不影响正常使用。
