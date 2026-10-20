# Tier 不等于「可批量清理」

## 背景

`clean --all` 的批量选择范围由 `internal/cli/clean.go` 里一张硬编码白名单决定：

```go
cleanKinds := map[string]bool{"cache": true, "old-version": true, "backup": true,
    "download": true, "index": true, "telemetry": true, "logs": true}
```

而每条 `Actionable item` 另有一个 `Tier`（safe / user）。做用户规则包时，很自然会想「用 Tier 推导批量范围，把这张白名单删掉」——白名单里的 7 种恰好**全是 safe**，`--include-data` 加入的 10 种恰好**全是 user**，看起来是冗余副本。

## 决定

**不按 Tier 推导，保留显式白名单**，并把它外置到配置。

### 为什么看起来能推导，实际不能

规则表里 `kind → tier` 映射确实是**一对一**的（实测 14 种 kind 无一跨 tier），所以「Tier 能唯一决定 kind 集合」成立。但它推导出的集合**不等于**批量清理范围：

| Kind | Tier | 在白名单 | 按 Tier 推导的后果 |
|---|---|---|---|
| `runtime` | **safe** | 否 | **会被误纳入** — 删除立即破坏工具可用性 |
| `old-version` | — | 是 | 扫描期**动态合成**的 kind，规则表里没有静态 Tier 可查 |
| `state` | — | 仅 `--include-data` | 代码里已不产生，白名单里的死条目 |

`runtime` 是决定性的反例。它标 `safe` 是对的——捆绑运行时确实可重新下载；但「可重下」不等于「适合放进一键批量清理」，删除会让工具当场不可用。

**两个问题不同：**
- `Tier` 回答「这个东西删了能不能恢复 / 是不是用户数据」
- 批量清理资格回答「默认替用户全选时，该不该包含它」

前者是对象的属性，后者是**默认策略**。把策略塞进对象属性里，就再也表达不出「safe 但不默认选中」这种正当情况。

## Considered Options

- **按 Tier 推导（safe 即批量可清）**：消灭两份硬编码（CLI 与 GUI），最简洁。但会把 `runtime` 误纳入默认批量，且 `old-version` 这类动态 kind 无处挂 Tier。**否决。**
- **把 `KindRuntime` 的 Tier 改成 user**：让两者一致。但 `runtime` 确实可重下，改成 user 会污染 Tier 的语义，还会让它进 `--include-data` 的 user 分支——行为反而更糟。**否决。**
- **保留显式白名单，外置到 `config.json`**（已选）：`clean.bulkKinds` 追加、`clean.bulkExclude` 剔除，内置 7 种仍是默认。用户规则包声明的自定义 Kind 因此能进入批量清理，而不需要改规则包 schema。
- **在规则包里给 kind 加 `bulk` 标记**：与「一条规则可能产生多种 kind」冲突（`old-version` 是运行时合成的），且要改规则包格式。**否决。**

## 后果

- 术语表新增 **Bulk-cleanable（可批量清理）**，并明确 `Tier` 只是「能否安全删除」的标签。
- 新增 Kind 默认**不可**批量清理——这是安全侧默认：用户自己加的规则不会因为标了 safe 就被 `--all` 一并清掉。
- **GUI 没有批量选择功能**（工具视图只能逐项勾选；「全选」只存在于未认领数据视图，与 Kind 无关）。因此本决定当前只影响 CLI，不存在两份白名单需要统一。若将来 GUI 加「全选可清理项」，必须调用 `internal/cleanpolicy` 而不是另写一份。
