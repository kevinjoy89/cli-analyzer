## Purpose

为被扫描的 CLI 工具提供按需的版本检测与官方升级：在工具详情页一键查询是否有新版本，展示或代跑官方升级命令，补齐"安装 → 升级 → 卸载"生命周期中缺失的升级一环。

## ADDED Requirements

### Requirement: 按需检测与页面守卫提醒

系统 SHALL 在用户于工具详情页点击「检查更新」时发起一次该工具的版本检测，每次点击 SHALL 发起全新查询（不缓存、不复用任何历史结果）。检测完成时，系统 SHALL 仅当用户仍停留在发起检测的工具详情页时展示结果，否则 SHALL 静默丢弃该结果。同一工具的检测进行中时，系统 SHALL 忽略新的点击。

#### Scenario: 点击触发检测
- **WHEN** 用户在 ripgrep 详情页点击「检查更新」
- **THEN** 系统立即发起对 ripgrep 的版本查询并展示进行中状态

#### Scenario: 离开页面不打扰
- **WHEN** 用户点击「检查更新」后、查询完成前离开该工具详情页
- **THEN** 查询结果被静默丢弃，不弹出任何提醒

#### Scenario: 停留在页面收到结果
- **WHEN** 查询完成且用户仍停留在该工具详情页
- **THEN** 系统展示检测结果（有更新 / 已是最新 / 无法检测）

#### Scenario: 检测进行中重复点击
- **WHEN** 同一工具的检测正在进行中，用户再次点击「检查更新」
- **THEN** 系统忽略第二次点击，不发起重复查询

### Requirement: 按安装来源检测新版本

系统 SHALL 依据工具的安装来源选择对应的检测方式，且 SHALL 使用包管理器自身的口径获取已安装版本与最新版本进行比较（brew/npm 单命令即得两者；pipx/cargo 分别从安装记录与版本源获取），不与该工具的 `--version` 探测值交叉比较。对没有版本检测能力的来源（go/versioned/other/pyenv），系统 SHALL 判定为无法检测。

#### Scenario: brew 公式有更新
- **WHEN** 被检测的 brew 来源工具被 `brew outdated` 报告为可更新
- **THEN** 系统报告存在新版本，并提供 brew 口径的已安装版本与最新版本

#### Scenario: brew 公式已是最新
- **WHEN** 被检测的 brew 来源工具在 `brew outdated` 中无输出
- **THEN** 系统报告该工具已是最新

#### Scenario: npm 全局包检测
- **WHEN** 被检测工具的安装来源为 npm
- **THEN** 系统通过 npm 的全局包过时查询判定其当前版本与最新版本

#### Scenario: pipx 包检测
- **WHEN** 被检测工具的安装来源为 pipx
- **THEN** 系统从 pipx 安装记录取已安装版本，从 pip 版本源取最新版本进行比较

#### Scenario: cargo 包检测
- **WHEN** 被检测工具的安装来源为 cargo
- **THEN** 系统从 cargo 安装列表取已安装版本，从 crates 注册表取最新版本进行比较

#### Scenario: 无检测能力来源
- **WHEN** 工具安装来源为 go、versioned、pyenv 或其他
- **THEN** 系统判定无法检测版本，仅提供升级命令或提示

### Requirement: 升级命令展示与代跑

系统 SHALL 按安装来源展示该工具的官方升级命令；对 brew/npm/pipx/cargo 来源 SHALL 提供代跑能力（以解析后的可执行文件结合增强 PATH 执行）；对命中已知官方脚本表的 local-bin 工具 SHALL 展示「重跑官方脚本」命令但不代跑；对 go、未知 local-bin、versioned、other 等来源 SHALL 展示提示而不编造具体命令。

#### Scenario: brew 可代跑
- **WHEN** 工具安装来源为 brew 且用户选择代跑
- **THEN** 系统执行 `brew upgrade <公式名>` 并流式展示执行输出

#### Scenario: npm 可代跑
- **WHEN** 工具安装来源为 npm 且用户选择代跑
- **THEN** 系统执行 `npm update -g <包名>` 并流式展示执行输出

#### Scenario: local-bin 已知官方脚本
- **WHEN** 工具的 local-bin 来源命中已知官方脚本表
- **THEN** 系统展示该官方脚本命令（如 uv 的官方安装脚本），但不代跑执行

#### Scenario: go 来源仅提示
- **WHEN** 工具安装来源为 go
- **THEN** 系统展示「重新执行当时的 go install（需模块路径）」提示，不给出编造的具体命令

### Requirement: 检测失败降级

系统 SHALL 在版本检测失败（网络不可达、查询命令缺失、超时等）时标注检测失败并继续展示升级命令或提示，且 SHALL NOT 阻塞页面的其余功能。对无法检测或检测失败的来源，系统 SHALL NOT 伪造或猜测版本信息。

#### Scenario: 网络失败仍给命令
- **WHEN** 检测查询因网络等原因失败
- **THEN** 系统提示无法检测，并仍展示该工具的官方升级命令或提示

#### Scenario: 失败但不打扰
- **WHEN** 检测失败且用户已离开发起检测的详情页
- **THEN** 系统不弹出任何错误提示，静默丢弃结果

### Requirement: 升级执行与完成反馈

系统 SHALL 在用户执行代跑升级后对该工具重新探测版本并刷新详情页版本显示，SHALL NOT 触发全量扫描。升级执行失败 SHALL 展示错误且不影响应用其余功能。

#### Scenario: 升级成功刷新版本
- **WHEN** 用户代跑升级且执行成功
- **THEN** 系统重新探测该工具版本，详情页显示新版本号，检测结果横幅同步更新

#### Scenario: 升级失败
- **WHEN** 代跑升级执行失败
- **THEN** 系统展示错误输出，应用其余功能不受影响

### Requirement: CLI 更新检查与执行

系统 SHALL 提供 `update check <工具>` 与 `update run <工具>` 子命令：`update check` 带工具名时 SHALL 检测该工具的新版本，不带工具名时 SHALL 保持现有应用自身检查行为（向后兼容），`--json` SHALL 输出结构化结果；`update run` SHALL 先展示升级命令再确认执行，`--yes` SHALL 跳过确认。退出码 SHALL 反映是否检测到新版本。

#### Scenario: 检测工具新版本
- **WHEN** 用户执行 `cli-analyzer update check ripgrep --json`
- **THEN** 系统输出 JSON（工具名、已安装版本、最新版本、是否有更新、升级命令、是否可代跑）

#### Scenario: 无参数保持应用检查
- **WHEN** 用户执行 `cli-analyzer update check`
- **THEN** 系统保持现有行为，检查应用自身是否有新版本

#### Scenario: 代跑升级
- **WHEN** 用户执行 `cli-analyzer update run ripgrep`
- **THEN** 系统展示 `brew upgrade ripgrep` 并确认后执行；传 `--yes` 时跳过确认

### Requirement: 安全与边界约束

系统 SHALL NOT 因其为系统关键工具而拦截升级（无黑名单）；SHALL NOT 缓存检测结果；SHALL NOT 在应用启动或进入详情页时自动发起检测（仅显式点击或命令触发）；SHALL NOT 在一次操作中批量升级多个工具（每个工具独立操作、独立确认）。

#### Scenario: 系统工具可升级
- **WHEN** 用户对 python、node 等系统关键工具点击「检查更新」
- **THEN** 系统正常检测与展示升级，不因黑名单拦截

#### Scenario: 无自动检测
- **WHEN** 应用启动或用户进入工具详情页
- **THEN** 系统不自动发起版本检测，等待用户显式点击

#### Scenario: 单工具独立操作
- **WHEN** 用户代跑升级
- **THEN** 系统仅执行所选工具自身的升级命令，不触及其他工具