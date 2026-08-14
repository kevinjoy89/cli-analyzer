# Design: git 家族合并 + Windows exe 发现 GUI 判定收紧

## Context

GUI 验收显示 git 相关工具被拆成多个（git 规则行无二进制、start-ssh-agent.cmd 等各自成行）。根因有两层：① `importsGUI` 把「导入 user32.dll」当作 GUI 充分信号，而 Git for Windows 的全部控制台工具（git.exe/git-lfs/scalar/tig/git-receive-pack/git-upload-pack）都链接 user32.dll（控制台与错误处理）→ 被误判为 GUI → **整个 git 不可见**；② 可见的附属命令（start-ssh-agent.cmd 等）没有家族归并机制。

## Goals / Non-Goals

**Goals**
- 修复 git.exe 等被误杀（importsGUI 判定收紧）
- git 家族归并为一行（nodejs 家族同机制）
- nodejs 安装辅助脚本（install_tools.bat/nodevars.bat）并入，消除同类噪音

**Non-Goals**
- 不合并独立产品：gh/hub/yarn/pnpm/brew 公式（git-lfs/tig）保持独立
- 不把 GUI 伴侣（gitk/git-gui）并入（它们应保持排除）

## Decisions

### D1: importsGUI 阈值 ≥2

`user32.dll` 单库命中不构成 GUI 充分信号（Git for Windows 控制台工具合法导入）；真正的 CUI 伪装 GUI 应用（Xshell 类）会导入完整界面栈。判定改为命中 ≥2 个 GUI 导入库（user32/gdi32/comctl32/uxtheme/dwmapi/d2d1）。风险：只导入单个 GUI 库的真 GUI 应用会漏网——罕见（最小 Win32 GUI 栈也需 user32+gdi32+comctl32），且厂商排除表（netsarang 等）兜底。

### D2: git 家族归并（仿 nodejs）

- `gitFamily` 名称表：git/git-lfs/scalar/tig/git-receive-pack/git-upload-pack/git-upload-archive/start-ssh-agent/start-ssh-pageant（gh/hub 是独立产品，不在表内；gitk/git-gui 是 GUI，被 IsConsoleExe 排除，不参与）
- classify 中置于 brew/versioned/npm/pipx/go/cargo 之后（具体来源优先）：brew 公式 git-lfs/tig 保持独立公式；Windows Git\cmd 命中家族
- `gitInstallRoot`：目录含 git.exe/git 时返回该目录（Git\cmd 计为 install 占用），语义与 nodejsInstallRoot 一致；不含则不设（按文件计大小）
- `probeOrder` 扩展：git 家族把 git 排到 Binaries[0]（`git --version`），与 nodejs 的 node 优先同理

### D3: nodejs 辅助脚本并入

install_tools.bat / nodevars.bat 是官方安装器自带脚本（同在安装目录），并入 nodejsFamily——与 git 家族同类的"安装自带物独立成行"噪音。

## Risks / Trade-offs

- [importsGUI 收紧漏掉单库 GUI 应用] → 罕见 + 厂商表兜底 + 子系统判定仍在
- [tig 名称并入 git（非 brew 独立安装时）] → tig 是 git 生态工具，并入无害；brew tig 保持独立
- [InstallRoot 只覆盖 Git\cmd 而非整个 Git 安装] → 与 nodejs 语义一致（保守、安全）；完整安装占用测量留作后续

## Migration Plan

无持久化格式变化；重扫即生效。GUI 工具栏 git 相关从多行合并为一行。
