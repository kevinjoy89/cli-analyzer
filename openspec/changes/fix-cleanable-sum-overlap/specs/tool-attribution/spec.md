## MODIFIED Requirements

### Requirement: 归因目录均为可处置项（Tier 为信息标签）

系统 SHALL 将工具的所有归因数据目录（安装根除外）列为可处置项；Tier（safe/user）SHALL 仅作为信息标签（"这是什么类型"），MUST NOT 作为删除门禁。同一物理路径（不同规则可能解析到同一目录）SHALL 去重。工具的 `cleanableBytes` SHALL 为其全部可处置项按**极大路径**口径之和：当某可处置项同时是另一可处置项的祖先（父子子目录规则，如 `.kimi-code` 与 `.kimi-code/bin`）时，子目录字节 MUST NOT 重复计入合计，因此 `cleanableBytes` SHALL 不大于 `footprintBytes`；`userBytes` 为 footprint 与该合计之差（约等于安装根与独立二进制）。清理时选中集合若同时包含父项与被其包含的子项/子路径，被包含者 SHALL 跳过（随父项处置），MUST NOT 产生二次删除错误。安装根（Kind install）MUST NOT 列为可处置项——删除安装根即卸载工具，归卸载流程。

#### Scenario: 配置/数据目录可处置
- **WHEN** 工具归因含 config 与 data 目录（Tier user）
- **THEN** 它们与其他可处置项同等列出、可勾选处置，不再被 cleaner 拒绝

#### Scenario: 安装根不作为可处置项
- **WHEN** 工具安装根（brew Cellar、node_modules/<pkg>、versions/…）被归因
- **THEN** 它仅作展示（安装目录区），不进入可处置列表

#### Scenario: 父子可处置项不重复计数
- **WHEN** kimi 的 `.kimi-code`（556M）与 `.kimi-code/bin`（353M）同为可处置项
- **THEN** `cleanableBytes` 按 `.kimi-code` 单份计入（不含 bin 的重复加和），且不大于 `footprintBytes`

#### Scenario: 父子同选去重
- **WHEN** 用户同时勾选 `.marscode` 与 `.marscode/.ckg` 并确认清理
- **THEN** 仅父项 `.marscode` 执行处置，`.ckg` 以「已包含于所选上级目录」跳过，不产生删除错误
