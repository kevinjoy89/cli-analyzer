## ADDED Requirements

### Requirement: 批量清理资格由统一策略判定

系统 SHALL 由一个共享策略判定某个 Kind 是否具备批量清理资格，CLI 的 `clean --all` 与 GUI 的批量选择 SHALL 调用同一判定，不得各自维护白名单副本。

#### Scenario: CLI 与 GUI 一致

WHEN 同一份扫描快照与同一份配置下分别执行 CLI `clean --all --dry-run` 与 GUI 的「全选可清理项」

THEN 两者选中的条目集合完全相同

#### Scenario: 默认行为不变

WHEN 用户从未修改过配置

THEN 可批量清理的 Kind 恰好是 cache / old-version / backup / download / index / telemetry / logs，与本次变更前一致

### Requirement: 配置可追加与剔除批量清理的 Kind

配置 SHALL 提供 `clean.bulkKinds`（追加）与 `clean.bulkExclude`（剔除）两组字段，最终集合 SHALL 为 `(内置默认 ∪ bulkKinds) − bulkExclude`。

#### Scenario: 追加自定义 Kind

WHEN 用户规则包声明了 `kind: "my-cache"` 且配置里 `clean.bulkKinds` 含 `"my-cache"`

THEN `clean --all` SHALL 选中该 Kind 的条目

#### Scenario: 剔除内置 Kind

WHEN 配置里 `clean.bulkExclude` 含 `"cache"`

THEN `clean --all` SHALL NOT 选中任何 cache 类条目，即使它在内置默认里

#### Scenario: 同时追加与剔除

WHEN `bulkKinds` 含 `"my-cache"` 且 `bulkExclude` 含 `"logs"`

THEN 最终集合含 `my-cache` 而不含 `logs`

### Requirement: 新增 Kind 默认不可批量清理

未在内置默认里、也未出现在 `bulkKinds` 中的 Kind SHALL 默认不具备批量清理资格——即使它的 Tier 是 safe。

#### Scenario: 自定义 Kind 未声明时不进批量

WHEN 用户规则包声明了 `kind: "my-cache"`、`tier: "safe"`，但配置未把它加入 `bulkKinds`

THEN `clean --all` SHALL NOT 选中它（仍可用显式 id 或逐项确认处置）

#### Scenario: safe 不等于可批量清理

WHEN 某个 Kind 的 Tier 是 safe 但不在批量集合内（如内置的 `runtime`）

THEN `clean --all` SHALL NOT 选中它

### Requirement: 显式指定不受批量策略限制

批量策略 SHALL 只影响 `clean --all` 的默认选择，SHALL NOT 阻止用户显式处置任意条目。

#### Scenario: 逐项确认仍可处置

WHEN 用户对某个不在批量集合内的条目逐项回答 y

THEN 该条目 SHALL 被正常处置，不受批量策略影响

#### Scenario: 显式 id 仍可处置

WHEN 调用方传入明确的 cleanable id

THEN 处置 SHALL 照常进行，不检查批量资格
