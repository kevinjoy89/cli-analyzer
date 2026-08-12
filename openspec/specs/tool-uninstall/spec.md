# tool-uninstall Specification

## Purpose

为工具提供完整的卸载能力：按安装来源给出标准卸载命令（可代跑）、卸载后检测残留数据目录、并将残留安全移入内置回收站（可恢复）。这是唯一允许触碰 USER 级配置/凭证数据的操作，硬约束为必须经回收站，延续工具"可反悔的安全"承诺。

## Requirements

### Requirement: 标准卸载命令

系统 SHALL 按工具的安装来源（brew/npm/pipx/cargo/go/pyenv/versioned/other）提供对应的标准卸载命令。GUI 卸载流程与 CLI `uninstall` 子命令 SHALL 展示该命令；GUI SHALL 提供「执行标准卸载」选项，代跑前 SHALL 再次确认并流式展示输出。

#### Scenario: 按来源给出命令
- **WHEN** 用户对 brew 安装的工具执行卸载
- **THEN** 展示 `brew uninstall <formula>`，且命令随安装来源正确映射

#### Scenario: 代跑标准卸载
- **WHEN** 用户在 GUI 确认「执行标准卸载」
- **THEN** 系统执行对应命令并流式展示输出，成功与失败均明确反馈

#### Scenario: 卸载失败仍继续
- **WHEN** 标准卸载命令执行失败（权限、环境等）
- **THEN** 系统展示失败输出，并仍继续残留检测（残留列表不受影响）

### Requirement: 系统关键工具保护

系统 SHALL 拒绝卸载系统关键工具（python/node/git/docker/go/brew 等基础设施）与 cli-analyzer 自身，不展示卸载入口也不执行命令。

#### Scenario: 拒绝卸载系统工具
- **WHEN** 用户尝试卸载 python 或 cli-analyzer 自身
- **THEN** 系统明确拒绝并说明原因，不提供卸载流程

### Requirement: 残留检测

系统 SHALL 在卸载后检测该工具的残留数据目录，输入为双源：规则表中的数据目录规则（按平台 roots 解析）与卸载前扫描快照中归因到该工具的目录。残留 SHALL 标注安全级别，其中配置/凭证类（USER 级）SHALL 明确标注"含登录凭证"。

#### Scenario: 检测到配置残留
- **WHEN** 标准卸载完成后 `~/.config/<tool>` 仍存在
- **THEN** 系统将其列为残留并标注 USER 级/含凭证

#### Scenario: 无残留
- **WHEN** 卸载后该工具的所有已知数据目录均不存在
- **THEN** 系统提示"未发现残留"

### Requirement: 残留清理走内置回收站

系统 SHALL 将用户确认的残留项移入内置回收站（同文件系统瞬时、可恢复），而非直接删除。残留清理 SHALL 是唯一允许触碰 USER 级数据的操作，且 SHALL NOT 绕过回收站直接删除。`--yes` 或任何标志 SHALL NOT 豁免此约束。

#### Scenario: 残留移入回收站
- **WHEN** 用户确认清理残留项
- **THEN** 残留项移入内置回收站，保留期内可恢复，状态栏回收站占用更新

#### Scenario: 不允许永久删除残留
- **WHEN** 用户尝试以永久方式删除 USER 级残留
- **THEN** 系统拒绝并保持回收站路径

### Requirement: 残留确认交互

系统 SHALL 在残留清理前展示残留列表（默认全选），提供一次确认；含凭证的项 SHALL 标红并在确认文案中明确说明将删除配置类数据（可恢复）。CLI 的 `--yes` SHALL 跳过交互确认但 SHALL NOT 改变回收站约束。

#### Scenario: 默认全选 + 一次确认
- **WHEN** 残留列表弹出且用户点击确认
- **THEN** 全部默认选中的残留项移入回收站

#### Scenario: 凭证项标红提示
- **WHEN** 残留列表含 USER 级配置目录
- **THEN** 该项标红并注明"含登录凭证"，确认文案说明可恢复

### Requirement: 卸载流程自动串联

系统 SHALL 将标准卸载、残留检测、残留清理串联为一个流程：卸载动作触发后依次执行，用户无需分别调用。GUI 详情页 SHALL 提供「卸载」入口；CLI 提供 `uninstall` 子命令（`--residue` 仅检测不清理，`--yes` 跳过交互）。

#### Scenario: GUI 一键卸载
- **WHEN** 用户在详情页点击「卸载」并确认
- **THEN** 依次展示标准卸载命令 → 残留检测 → 残留清理，一次完成

#### Scenario: CLI 仅检测残留
- **WHEN** 用户执行 `cli-analyzer uninstall <tool> --residue`
- **THEN** 仅列出残留项（含级别标注），不执行删除
