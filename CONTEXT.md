# CLI Analyzer

识别本机上通过各类包管理器与脚本安装的 CLI 工具，量化其磁盘占用、区分可清理数据，并代跑标准卸载/升级命令。CLI 与 GUI（Wails）共用同一套领域模型。

> 本文件是本项目的术语表（glossary），只定义语言，不放规格与实现决策。

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
工具二进制与其归因数据目录的磁盘总字节数，按极大路径求和（互为祖先的目录不重复计）。
_Avoid_: 大小、size（易与单目录字节数混淆）

**Install footprint（安装占用）**:
Footprint 中属于安装形态的那一部分：CLI 安装根与桌面端应用包。用户数据与可处置项都不算。
_Avoid_: 安装大小、程序占用

**Actionable item（可处置项）**:
工具名下**可以被用户处置的目录条目**（一个归因目录 = 一个可处置项），是集合而不是字节数。每个条目带 Kind 标签与处置描述，是否处置由用户决定。
_Avoid_: 可清理（那是字节合计，见 Cleanable）、垃圾项

**Tier（层级标签）**:
Actionable item 上的信息性标签：safe 表示可重新生成，user 表示用户数据。**只描述「能否安全删除」，不表示「能否默认批量删除」**——两者不同，例如 runtime 是可重下的 safe，但删除会立即破坏工具可用性。
_Avoid_: 安全等级、风险级别（会被误读成删除门槛）

**Bulk-cleanable（可批量清理）**:
某个 Kind 是否会被 `clean --all` 默认选中。与 Tier 无关，由内置白名单与用户配置共同决定。新增 Kind 默认**不可**批量清理，需显式声明。
_Avoid_: 可清理（那是字节合计）、safe（那是 Tier）

**Cleanable（可清理）**:
工具名下全部 Actionable item 的字节合计，按极大路径口径求和——父子条目并存时子目录字节不重复计入，故恒不超过 Footprint。
_Avoid_: 垃圾、junk、可处置（指条目集合时用 Actionable item）

**User bytes（用户数据）**:
Footprint 减去 Cleanable 后的余额：安装占用与不可再生成的用户数据（配置、会话历史等）。
_Avoid_: 用户占用、残留字节

**Form（形态）**:
同一个 Tool 的存在形式。CLI form 指 PATH 上的可执行入口与其 dotdir/XDG 数据；Desktop form 指桌面端宿主应用包与其 AppSupport 数据。一个 Tool 可同时有两种形态。
_Avoid_: 端、版本（形态不是版本）

**Form subtotal（形态小计）**:
按 Form 分组的占用合计，用于回答「桌面端占多少、CLI 占多少」；同样按极大路径求和。
_Avoid_: 形态大小

**Form version（形态版本）**:
某个 Form 自己的版本号。双形态工具的两个 Form **可以不同版本**（同一工具的 CLI 与桌面端跑不同构建）；没有那个形态就没有那个版本的版本。
_Avoid_: 双版本（口语可用）、版本号（未区分归属时会歧义）

**Orphan（孤儿数据）**:
数据根下未被任何 Tool 认领的顶层目录（经非 CLI 排除体系过滤后仍无主者）。
按**数据根**看：它没有 Tool 身份。与 Residue 的区别：Orphan 是「无主」，Residue 是「有主但工具已不在」。同一个目录只会落在其中一边。
_Avoid_: 未归因数据展示文案之外的"未识别工具"；术语统一为 Orphan / Unattributed

**Residue（残留）**:
工具卸载后仍存在的数据目录（规则表数据目录 ∪ 扫描快照归因目录，卸载后仍存在者）。
按**已知工具**看：它有 Tool 身份（规则表里查得到），只是那个工具的二进制与桌面端都不在了。与 Orphan 的区别见上条。
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
将已安装工具更到新版本的代跑动作（brew upgrade / npm update -g / 官方自升级子命令）。命令面：`upgrade check <tool>` / `upgrade run <tool>`（旧的 `update check <tool>` / `update run <tool>` 为兼容别名）。
_Avoid_: 更新（Update 专指应用自身）

**Update（应用更新）**:
cli-analyzer 自身的版本检查、下载、校验与安装。命令面：`update check`（不带工具参数）。
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
