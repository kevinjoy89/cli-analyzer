# tool-upgrade（增量：代跑命令健壮性）

## MODIFIED Requirements

### Requirement: 升级命令展示与代跑

系统 SHALL 按安装来源展示该工具的官方升级命令；对 brew/npm/pipx/cargo 来源 SHALL 提供代跑能力（以解析后的可执行文件结合增强 PATH 执行）；对命中已知官方脚本表的 local-bin 工具 SHALL 展示「重跑官方脚本」命令但不代跑；对 go、未知 local-bin、versioned、other 等来源 SHALL 展示提示而不编造具体命令。npm 等按包名寻址的来源，其包名无法解析时（扫描缓存缺失且逆映射未命中）SHALL 降级为不可代跑的提示文案；代跑命令 MUST NOT 携带空参数。代跑时若命令在增强 PATH 中无法解析出可执行文件，系统 SHALL 报明确错误并提示用户复制命令自行执行，MUST NOT 静默以裸名执行。

#### Scenario: brew 可代跑

- **WHEN** 工具安装来源为 brew 且用户选择代跑
- **THEN** 系统执行 `brew upgrade <公式名>` 并流式展示执行输出

#### Scenario: npm 可代跑

- **WHEN** 工具安装来源为 npm 且用户选择代跑
- **THEN** 系统执行 `npm update -g <包名>` 并流式展示执行输出

#### Scenario: npm 包名不可解析时降级为提示

- **WHEN** npm 来源工具的真实包名无法解析（旧缓存缺 Package 字段且工具名不在逆映射表）
- **THEN** 系统展示不可代跑的提示文案，不产出携带空参数的代跑命令

#### Scenario: PATH 解析失败明确报错

- **WHEN** 用户选择代跑但命令对应的可执行文件在增强 PATH 中不存在
- **THEN** 系统报明确错误并提示复制命令自行执行，不发起注定失败的子进程

#### Scenario: local-bin 已知官方脚本

- **WHEN** 工具的 local-bin 来源命中已知官方脚本表
- **THEN** 系统展示该官方脚本命令（如 uv 的官方安装脚本），但不代跑执行

#### Scenario: go 来源仅提示

- **WHEN** 工具安装来源为 go
- **THEN** 系统展示「重新执行当时的 go install（需模块路径）」提示，不给出编造的具体命令
