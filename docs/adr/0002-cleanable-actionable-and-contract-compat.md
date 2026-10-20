# 领域口径：Cleanable / Actionable / Form 与 JSON 契约的兼容策略

## 背景

`CONTEXT.md` 与代码/README 在三个词上出现了漂移：

1. **Actionable** 只出现在 `README.md` 的列名（`Tool Cmd Total Actionable Install Source`）和 `internal/scanner/types.go` 的注释里（"sum of all actionable items"），而术语表只定义了 **Cleanable**。
2. **Cleanable** 的定义原本是「按清理规则识别出的数据条目」——说的是**条目**，但字段 `cleanableBytes` 是**字节合计**。`fix-cleanable-sum-overlap` 修正求和口径后，这个歧义更明显：父子条目并存时到底算几份？
3. **Form**（`add-form-factor-split`）已完整落地（`DataDir.form`、`formSubtotals`），但术语表里查不到；随之而来的 **安装占用** 也从未被定义。

不做决定的话，`add-cli-machine-contract` 要把这些量写进对外 JSON 契约，届时改名的成本会高一个量级——所以先定口径。

## 决定

**1. 把「条目」与「字节」拆成两个概念。**

- **Actionable item（可处置项）** = 一个归因目录条目（集合）。它带 Kind 标签，是用户可以勾选与处置的**单位**。
- **Cleanable（可清理）** = 全部 Actionable item 的字节合计，**按极大路径求和**：父子条目并存时子目录字节不重复计入，因此恒不超过 Footprint。

`README` 的 `Actionable` 列名指的是前者（用户勾选的东西），`types.go` 注释里的 "sum of all actionable items" 指的是后者——两处都没错，是两个词被混用了。术语表里补上 Actionable item 并互相交叉引用。

**2. `userBytes` 重新定义为 Footprint − Cleanable**（安装占用 + 用户数据）。它一直就是这么算的（`internal/scanner/types.go` 的注释即如此），只是术语表里写成了「不可再生成的用户数据」，把安装占用漏在了外面，导致 `fix-cleanable-sum-overlap` 期间出现负数钳到 0 时没人立刻发现。

**3. Form 进术语表，并补上 Install footprint。** Form = cli / desktop，一个 Tool 可同时具备两种；Form subtotal 是按 Form 分组的极大路径合计。Install footprint = CLI 安装根 ∪ 桌面端应用包。

**4. 对外 JSON 契约只增不删。** 已发布字段不得重命名或移除；表达新信息时只加可选字段；破坏性调整必须提升 `schemaVersion`。因此 `cleanableBytes` 这个**字段名保持不变**，口径变化（极大路径求和）作为修正写进 Capabilities，而不是靠改名表达。

## Considered Options

- **统一成一个词 `Cleanable`**（把 README 的 `Actionable` 降级为口语）：词表最小，但抹掉了「条目 vs 字节」这层真实差异——用户勾选的是条目，看到的数字是字节，两者在父子条目场景下数量不一致，合并会持续制造困惑。
- **拆成两个词**（已选）：多一个术语，但每个词都精确，且能解释「7.9G > 3.9G」这类 bug 的成因。
- **重命名 `cleanableBytes` → `actionableBytes`**：更贴合「条目」语义，但会打断所有下游消费者与历史快照，收益不抵成本；口径已在 Capabilities 里写清。
- **不定义 Form，等规则包外置时一起做**：`add-form-factor-split` 已经上线，拖延只会让新写的 spec 与 ADR 继续含糊。

## 后果

- `CONTEXT.md` 已就地更新（本 ADR 落地时同步）。
- `add-cli-machine-contract` 的 JSON 契约以此口径为准：`cleanableBytes` / `userBytes` / `footprintBytes` 的语义固定，新增 `schemaVersion` 与 `kind`。
- 任何未来的「可处置」相关需求，必须先明确说的是**条目集合**还是**字节合计**。
