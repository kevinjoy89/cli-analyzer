# 打包与发布

CLI Analyzer 是单一二进制双接口（CLI + GUI）。CLI 部分为纯 Go，可任意交叉编译；GUI 部分依赖各平台原生 webview（macOS WebViewKit / Windows WebView2 / Linux WebKitGTK），**必须用目标平台的构建环境**（本机或 CI 原生 runner）执行 `wails build`。因此跨平台打包的核心是：每个平台各打包一次这一个可执行文件。

## 版本与安装来源注入（重要）

自 v0.3.0 起，版本号与**安装来源标识**由构建期 `-ldflags` 注入（单一事实来源）：

```
-X cli-analyzer/internal/buildinfo.Version=<tag 版本>       # 如 0.3.3
-X cli-analyzer/internal/buildinfo.InstallSource=<来源>     # deb|tarball|installer|portable|dmg
```

- 版本号：tag 是唯一来源。CI 在构建前用 `sed` 把 tag 版本写入 `wails.json` 的 `productVersion`（安装包元数据 + dmg 文件名），再由 `build-dmg.sh` / 各平台构建命令注入 `Version`。
- 安装来源：**歧义平台需要构建两次**（同一个二进制打进两种包，来源标识不同），更新功能据此选择与用户安装方式匹配的产物：

| 平台 | 来源值 | 产物 |
|---|---|---|
| macOS | `dmg` | `CLI-Analyzer-<v>-macos-{arm64,amd64}.dmg`（无歧义，一次构建） |
| Windows | `installer` / `portable` | `...-windows-amd64-installer.exe`（NSIS）/ `...-windows-amd64-portable.zip` |
| Linux | `deb` / `tarball` | `...-linux-amd64.deb` / `...-linux-amd64.tar.gz` |

- 本地 `go build`（无 ldflags）时版本为 `dev`、来源为 `unknown`——此时更新检查会跳过（无法比较版本）。
- `cli-analyzer version` 输出即验证注入结果：`cli-analyzer 0.3.3 (darwin, dmg)`。

## 前置条件

```bash
brew install go wails create-dmg        # macOS 打包必需
export PATH="/opt/homebrew/bin:$PATH:$HOME/go/bin"
```

> 中国大陆网络：Go 依赖拉取失败时加 `GOPROXY=https://goproxy.cn,direct`；npm 走镜像同理。

## macOS

本机即可产出 dmg 安装包。默认打 **通用二进制**（arm64 + x86_64 合一），也可按芯片分开打包：

```bash
./scripts/build-dmg.sh                          # → dist/CLI Analyzer-<版本>.dmg（universal）
./scripts/build-dmg.sh arm64                    # → dist/CLI Analyzer-<版本>-arm64.dmg（Apple Silicon）
./scripts/build-dmg.sh amd64                    # → dist/CLI Analyzer-<版本>-amd64.dmg（Intel）
```

脚本步骤（对应 `scripts/build-dmg.sh`）：

1. 从 `wails.json` 读 `productVersion`，经 `-ldflags` 注入 `Version` 与 `InstallSource=dmg`
2. `wails build -platform darwin/<universal|arm64|amd64>` —— 按目标架构出包
3. `create-dmg` 打包 —— 卷图标用应用图标，600×400 窗口，app 图标 + Applications 拖放链接
4. 清理 `build/bin/CLI Analyzer.app` 中间产物（同时避免它被 Spotlight 索引成重复应用）

> 卷名覆盖：本地测试构建时若目标卷名已被挂载的 dmg 占用（create-dmg 会卸载失败），用 `VOLNAME="CLI Analyzer-Test" ./scripts/build-dmg.sh arm64` 换个卷名避开。

### 签名与公证（对外分发才需要）

本机自用无需签名。若要把 dmg 发给其他 Mac（U 盘 / AirDrop / 网络下载），Gatekeeper 会拦截未公证应用：

```bash
# 1. 应用签名（需 Apple Developer 账号的 Developer ID 证书）
codesign --deep --force --options runtime --sign "Developer ID Application: <你的名字>" \
  "build/bin/CLI Analyzer.app"

# 2. 公证 + 装订（上传给 Apple 校验，通常几分钟）
ditto -c -k --keepParent "build/bin/CLI Analyzer.app" "/tmp/CLI Analyzer.app.zip"
xcrun notarytool submit "/tmp/CLI Analyzer.app.zip" \
  --apple-id <你的AppleID> --team-id <TEAMID> --password <app专用密码> --wait
xcrun stapler staple "build/bin/CLI Analyzer.app"

# 3. 然后再 create-dmg（见脚本）
```

## Windows

需在 **Windows 宿主**（或 GitHub Actions `windows-latest`）上构建。**构建两次**注入不同来源标识：

```powershell
# 1. installer flavor：NSIS 安装器内嵌 installer 标识的 exe
wails build -platform windows/amd64 -nsis -ldflags "-X cli-analyzer/internal/buildinfo.Version=<v> -X cli-analyzer/internal/buildinfo.InstallSource=installer"
# 2. portable flavor：重建普通 exe（portable 标识）后打 zip
wails build -platform windows/amd64 -ldflags "-X cli-analyzer/internal/buildinfo.Version=<v> -X cli-analyzer/internal/buildinfo.InstallSource=portable"
Compress-Archive -Path build/bin/cli-analyzer.exe, build/windows/icon.ico -DestinationPath CLI-Analyzer-<v>-windows-amd64-portable.zip
```

> `wails -nsis` 需要 makensis（NSIS）在 PATH 里；安装器产物为 `build/bin/<产品名>-<arch>-installer.exe`。
> 签名：无证书时 SmartScreen 会弹「未知发布者」；`signtool sign` 用代码签名证书消除。

## Linux

构建机需安装 Wails 的 cgo 依赖。同样**构建两次**：

```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential pkg-config
# deb flavor
wails build -platform linux/amd64 -ldflags "-X cli-analyzer/internal/buildinfo.Version=<v> -X cli-analyzer/internal/buildinfo.InstallSource=deb"
# tarball flavor
wails build -platform linux/amd64 -ldflags "-X cli-analyzer/internal/buildinfo.Version=<v> -X cli-analyzer/internal/buildinfo.InstallSource=tarball"
```

分发方式：

- **`.deb`**：`dpkg-deb --build` 写入 `/usr/bin` + `.desktop` 文件（见 `release.yml` 实际步骤）
- **便携 tar.gz**：直接打包单一二进制

> **webkit2gtk 版本红线**：Wails v2 需要 4.1（Ubuntu 22.04+）。Ubuntu 20.04 等老发行版是 4.0，GUI 跑不起来——此时只能分发 CLI 二进制。

## CI：tag 触发三平台发布（`release.yml`）

打 `v*` tag 即触发 GitHub Actions 全流程（与 `docs/packaging.md` 旧版不同，实际工作流是 `release.yml`，tag 驱动、非手动 build）：

1. **Derive version**：`VERSION=${GITHUB_REF_NAME#v}`
2. **Stamp version**：`sed` 把 `VERSION` 写入 `wails.json` 的 `productVersion`（tag 单一来源）
3. 三平台 job 并行：
   - **macOS**：`build-dmg.sh arm64/amd64`（注入 `dmg` 标识）→ 两个 dmg
   - **Windows**：双构建（`installer` / `portable`）+ NSIS
   - **Linux**：双构建（`deb` / `tarball`）
4. **Publish job**：下载三平台产物 → **汇总生成唯一的 `checksums.txt`**（对全部 6 个产物逐文件 SHA256，供更新功能校验下载）→ 创建 **draft** release（softprops，附双语发布介绍后手动/命令 Publish）

> 历史教训（勿回退）：checksums.txt 必须由 release job 汇总生成，**不能**各平台 job 各自生成——同名文件上传时会互相覆盖，只剩一个平台的条目，导致其他平台产物无校验覆盖（v0.3.0 曾因此翻车，已修）。

## 发布 playbook

```bash
git tag v0.3.3 && git push origin main && git push origin v0.3.3   # 触发 CI
# CI 全绿后（约 3 分钟）：
# 1. 核对 draft release 产物：6 个安装包 + checksums.txt（应覆盖 6/6）
# 2. 按历史模板写双语发布介绍（中/英两段：标题 → 变更 → 下载 → 未签名提示）
# 3. gh release edit <tag> --notes-file notes.md --draft=false   # Publish
```

更新功能闭环依赖：release 必须附带 `checksums.txt`，否则用户在下载完成后会因校验信息缺失拿不到安装入口（安全优先，故意如此）。

## 常见问题

| 现象 | 原因与处理 |
|---|---|
| 下载的 dmg 提示「已损坏/无法验证开发者」 | 未签名/公证。右键打开，或签名+公证（见上） |
| Windows 提示未知发布者 | 无代码签名证书，属正常 |
| Linux 旧发行版打不开 GUI | webkit2gtk 版本 < 4.1，仅用 CLI |
| Spotlight 搜到两个「CLI Analyzer」 | `build/bin` 里的构建中间产物被索引；`build-dmg.sh` 构建后自动删除。彻底解决：系统设置 → Siri 与 Spotlight → 聚焦隐私，把项目 `build/` 目录加入排除 |
| 本地 `build-dmg.sh` 报「资源忙」无法卸载临时卷 | 已有一个同名卷被挂载（如正在使用的测试 dmg）。用 `VOLNAME` 换卷名，或先推出占用卷 |
| 更新检查提示「已是最新」但刚发了新版本 | 24h 限流缓存尚未刷新；稍后或手动「检查更新」 |
