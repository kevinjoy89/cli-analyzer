## 1. 版本单一来源

- [x] 1.1 新建 `internal/buildinfo/buildinfo.go`：`var Version = "dev"`、`var InstallSource = "unknown"`（注释说明由 `-ldflags -X` 注入）
- [x] 1.2 将 `internal/cli/run.go` 的 `Version` 常量替换为引用 buildinfo；`version` 子命令输出追加安装来源（`cli-analyzer 0.2.4 (linux, deb)`）
- [x] 1.3 将 `gui/service.go` 的 `AppVersion` 常量替换为引用 buildinfo（保留 `AppVersion` 导出名或统一改引用，随既有调用方）
- [x] 1.4 运行 `go build ./...` 与既有测试，确认版本引用重构无回归

## 2. updater 核心包

- [x] 2.1 实现语义化版本比较（`internal/updater/version.go`）：解析 `v` 前缀、major.minor.patch、数值比较；含单元测试（相等/更高/更低/非法输入）
- [x] 2.2 实现 GitHub Releases 查询（`internal/updater/release.go`）：`GET /repos/kevinjoy89/cli-analyzer/releases?per_page=10`，过滤 draft/prerelease，取最新正式版 tag 与 assets；超时与错误处理；含 mock 服务器测试
- [x] 2.3 实现产物选择（`internal/updater/asset.go`）：按 `GOOS/GOARCH + InstallSource` 从 assets 匹配资产名（命名约定见 design D4），返回 browser_download_url 与 size；匹配不到返回明确错误
- [x] 2.4 实现下载（`internal/updater/download.go`）：流式写入 `~/Downloads/<名>.part`，进度回调（用 asset size 兜底缺失的 Content-Length），支持 context 取消；完成后 rename 去掉 `.part`；失败/取消删除 `.part`
- [x] 2.5 实现 SHA256 校验（`internal/updater/checksum.go`）：从 release assets 取 `checksums.txt`，解析资产名 → 哈希映射，比对下载文件；缺失/不匹配返回明确错误
- [x] 2.6 实现安装来源兜底探测（`internal/updater/source.go`）：`InstallSource == "unknown"` 时尝试 `dpkg -S`（非 Debian 系或失败则返回 unknown）；含测试（注入假 dpkg 命令）
- [x] 2.7 实现打开文件/URL 的平台分发（`internal/updater/open.go`，build tags）：darwin `open`、windows `cmd /c start ""`、linux `xdg-open`，失败时返回错误由调用方展示路径
- [x] 2.8 为 updater 包补全单元测试并 `go vet ./...` 通过

## 3. 配置扩展

- [x] 3.1 `internal/config/config.go` 新增 `UpdateConfig`：`CheckUpdates *bool`（nil → 默认 true）、`LastCheckAt string`、`IgnoredVersion string`；并入 `Config` 与 `Default()`
- [x] 3.2 `normalize()` 兜底：缺失 `update` 段或 `CheckUpdates == nil` 时置默认 true；确认旧 config.json 加载兼容
- [x] 3.3 新增/更新 config 测试：默认值、旧文件兼容、保存后回读

## 4. GUI 集成

- [x] 4.1 `gui/service.go` 新增绑定：`CheckForUpdates() string`（返回 JSON 结果，手动检查不受缓存限制）、`DownloadUpdate()` / `CancelDownload()`、`InstallUpdate() string`、`GetUpdateConfig()/SetUpdateConfig()`
- [x] 4.2 `Startup` 中触发自动检查（读取 `CheckUpdates`，受 24h 缓存约束）；检查期间通过事件推状态（开始/完成/失败），失败静默
- [x] 4.3 下载进度经 `EventsEmit("update-progress", …)` 推送；完成、失败、取消均推送对应事件
- [x] 4.4 前端 `frontend/src/main.ts`：更新提示弹窗（当前/最新版本、[下载] [忽略该版本]）、进度条（含取消）、下载完成弹窗（[立即安装] [稍后]）、校验失败弹窗（给 Release 页链接）
- [x] 4.5 首选项面板加「自动检查更新」开关；`openPrefs` 读写 `UpdateConfig`
- [x] 4.6 忽略版本持久化：点「忽略该版本」写入 `IgnoredVersion`；检查结果等于被忽略版本时不提示（spec: 忽略版本）
- [x] 4.7 Help 菜单加「检查更新…」（`main.go` buildMenu），点击调手动检查并复用同一提示流程
- [x] 4.8 「立即安装」流程：先 detached 打开安装包（tarball/portable 时提示中展示 `os.Executable()` 路径），再 `runtime.Quit`

## 5. CLI 子命令

- [x] 5.1 `internal/cli/update.go`：`update check [--json]`；文本模式人类可读输出，`--json` 输出 `{current, latest, updateAvailable, assetName, downloadURL}`
- [x] 5.2 退出码约定：0 = 已是最新、2 = 有更新、1 = 错误；接入 `internal/cli/run.go` 分发与 usage 文案
- [x] 5.3 CLI 侧测试：mock release 服务器 + 退出码/输出断言

## 6. 发布流程

- [x] 6.1 `release.yml` Linux job：构建两次（注入 `InstallSource=deb` / `tarball`），每次构建后立即移走产物再构建下一次
- [x] 6.2 `release.yml` Windows job：构建两次（`installer` / `portable`），NSIS 安装器仅 installer flavor 生成
- [x] 6.3 `release.yml` macOS job：单次构建注入 `InstallSource=dmg`
- [x] 6.4 所有 job 构建产物收齐后生成 `dist/checksums.txt`（逐文件 `sha256sum`），随 release 上传
- [x] 6.5 各 job `wails build` 前用 `VERSION`（tag 派生）覆写 `wails.json` 的 `productVersion`
- [x] 6.6 本地跑一次 Linux 双构建，确认两个产物 InstallSource 各为 deb/tarball（`cli-analyzer version` 验证）

## 7. 端到端验证

- [x] 7.1 手动测试链路：mock 或真实 Release → 启动检查 → 提示 → 下载（观察进度/取消）→ 校验 → 立即安装（打开安装包 + 退出）
- [x] 7.2 验证 24h 缓存：连续启动两次，第二次无网络请求
- [x] 7.3 验证离线场景：断网启动 → 静默无提示；手动检查 → 明确错误提示
- [x] 7.4 验证忽略版本与"稍后"：忽略后重启不提示；出现更新版本后恢复提示
- [x] 7.5 验证 `update check --json` 退出码与输出；`cli-analyzer version` 输出安装来源
- [x] 7.6 更新 README（安装/更新说明、macOS 未签名说明）与 usage 文案
