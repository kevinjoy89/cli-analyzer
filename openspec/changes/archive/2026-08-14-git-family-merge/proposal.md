## Why

GUI 验收发现 git 相关的工具在 CLI 工具栏被拆成多个：Git for Windows 的 `cmd/` 目录（git.exe/git-lfs.exe/scalar.exe/tig.exe/start-ssh-agent.cmd/start-ssh-pageant.cmd 等）各自独立成行，curated `git` 规则行反而没有二进制。进一步定位发现更严重的问题：**Git for Windows 的全部控制台工具都链接了 user32.dll（控制台/错误处理用），被 exe 发现环节的 `importsGUI` 单库命中误判为 GUI 应用 → 整个 git 在扫描中完全不可见**。

## What Changes

- **定义（git 相关工具是否属于 CLI）**：git / git-lfs / scalar / tig / start-ssh-agent / start-ssh-pageant 是 CLI 工具，属于本应用管理范围；gitk / git-gui 是 git 自带的 GUI 伴侣，不作为独立工具（被 IsConsoleExe 的子系统判定正确排除，不进入家族）。gh / hub 是独立 CLI 产品，保持独立。
- **importsGUI 判定收紧（修复 git.exe 不可见）**：`user32.dll` 单库命中不再是 GUI 充分信号——Git for Windows 控制台工具合法链接 user32；判定改为命中 ≥2 个 GUI 导入库（真正的 CUI 伪装 GUI 应用会导入完整 Win32 界面栈 user32+gdi32+comctl32…）。
- **git 家族合并**：仿 nodejsFamily 先例新增 `gitFamily`（git/git-lfs/scalar/tig/git-receive-pack/git-upload-pack/git-upload-archive/start-ssh-agent/start-ssh-pageant），classify 在 brew/versioned/npm 等具体来源之后归并 → 一条 `git`；`gitInstallRoot` 仅当目录含 git.exe/git 时作为安装根（Git\cmd 计入 install 占用）；`probeOrder` 让 git 排到 Binaries[0]（版本探测取 `git --version`）。brew 公式 git-lfs/tig 保持独立（brew 分支优先）。
- **nodejs 辅助脚本并入**：install_tools.bat / nodevars.bat（官方安装器自带，同在安装目录）并入 nodejs 家族，消除噪音行。

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
- `tool-attribution`: git 家族归并（与 nodejs 家族同机制）；Windows exe 发现环节的 GUI 判定从单库命中收紧为双库命中

## Impact

- **代码**：`internal/platform/pe_imports_windows.go`（importsGUI ≥2）、`internal/scanner/classify.go`（gitFamily/gitInstallRoot/probeOrder/nodejsFamily 扩充）
- **测试**：`pe_imports_windows_test.go`（多 DLL 夹具 + user32 单库保留用例）、`classify_test.go`（git 家族用例）
- **行为**：Windows 上 git 从"不可见 + 拆行"变为一行 `git`（8 个二进制合并，含 ~/.gitconfig 与 Git\cmd 安装占用）；nodejs 行收编 install_tools.bat/nodevars.bat；控制台工具若仅导入单个 GUI 库（user32）不再被误杀
- **不引入**：gh/hub/yarn/pnpm 等独立产品不合并；gitk/git-gui（GUI）保持排除；brew 公式保持独立
