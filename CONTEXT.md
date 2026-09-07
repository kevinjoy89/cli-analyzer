# CLI Analyzer

识别本机上通过各类包管理器与脚本安装的 CLI 工具，量化其磁盘占用、区分可清理数据，并代跑标准卸载/升级命令。CLI 与 GUI（Wails）共用同一套领域模型。

## Language

### 扫描与归因

**Tool（工具）**:
扫描结果中被归因到同一身份的一组二进制与数据目录；有唯一的展示名（ToolID）。
_Avoid_: 程序、软件、binary

**Binary（二进制）**:
PATH 上发现的一个可执行入口；含入口路径（Path）与真实路径（Real，符号链接解出后）。
_Avoid_: 命令（命令名只是它的 Name）

**Installer（安装来源）**:
工具的安装渠道（brew/npm/pipx/cargo/go/pyenv/rustup/local-bin/versioned/other），决定卸载与升级命令的映射。
_Avoid_: 来源（口语可用，术语统一为 Installer）、package manager

**Footprint（总占用）**:
工具二进制与其归因数据目录的磁盘总字节数。
_Avoid_: 大小、size（易与单目录字节数混淆）

**Cleanable（可清理）**:
工具名下按清理规则（缓存/日志/旧版本等）识别出的可安全删除的数据条目。
_Avoid_: 垃圾、junk

**User bytes（用户数据）**:
工具名下不可再生成的用户数据字节数（配置、会话历史等），清理建议中只提示不代删。
_Avoid_: 用户占用

**Orphan（孤儿数据）**:
数据根下未被任何 Tool 认领的顶层目录（经非 CLI 排除体系过滤后仍无主者）。
_Avoid_: 未归因数据展示文案之外的"未识别工具"；术语统一为 Orphan / Unattributed

**Residue（残留）**:
工具卸载后仍存在的数据目录（规则表数据目录 ∪ 扫描快照归因目录，卸载后仍存在者）。
_Avoid_: 残留物、leftover

### 处置

**Built-in Trash（内置回收站）**:
应用自管的延迟删除区：移入的条目保留期可恢复，到期按配置清理。
_Avoid_: 回收站（单独说"回收站"时指系统回收站 System Trash）

**System Trash（系统回收站）**:
操作系统提供的回收站（macOS Finder 废纸篓 / Windows Recycle Bin）；仅过期清理的默认去向与降级路径。
_Avoid_: 废纸篓、Recycle Bin

**Purge（彻底删除）**:
跳过一切回收站的不可恢复删除；仅用户显式确认触发。
_Avoid_: 清空（"清空回收站"即 Purge 全部项）、永久删除以外的模糊说法

**Upgrade（升级）**:
将已安装工具更到新版本的代跑动作（brew upgrade / npm update -g / 官方自升级子命令）。
_Avoid_: 更新（Update 专指应用自身）

**Update（应用更新）**:
cli-analyzer 自身的版本检查、下载、校验与安装。
_Avoid_: 与工具 Upgrade 混用"更新"

**Probe（探测）**:
运行工具的 --version/-h 等参数以获取版本号或自升级子命令；带超时与进程组击杀，结果缓存。
_Avoid_: 检测（区分于 Update 的"检查更新"）

### 数据

**Fingerprint（指纹）**:
PATH 目录集合的变更指纹（mtime+size 摘要）；未变化时扫描直接返回缓存。
_Avoid_: hash、签名

**Snapshot / History（快照/历史）**:
一次扫描的持久化记录（整体与逐工具字节数），供占用趋势与增量排行使用。
_Avoid_: 缓存（缓存 cache 指上次扫描结果的复用副本，不进入趋势）
