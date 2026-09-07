# tool-uninstall（增量：卸载代跑命令健壮性）

## MODIFIED Requirements

### Requirement: 标准卸载命令

系统 SHALL 按工具的安装来源（brew/npm/pipx/cargo/go/pyenv/versioned/other）提供对应的标准卸载命令。GUI 卸载流程与 CLI `uninstall` 子命令 SHALL 展示该命令；GUI SHALL 提供「执行标准卸载」选项，代跑前 SHALL 再次确认并流式展示输出。npm 等按包名寻址的来源，其包名无法解析时（扫描缓存缺失且逆映射未命中）SHALL 降级为不可代跑的提示文案；卸载代跑命令 MUST NOT 携带空参数。代跑时若命令在增强 PATH 中无法解析出可执行文件，系统 SHALL 报明确错误并提示用户复制命令自行执行，MUST NOT 静默以裸名执行。系统关键工具黑名单 SHALL 对畸形多层可执行扩展名（如 `python.exe.exe`）同样拦截。

#### Scenario: 按来源给出命令
- **WHEN** 用户对 brew 安装的工具执行卸载
- **THEN** 展示 `brew uninstall <formula>`，且命令随安装来源正确映射

#### Scenario: 代跑标准卸载
- **WHEN** 用户在 GUI 确认「执行标准卸载」
- **THEN** 系统执行对应命令并流式展示输出，成功与失败均明确反馈

#### Scenario: 卸载失败仍继续
- **WHEN** 标准卸载命令执行失败（权限、环境等）
- **THEN** 系统展示失败输出，并仍继续残留检测（残留列表不受影响）

#### Scenario: npm 包名不可解析时降级为提示

- **WHEN** npm 来源工具的真实包名无法解析（包名为空且逆映射未命中）
- **THEN** 系统展示不可代跑的提示文案，不产出携带空参数的代跑命令

#### Scenario: PATH 解析失败明确报错

- **WHEN** 用户选择代跑但命令对应的可执行文件在增强 PATH 中不存在
- **THEN** 系统报明确错误并提示复制命令自行执行，不发起注定失败的子进程

#### Scenario: 多层扩展名仍被拦截

- **WHEN** 工具名为带多层可执行扩展名的系统关键工具（如 `python.exe.exe`）
- **THEN** 卸载请求被系统关键工具黑名单拒绝
