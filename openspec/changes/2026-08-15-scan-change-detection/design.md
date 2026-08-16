# Design: 扫描增量优化（mtime 指纹变更检测）

## Context

现状：GUI 每次启动都全量重扫（frontend `init()` 末尾 `rescan()`），大磁盘慢；CLI `scan` 无 `--refresh` 无条件信任缓存，数据变化后输出陈旧。二者缺一个廉价的"数据是否变化"判定。目标是：未变化 → 秒回缓存；变化 → 自动全量。判定必须足够廉价（不能比全量扫描本身还贵），否则无意义。

## Goals / Non-Goals

**Goals**
- 指纹采集只 stat 不递归，开销毫秒级（远小于全量遍历）
- 指纹命中时 GUI 启动/CLI 非刷新路径直接返回缓存，不扫描、不写历史、不重探测
- 数据变化（增删路径、二进制变化、目录 mtime 变化）时自动触发全量
- `cache --clear` 连带清指纹

**Non-Goals**
- 不做文件系统监听（fsnotify）——增加常驻状态与跨平台依赖，超出本次范围
- 不改 `last-scan.json` 格式（指纹独立文件，规避缓存迁移风险）
- 不改变手动"重新扫描"/`--refresh` 的强制全量语义
- 不引入增量扫描（只跳过"无变化"的全量，不做部分重扫）

## Decisions

### D1: 指纹文件独立（不改缓存格式）

指纹存 `last-scan.fp.json`，与 `last-scan.json` 并列。`LoadCache`/`SaveCache` 签名不变，全部消费方（GUI 启动、CLI scan、测试）零改动。`ClearCache` 顺带删除指纹文件。代价是两份文件的原子性不绑定（极端情况下指纹与缓存不一致 → 保守走全量，安全方向）。

### D2: 指纹语义（mtime + size + isDir，仅 stat）

`FingerprintEntry{Path, MTime, Size, IsDir}`。对扫描结果的测量路径集合（二进制 real、dataDirs、cleanables、孤儿路径）逐条 `os.Stat`：

- 目录 mtime 在子项**增删**时更新（NTFS/APFS/ext4 均如此）→ 覆盖"装了/删了东西"；
- 二进制 mtime+size → 覆盖"升级/替换"；
- 文件内容原地修改不改变父目录 mtime → **已知盲区**：占用大小通常也不变，漏检可接受；`--refresh`/GUI 手动重扫兜底；
- 路径不存在 → 不产生条目；`FingerprintsEqual` 把"条目缺失"判为变更（比"该路径消失但指纹恰好一致"更保守）。

### D3: `ScanIfUnchanged` 独立函数（而非 Options 标志）

`Scan` 主体抽为内部 `scan(opts, skipIfUnchanged bool)`，对外保留 `Scan` 与新增 `ScanIfUnchanged`。语义清晰、调用点意图明确（GUI 启动/CLI 非刷新用后者，手动重扫/`--refresh` 用前者），避免在 `Options` 上堆布尔。

### D4: GUI 绑定 `ScanIfChanged` + 手动按钮保持全量

新增 `ScannerService.ScanIfChanged()`：内部走 `scanner.ScanIfUnchanged`，事件契约与 `Scan` 一致（`scan:done`）。关键差异：

- **历史快照**：仅真实扫描（`prev == nil || prev.ScannedAt != res.ScannedAt`）才 `history.Record`——缓存命中不产生重复趋势点；
- **版本探测**：仅真实扫描才 `probeAll()`——缓存命中时版本探测结果也来自缓存，无需重跑；
- frontend `init()` 末尾由 `rescan()` 改为 `ScanIfChanged()`；`rescanBtn` 的 onclick 不变（`rescan()` → 强制全量）。

### D5: CLI 接入（非 `--refresh` 用 `ScanIfUnchanged`）

`runScan` 重构：`--refresh` 走 `Scan`；非 `--refresh` 走 `ScanIfUnchanged`（`--no-cache` 时跳过捷径直接 `Scan`，避免缓存语义混淆）。行为从"缓存命中即返回（可能陈旧）"变为"指纹一致秒回、变化自动重扫"。历史记录仅真实扫描时追加（`ScanIfUnchanged` 缓存命中不 Record，与 GUI 一致）。

### D6: 指纹基于未过滤结果

`scanner.go` 缓存分支已保证写入的是未过滤的 `cached`（`ToolFilter` 时重建）；指纹必须基于同一 `cached` 计算并同写，与 `ScanIfUnchanged` 的缓存命中判定保持一致，避免过滤扫描污染指纹。

### D7: 测试策略

- `fingerprint_test.go`：纯函数全平台测试（临时目录构造结果；等值/顺序无关/条目缺失/mtime 变化/size 变化/真实 stat 一致性）
- `cache_test.go` 追加：指纹往返（隔离 `XDG_CACHE_HOME` + `LOCALAPPDATA`）、`ClearCache` 联动清指纹
- `unchanged_test.go`：缓存命中（返回缓存、ScannedAt 不变）；指纹文件缺失回退（走全量、ScannedAt 变化）——后者触发一次真实扫描，断言只比较 ScannedAt，不依赖具体机器状态
- CLI 接入无新单测（行为由既有 `cli_test.go` 守护），手动验证

## Risks / Trade-offs

- [文件内容原地修改漏检] → 占用大小通常不变，影响有限；`--refresh`/GUI 手动重扫兜底；README 已知限制文档化
- [指纹与缓存极端不一致] → 保守走全量（安全方向）
- [首次运行无指纹文件] → 保守全量（不误判为"无变化"）
- [Windows 目录 mtime 粒度] → NTFS 支持子项增删更新目录 mtime；`touch` 类操作也更新；粒度足够覆盖"装/删/换"三类主要变化

## Migration Plan

无持久化格式变化（`last-scan.json` 不动）。指纹文件为新增产物：旧缓存 + 无指纹 → 首次启动走一次全量并生成指纹，此后生效。`cache --clear` 同时清两份文件。
