## Why

`clean --all` 的批量选择范围由 `internal/cli/clean.go` 里一张硬编码白名单决定，GUI 侧另有一套。用户规则包（`rules.d/*.json`）已经能声明自定义 Kind，但那个 Kind **永远进不了批量清理**——规则包只能影响归因，不能影响批量处置。这是上一轮规则包功能的闭环缺口。

同时查证发现一个被长期误用的等价关系：`Tier`（safe / user）被当成「能否批量清理」用了。两者不是一回事——`runtime` 标为 `safe`（捆绑运行时可重新下载）但删除会立即破坏工具可用性，所以它明确不在批量白名单里。见 ADR 0005。

## What Changes

- 新增 `internal/cleanpolicy` 包：批量清理资格的**唯一判定处**，CLI 与 GUI 共用。
- 内置默认仍是今天那 7 种 Kind（cache / old-version / backup / download / index / telemetry / logs），行为不变。
- 配置新增两组字段，与内置默认正交合并：
  - `clean.bulkKinds`：**追加**可批量清理的 Kind（用户自定义 Kind 走这里）
  - `clean.bulkExclude`：**剔除**内置默认里的 Kind
- 合并规则：`(内置默认 ∪ bulkKinds) − bulkExclude`。
- 新增 Kind 默认**不可**批量清理（安全侧默认）：用户加的规则即使标了 safe，也不会被 `--all` 一并清掉，除非显式声明。
- CLI 与 GUI 都改为调用 `cleanpolicy`，删除两处各自的白名单副本。

## Capabilities

### New Capabilities
- `bulk-clean-policy`: 批量清理资格的判定口径（内置默认 + 配置追加 + 配置剔除）、新增 Kind 的默认值、CLI 与 GUI 必须共用同一判定。

### Modified Capabilities
- 无。`tool-attribution` / `trash-recycle-bin` 的领域口径不变；`clean --all` 在默认配置下的**行为完全不变**。

## Impact

- 新增 `internal/cleanpolicy/cleanpolicy.go` + 测试。
- `internal/config/config.go`：新增 `CleanConfig`（`bulkKinds` / `bulkExclude`）。
- `internal/cli/clean.go`：删除 `cleanKinds` 硬编码，改调 `cleanpolicy.BulkEligible`。
- `gui/service.go`：批量选择改调同一函数。
- `internal/i18n/locales/*`：`--include-data` 的说明文案微调（它现在只影响 user 类，不再影响 Kind 白名单）。
- 文档：`README` / `docs/user-rules.md` 说明自定义 Kind 如何进入批量清理。

## 不在本次范围

- `Tier` 语义调整（`runtime` 保持 safe，不改成 user）。
- 回收站里 Kind 的分组展示。
- 规则包里声明 `bulk` 标记——与「一条规则可能产生多种 Kind」冲突（`old-version` 是扫描期动态合成的），已在 ADR 0005 记录否决理由。
