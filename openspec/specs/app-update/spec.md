# app-update Specification

## Purpose

为应用内置版本更新能力：自动检查新版本（默认开启、可关闭）、提示下载、展示下载进度、校验完整性，并在用户确认后打开安装包完成手动安装。让"发现新版本"从手动上 GitHub 查看变为应用内流程，同时把"安装"这一高风险动作保留在系统原生安装流程中。

## Requirements

### Requirement: 自动检查更新

系统 SHALL 在 GUI 启动时后台静默检查是否有新版本，且 SHALL 允许用户在首选项中开启或关闭该行为（默认开启）。检查 SHALL 不阻塞界面、不因网络失败打扰用户。系统 SHALL 缓存检查结果，距上次成功检查不足 24 小时时 SHALL 跳过网络请求。

#### Scenario: 启动时自动检查发现新版本
- **WHEN** 用户启动应用且「自动检查更新」为开启状态，且距上次检查超过 24 小时
- **THEN** 系统在后台查询最新版本，若存在新版本则弹出提示（含当前版本与最新版本），提供「下载」与「忽略该版本」选项

#### Scenario: 24 小时内不重复请求
- **WHEN** 距上次成功检查不足 24 小时
- **THEN** 系统使用缓存结果，不发起网络请求

#### Scenario: 关闭自动检查
- **WHEN** 用户在首选项中关闭「自动检查更新」
- **THEN** 系统启动时不再发起更新检查，且不影响「检查更新…」手动入口

#### Scenario: 网络失败静默
- **WHEN** 更新检查请求失败（网络不可达、超时、HTTP 错误）
- **THEN** 系统静默放弃本次检查，不弹出任何提示，不影响应用其他功能

### Requirement: 版本比较与发布过滤

系统 SHALL 按语义化版本（major.minor.patch）比较当前版本与最新版本，且 SHALL 忽略 GitHub 上的草稿（draft）版本与预发布（prerelease）版本，只对正式发布版本发起提示。

#### Scenario: 有更新的正式版本
- **WHEN** 最新正式发布版本为 v0.3.0，当前版本为 0.2.3
- **THEN** 系统判定存在新版本并提示用户

#### Scenario: 跳过草稿与预发布
- **WHEN** 最新的发布是 draft 或 prerelease，其后存在更新的正式版本
- **THEN** 系统以最新的正式版本作为比较对象，不提示草稿/预发布版本

#### Scenario: 已是最新
- **WHEN** 当前版本与最新正式版本相同或更高
- **THEN** 系统不提示更新

### Requirement: 安装来源标识

系统 SHALL 在构建期注入安装来源标识（deb/tarball/installer/portable/dmg），用于在更新时选择与当前安装方式匹配的产物。来源标识未知时，系统 SHALL 依次尝试运行时探测；仍无法确定时 SHALL 打开 Release 页面而非猜测下载产物。

#### Scenario: 按安装来源选择产物
- **WHEN** 当前安装来源为 deb，且存在新版本
- **THEN** 系统下载 Linux 的 deb 安装包

#### Scenario: 标识未知时运行时探测
- **WHEN** 安装来源标识为 unknown，且系统检测到当前可执行文件由 dpkg 包管理器管理
- **THEN** 系统按 deb 产物处理

#### Scenario: 标识未知且无法探测
- **WHEN** 安装来源标识为 unknown，且运行时探测无法确定安装方式
- **THEN** 系统打开新版本的 Release 页面，由用户自行选择下载，不猜测产物

### Requirement: 下载与进度

系统 SHALL 在用户确认下载后下载对应平台与安装来源的 release 产物到用户下载目录，SHALL 展示下载进度（已下载/总量），并 SHALL 允许用户在下载过程中取消。下载失败 SHALL 不中断应用。

#### Scenario: 显示下载进度
- **WHEN** 用户点击「下载」且下载进行中
- **THEN** 系统展示进度条，实时反映已下载字节与总字节

#### Scenario: 取消下载
- **WHEN** 用户在下载过程中点击取消
- **THEN** 系统中止下载并删除已下载的不完整文件，应用保持正常运行

#### Scenario: 下载失败
- **WHEN** 下载中断或服务器返回错误
- **THEN** 系统提示下载失败，删除不完整文件，应用保持正常运行

### Requirement: 下载完整性校验

系统 SHALL 在下载完成后、提供安装引导前，校验下载产物的 SHA256 与发布中附带的校验和一致。校验和缺失或校验失败时，系统 SHALL NOT 提供「立即安装」入口。

#### Scenario: 校验通过
- **WHEN** 下载完成且 SHA256 与发布附带的 checksums 文件一致
- **THEN** 系统提示「立即安装」/「稍后」

#### Scenario: 校验和缺失或失败
- **WHEN** 发布未附带校验和，或校验和不匹配
- **THEN** 系统提示校验失败，不提供「立即安装」，并给出 Release 页面入口

### Requirement: 安装引导

系统 SHALL 在用户点击「立即安装」后，先调用系统默认方式打开已下载的安装包（macOS 打开 dmg、Windows 打开安装器、Linux 打开 deb/压缩包），再退出应用。后续安装步骤 SHALL 由用户通过系统原生流程手动完成。对于压缩包形式的产物（tar.gz/zip），系统 SHALL 额外展示当前二进制文件路径，辅助用户手动替换。

#### Scenario: 打开安装包并退出
- **WHEN** 用户点击「立即安装」
- **THEN** 系统以系统默认程序打开安装包，随后退出应用

#### Scenario: 压缩包产物展示二进制路径
- **WHEN** 安装来源为 tarball 或 portable 且用户点击「立即安装」
- **THEN** 系统在提示中展示当前可执行文件路径，并打开该压缩包

### Requirement: 忽略版本

系统 SHALL 在用户选择「忽略该版本」后记住该版本号，在出现比它更新的版本前不再提示。

#### Scenario: 忽略后不再提示该版本
- **WHEN** 用户对 v0.3.0 选择「忽略该版本」，之后应用重启且最新版本仍为 v0.3.0
- **THEN** 系统不再次提示

#### Scenario: 忽略后出现更新版本
- **WHEN** 用户已忽略 v0.3.0，之后最新版本变为 v0.4.0
- **THEN** 系统正常提示 v0.4.0

### Requirement: 手动检查更新

系统 SHALL 提供不依赖自动检查开关的手动检查入口：GUI 的「检查更新…」菜单项与 CLI 的 `update check` 子命令。手动检查 SHALL 不受 24 小时缓存限制，CLI 模式 SHALL 以 JSON 输出结果供脚本使用。

#### Scenario: GUI 手动检查
- **WHEN** 用户点击菜单「检查更新…」
- **THEN** 系统立即发起检查（不受缓存限制），并展示与自动检查一致的提示

#### Scenario: CLI 手动检查
- **WHEN** 用户在终端执行 `cli-analyzer update check --json`
- **THEN** 系统输出 JSON（当前版本、最新版本、是否有更新、产物下载地址等），退出码反映是否存在更新

### Requirement: 版本与来源可见性

系统 SHALL 在 `cli-analyzer version` 输出中附带当前安装来源。

#### Scenario: version 输出安装来源
- **WHEN** 用户执行 `cli-analyzer version`
- **THEN** 输出包含版本号与安装来源，如 `cli-analyzer 0.2.4 (linux, deb)`
