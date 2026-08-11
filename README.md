# CLI Analyzer

清理软件（如 CleanMyMac）能看到桌面端应用的磁盘占用，却看不到 CLI 工具的占用——因为 CLI 数据散落在主目录的隐藏点目录和 XDG 数据目录里（`~/.claude`、`~/.local/share`、`~/.npm`、`~/.cache/*`），被归类为"其他"或被忽略。

CLI Analyzer 扫描系统上安装的 CLI 工具，归因每个工具的总磁盘占用（可执行文件 + 包目录 + 数据/缓存目录），并识别**可安全清理**的空间。同一二进制既是 CLI 也是原生 GUI（Wails）。

## 功能

- **检测**：枚举 `$PATH` 可执行文件，解析符号链接，按真实路径分类到安装源（versioned 安装器 / brew formula / npm 包 / pipx / pyenv shim / go / cargo / 其他）
- **归因**：每个工具的总占用 = 可执行文件 + 包目录（Cellar、node_modules、versions/…）+ 平台数据目录（`~/.cache/<名>`、`~/.config/<名>`、`~/.local/share/<名>`、macOS `~/Library/*`、Windows `%APPDATA%`…）
- **两级清理安全模型**：
  - **SAFE**（缓存、旧版本、备份、包管理器缓存）—— 逐项确认后可删除
  - **USER**（配置、历史、venv）—— 仅展示，**任何情况下都不会被自动删除**（硬门槛在 cleaner 层，`--yes` 也无法绕过）
- **树形明细**：可清理项展开显示一级子目录占用（如 `~/.npm` → `_cacache` 10G / `_npx` 764M），可只勾选部分子项单独清理；子路径删除同样经过 SAFE 门槛与守卫（必须是扫描归因过的父项子路径）
- **双接口**：CLI（`scan` / `clean` / `cache` / `version`）+ 原生 GUI（Wails v2，暗色界面）

## 构建

```bash
# 需要 Go >= 1.22 与 npm
brew install go
export PATH="/opt/homebrew/opt/go/bin:$HOME/go/bin:$PATH"
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# CLI 快速构建
go build -o bin/cli-analyzer .

# GUI 应用（同一二进制也可作 CLI）
wails build        # → build/bin/cli-analyzer
```

跨平台安装包（macOS dmg / Windows 安装器 / Linux deb-AppImage）的打包与签名见 **[docs/packaging.md](docs/packaging.md)**；macOS 一键出包用 `./scripts/build-dmg.sh`。

> **网络镜像说明**：本项目构建**不依赖任何特定 Go 镜像**，使用 Go 默认 proxy 即可，无需任何配置。仅**中国大陆网络**用户若拉取 Go 依赖超时，请在本机执行一次：
>
> ```bash
> go env -w GOPROXY=https://goproxy.cn,direct
> ```
>
> 该配置只写入你自己的环境（`~/.config/go/env`），不进入仓库、不影响其他开发者；切到其他网络环境时可用 `go env -u GOPROXY` 恢复默认。

## 用法

```bash
cli-analyzer scan                    # 扫描（首次较慢，之后读缓存秒开）
cli-analyzer scan --refresh --json   # 强制重扫并输出 JSON
cli-analyzer clean                   # 交互式逐项确认清理 SAFE 项
cli-analyzer clean --dry-run --all   # 只显示清理计划，不删除
cli-analyzer clean --yes kimi        # 清理指定工具的全部 SAFE 项
cli-analyzer cache --clear           # 清除扫描缓存
cli-analyzer                         # 打开 GUI
```

安全模型：只有 SAFE 级（缓存/旧版本/备份/包管理器缓存）会被清理；USER 级（配置/历史/venv）仅展示。旧版本清理会自动保留当前版本（如 claude 的软链接目标）。

### 清理安全边界（经过实测核验，以下一律不列入 SAFE）

| 目录 | 为何不可自动删除 |
|---|---|
| `~/.cache/opencode`、`~/.cache/mimocode` | 名为 cache，实为 opencode 系工具的**插件安装目录**（MCP 服务器、语言服务器、skills、`package.json` 清单）。删除会丢失全部插件且清单不可恢复 |
| pyenv / rustup 旧工具链（非当前版本） | pip 安装的命令会把解释器路径写死在 shebang 里（如 16 个命令引用 `~/.pyenv/versions/3.6.15/bin/python3.6`），项目也可能用 `rust-toolchain.toml` 锁定工具链。仅展示，手动用 `pyenv uninstall` / `rustup toolchain uninstall` 移除 |
| brew 非当前 Cellar 版本 | 可能被其他 formula 依赖（fontconfig 等），删除会破坏它们；`brew cleanup` 可安全处理 |
| `~/.cache/codex-runtimes/codex-primary-runtime` | codex 正在使用的运行时，删除会导致 codex 不可用；仅清理 `codex-runtime-install-*` 暂存目录 |

## 跨平台

- 扫描核心是纯 Go 标准库，`GOOS=linux/windows go build ./...` 可直接交叉编译
- macOS：额外扫描 `~/Library/{Caches, Application Support, Preferences}`
- Linux：XDG 目录；构建 GUI 需 `libwebkit2gtk-4.1`、`libgtk-3`
- Windows：`%APPDATA%`/`%LOCALAPPDATA%`；GUI 需在 Windows 宿主构建（WebView2）

## 已知限制（v1）

- 硬链接文件按路径重复计数（大小略偏高）；后续可用 inode 去重
- 扫描结果为快照；文件变更后需 `--refresh` 或 GUI 里点"重新扫描"

## 项目结构

```
main.go             # argv 分发：scan/clean/cache/version → CLI；否则 Wails GUI
gui/service.go      # Wails 绑定（唯一 import wails 的文件）
internal/scanner/   # 发现→分类→归因→可清理判定（纯核心）
internal/rules/     # 两级规则表 + 通用解析器
internal/platform/  # 各 OS 数据根目录与可执行检测（build tags）
internal/disk/      # 并行目录大小测量（无 du 依赖）
internal/cleaner/   # SAFE 硬门槛 + 守卫 + 删除
internal/cli/       # scan / clean / cache 子命令
```
