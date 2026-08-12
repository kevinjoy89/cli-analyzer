## Context

现状（详见 proposal.md - Why）：应用无任何更新机制，版本号手写散落三处（`internal/cli/run.go`、`gui/service.go`、`wails.json`），发布流程已具备（tag → 三平台构建 → draft Release），产物命名有约定。本设计覆盖"检查 → 下载 → 校验 → 打开安装包 → 退出"整条链路，安装本身交给系统原生流程。

## Goals / Non-Goals

**Goals:**
- 确定性：更新产物选择不靠猜测——构建期注入安装来源标识。
- 安全对齐工具气质：下载后必校验 SHA256，校验不过不给安装入口。
- 不打扰：网络失败静默、24h 限流缓存、可关闭、可忽略特定版本。
- 可测试：updater 核心为纯逻辑包，不依赖 wails。

**Non-Goals:**
- 自动替换自身二进制（Windows 文件锁、macOS codesign 破坏均在此范围外；"立即安装"后由用户手动完成）。
- tar.gz/portable 用户的二进制自动替换（Linux 上技术上可行，属范围扩张，留待后续）。
- 更新镜像源 / 国内加速（v1 仅静默降级）。
- 下载断点续传（v1 仅支持取消，不支持续传）。

## Decisions

### D1. 版本单一来源：`internal/buildinfo` + `-ldflags -X` 注入

新包 `internal/buildinfo` 声明 `var Version = "dev"`、`var InstallSource = "unknown"`，由 CI 构建命令注入。`internal/cli/run.go` 的 `Version` 常量与 `gui/service.go` 的 `AppVersion` 改为引用 buildinfo。`wails.json` 的 `productVersion` 保留（安装包元数据用），CI 在 `wails build` 前用 tag 覆写，避免漂移。

- 理由：tag 是唯一事实来源（release.yml 已从 `GITHUB_REF_NAME` 派生 VERSION）。
- 替代：保持手写三处——漂移风险正是本次要消除的，否决。

### D2. 安装来源标识：每平台构建两次

release.yml 改为对歧义平台构建两次，分别注入 InstallSource：

```
linux    wails build ... -ldflags "-X cli-analyzer/internal/buildinfo.InstallSource=deb"      → deb
         wails build ... -ldflags "-X cli-analyzer/internal/buildinfo.InstallSource=tarball"  → tar.gz
windows  同上 → installer | portable
macos    单次 → dmg（无歧义，注入仅为 version 输出统一）
```

每次构建产物立即移出 `build/bin` 再构建下一次，避免互相覆盖。

- 理由：deb postinst 写标记文件的方案不对称（tar.gz 无安装钩子），否决；运行时启发式探测（`dpkg -S`）脆弱且依赖发行版，仅作 fallback（见 D6）。
- 同源双构建：同一 commit、同一命令、仅 ldflags 不同，产物除标识外一致，无漂移风险。

### D3. 检查机制：GitHub Releases 列表 + 24h 缓存

- 请求 `GET /repos/kevinjoy89/cli-analyzer/releases?per_page=10`，过滤 `draft` 与 `prerelease`，取第一个 `tag_name` 作最新版。
- 选列表而非 `/releases/latest`：后者不返回 draft 但会返回 prerelease，过滤不完整。
- 未认证限流 60 req/h/IP（NAT 下共享），故配置中缓存 `lastCheckAt`；距上次成功检查 < 24h 时用缓存结果，不发请求。手动检查（菜单/CLI）不受缓存限制。
- 语义化版本比较：仅数字 major.minor.patch，忽略 tag 前缀 `v`；当前版本 `dev`（源码构建）时跳过自动比较，仅手动检查时报"无法确定当前版本"。

### D4. 产物选择：API assets 列表匹配命名约定

用 D3 返回的 release `assets` 列表，按 `GOOS/GOARCH + InstallSource` 匹配资产名（与 release.yml 命名约定一致）：

```
darwin/arm64 → CLI-Analyzer-<v>-macos-arm64.dmg
darwin/amd64 → CLI-Analyzer-<v>-macos-amd64.dmg
windows/installer → CLI-Analyzer-<v>-windows-amd64-installer.exe
windows/portable  → CLI-Analyzer-<v>-windows-amd64-portable.zip
linux/deb         → CLI-Analyzer-<v>-linux-amd64.deb
linux/tarball     → CLI-Analyzer-<v>-linux-amd64.tar.gz
```

取 `browser_download_url` 与 `size`（进度条总量用）。匹配不到 → 按 D6 兜底打开 Release 页面。

### D5. 下载与校验

- 下载目标：`~/Downloads/CLI-Analyzer-<v>-<平台>.<后缀>`；先写 `<目标>.part`，完成后 rename，取消/失败即删 `.part`。
- 进度：流式读取，`Content-Length` 缺失时用 release asset 的 `size`；通过 `EventsEmit("update-progress", …)` 推给前端。
- 取消：context cancel，删除 `.part`。
- 校验：从同一 release 的 assets 中找 `checksums.txt`（新增发布步骤生成），解析出与资产名匹配的 SHA256 比对。checksums 缺失或不匹配 → 不提供「立即安装」，弹窗给 Release 页面链接。安全优先：宁可不给入口，不跳过校验。

### D6. 来源未知的兜底链

`InstallSource == "unknown"` 时依次尝试：① 运行时 `dpkg -S <可执行文件路径>` 判定是否为 deb 管理（仅 Debian 系可用）；② 仍未知 → 打开 Release 页面，由用户自行选择。绝不猜测产物。

### D7. 安装引导：先打开后退出

点击「立即安装」→ 先以 detached 子进程调用系统打开命令（darwin `open`、windows `cmd /c start ""`、linux `xdg-open`，失败则弹窗展示文件路径）→ 再 `runtime.Quit`。顺序不可反：先 quit 再 spawn 在部分平台上子进程会被一并终止。tarball/portable 场景在提示中额外展示 `os.Executable()` 路径（对应 spec "压缩包产物展示二进制路径"）。

### D8. CLI 子命令 `update check`

`cli-analyzer update check [--json]`：非 JSON 时输出人类可读文本；`--json` 输出 `{current, latest, updateAvailable, assetName, downloadURL}`。退出码：0 = 已是最新、2 = 有更新（脚本友好）、1 = 错误。CLI 不做下载（下载/安装引导是 GUI 交互场景）。

### D9. 配置扩展

`config.json` 新增 `update` 段：

```json
{ "update": {
    "checkUpdates": true,        // 默认开
    "lastCheckAt": "...",        // 限流缓存
    "ignoredVersion": "0.3.0"    // 忽略的版本（spec: 忽略该版本）
} }
```

注意 Go bool 零值为 false，`checkUpdates` 用 `*bool` 或 Load 后 normalize（缺失字段 → true），沿用 config 包现有 `normalize()` 模式。旧 config.json 无 `update` 段 → 默认值，向后兼容。

### D10. release.yml 扩展

- Linux/Windows job：按 D2 双构建并注入标识；macOS 单构建注入 `dmg`。
- 所有 job 构建完成后生成 `dist/checksums.txt`（对 dist/ 下每个产物 `sha256sum`），随 release 上传（softprops 已上传 `artifacts/**/*`，checksums 放 dist/ 即可自动带上）。
- `wails build` 前用 `VERSION` 覆写 `wails.json` 的 `productVersion`（sed），保证安装包元数据与 tag 一致。

## Risks / Trade-offs

- GitHub 未认证限流（NAT 共享 IP 更易触发）→ 24h 缓存兜底；手动检查遇 403 时提示"稍后再试"，不崩溃。
- checksums.txt 依赖新发布流程：历史 release 无此文件 → 校验缺失 → 不给安装入口（符合安全优先）。旧版本本无更新功能，首次升级靠手动，可接受。
- macOS dmg 未签名/未公证：应用内下载不经浏览器、无 quarantine 属性，Gatekeeper 行为与浏览器下载不同；产物本就未签名，行为一致，README 注明即可。
- `xdg-open` 在无桌面环境（纯服务器 Linux）失败 → 弹窗/输出展示文件路径，用户可手动处理。
- 大文件下载（dmg ~100MB+）期间取消/失败的状态管理 → `.part` + rename 模式保证目标目录不残留半截文件。
- `wails.json` productVersion 与 tag 漂移 → CI 构建前覆写（D10），消除。

## Migration Plan

1. 发布含更新功能的版本（如 0.3.0）：旧版用户（≤0.2.3）无更新功能，维持现状手动升级一次。
2. 新版本发布后，自动检查对 0.3.0+ 用户生效；checksums.txt 随新发布流程自动附带。
3. 回滚：更新功能不改变任何既有行为，无需回滚路径；`config.json` 向后兼容（未知字段忽略、缺失字段 normalize）。

## Open Questions

无阻塞项。以下留待后续，不影响本方案落地：
- CLI `scan` 输出尾部附加"有新版可用"提示（需避免污染 `--json` 输出，v2 再议）。
- tar.gz/portable 用户的二进制自动替换（Linux 上安全、Windows 需退出后替换，v2 候选）。
