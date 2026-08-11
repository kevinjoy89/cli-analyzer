# 打包与发布

CLI Analyzer 是单一二进制双接口（CLI + GUI）。CLI 部分为纯 Go，可任意交叉编译；GUI 部分依赖各平台原生 webview（macOS WebViewKit / Windows WebView2 / Linux WebKitGTK），**必须用目标平台的构建环境**（本机或 CI 原生 runner）执行 `wails build`。因此跨平台打包的核心是：每个平台各打包一次这一个可执行文件。

## 前置条件

```bash
brew install go wails create-dmg        # macOS 打包必需
export PATH="/opt/homebrew/bin:$PATH:$HOME/go/bin"
```

> 中国大陆网络：Go 依赖拉取失败时加 `GOPROXY=https://goproxy.cn,direct`；npm 走镜像同理。

## macOS

本机即可产出 **通用二进制**（x86_64 + arm64 合一）的 dmg 安装包：

```bash
./scripts/build-dmg.sh                  # → dist/CLI Analyzer-<版本>.dmg
```

脚本步骤（对应 `scripts/build-dmg.sh`）：

1. `wails build -platform darwin/universal` —— 一条命令出双架构通用二进制
2. `create-dmg` 打包 —— 卷图标用应用图标，600×400 窗口，app 图标 + Applications 拖放链接
3. 清理 `build/bin/CLI Analyzer.app` 中间产物（同时避免它被 Spotlight 索引成重复应用）

dmg 版本号自动从 `wails.json` 的 `productVersion` 读取。

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

需在 **Windows 宿主**（或 GitHub Actions `windows-latest`）上构建：

```powershell
wails build -platform windows/amd64      # → build/bin/cli-analyzer.exe
```

分发方式（任选）：

- **便携 zip**（最简单）：把 exe 与 `build/windows/icon.ico` 一起压缩，解压即用
- **Inno Setup**（向导安装器）：脚本见下节示例，产出 setup.exe
- **MSIX / WiX**：面向商店分发

安装器脚本要点（Inno Setup `.iss`）：`[Files]` 引用 exe 与 icon，`[Icons]` 建开始菜单/桌面快捷方式，`[Registry]` 可选加 PATH。

> 签名：无证书时 SmartScreen 会弹「未知发布者」；`signtool sign` 用代码签名证书消除。

## Linux

构建机需安装 Wails 的 cgo 依赖：

```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev build-essential pkg-config
wails build -platform linux/amd64        # → build/bin/cli-analyzer
```

> **webkit2gtk 版本红线**：Wails v2 需要 4.1（Ubuntu 22.04+）。Ubuntu 20.04 等老发行版是 4.0，GUI 跑不起来——此时只能分发 CLI 二进制。

分发方式：

- **`.deb`**：`dpkg-deb --build` 或 `fpm -s dir -t deb`，写入 `/usr/bin` + `.desktop` 文件
- **AppImage**：`appimagetool` 打包，免安装、跨发行版
- **纯 CLI 用户**：直接发 tar.gz 里的单一二进制即可

## CI：一次构建三平台安装包

项目在 GitHub 上时，用 Actions 矩阵在原生环境各打一包，避免自备 Windows/Linux 机器：

```yaml
# .github/workflows/build.yml（骨架）
jobs:
  build:
    strategy:
      matrix:
        include:
          - os: macos-latest      # → dmg
          - os: windows-latest    # → setup.exe
          - os: ubuntu-22.04      # → .deb
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: actions/setup-node@v4
      - uses: dAppServer/wails-build-action@v2.2   # 装 wails + 依赖并构建
      # ... 各平台再跑对应打包命令，上传 artifacts
```

## 常见问题

| 现象 | 原因与处理 |
|---|---|
| 下载的 dmg 提示「已损坏/无法验证开发者」 | 未签名/公证。右键打开，或签名+公证（见上） |
| Windows 提示未知发布者 | 无代码签名证书，属正常 |
| Linux 旧发行版打不开 GUI | webkit2gtk 版本 < 4.1，仅用 CLI |
| Spotlight 搜到两个「CLI Analyzer」 | `build/bin` 里的构建中间产物被索引；`build-dmg.sh` 构建后自动删除。彻底解决：系统设置 → Siri 与 Spotlight → 聚焦隐私，把项目 `build/` 目录加入排除 |
