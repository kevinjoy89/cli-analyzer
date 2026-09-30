## MODIFIED Requirements

### Requirement: 厂商排除表

应用 SHALL 维护已知 GUI 产品厂商的排除规则表。每条规则 MUST 包含厂商模式与可选例外列表。匹配 MUST 采用路径片段精确匹配（大小写不敏感）：路径任一目录片段与模式相等即命中；MUST NOT 使用子串匹配（避免 `code` 命中 `opencode`）。排除表 MUST 统一应用于可执行文件发现与数据目录归因两个环节；产品级短名条目（DataOnly）MUST 仅应用于数据目录归因环节（孤儿过滤），不得拦截可执行文件发现——短名（如 `code`、`amd`）可能是用户的真实 PATH 目录，拦截会误伤其中的工具。代理/网络类 CLI 核心（clash、v2ray、mihomo 等独立二进制）MUST NOT 列入排除表，仅其 GUI 客户端（clash-verge、v2rayN 等）可列入。例外：AI 编目条目（ai-tool-catalog）的目录归因 MUST 在排除表判定之前完成，命中编目即豁免排除，AI IDE 的数据目录不因「GUI 应用」身份被排除归因；排除表对其余非 AI 的 GUI 产品照旧生效。

#### Scenario: 命中厂商模式的安装目录整体排除

WHEN 扫描发现 PATH 目录 `C:\Program Files (x86)\NetSarang\Xshell 8\` 且排除表包含模式 `netsarang`

THEN 该目录下所有可执行文件（含 Xshell.exe、installanchorservice.exe、xftpcl.exe 等）均不作为工具扫描

#### Scenario: 数据目录命中厂商模式

WHEN 数据根下存在未认领目录 `%APPDATA%\NetSarang Computer\7\Xshell`

THEN 该目录不列为孤儿数据

#### Scenario: 片段匹配不误伤相似名称

WHEN 存在目录 `~/.config/opencode` 而排除表模式包含 `code`

THEN 该目录不受影响（片段 `opencode` 与模式 `code` 不相等）

#### Scenario: DataOnly 条目不拦截 exe 发现

WHEN 排除表含 DataOnly 模式 `iterm2` 且 PATH 存在用户自建目录 `~/bin/iterm2` 内含可执行脚本

THEN 该目录下的可执行文件仍作为工具扫描，但数据根下未认领目录 `~/.config/iterm2` 不列为孤儿数据

#### Scenario: CLI 代理核心不排除

WHEN 数据根下存在未认领目录 `~/.config/mihomo`（mihomo 核心为独立 CLI 二进制）

THEN 该目录仍为孤儿数据候选（仅 GUI 客户端目录 clash-verge/v2rayN 等被排除）

#### Scenario: AI 编目条目豁免排除表

WHEN AI 编目包含 VS Code 且排除表命中其厂商模式

THEN VS Code 的编目数据目录（如 `~/Library/Application Support/Code`）仍被归因并带 AI 语义标注
