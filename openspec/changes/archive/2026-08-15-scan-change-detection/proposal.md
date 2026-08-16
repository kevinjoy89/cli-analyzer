# 扫描增量优化：mtime 指纹变更检测

## Why

每次启动 GUI 都无条件全量重扫（`frontend/src/main.ts` `init()` 末尾调 `rescan()`）：遍历所有归因目录（node_modules、.npm、模型缓存等巨大目录），大磁盘上耗时数秒到数十秒。缓存（`last-scan.json`）只用于首屏秒开，随后必然来一遍全量 IO。CLI 侧 `scan`（无 `--refresh`）则相反——无条件信任缓存，数据变化后输出可能陈旧。两者都缺一个廉价的"数据是否变化"判定：既能免掉无谓的全量扫描，又能在数据变化时自动重扫。

## What Changes

- **mtime 指纹（新）**：`internal/scanner/fingerprint.go` 新增 `FingerprintEntry{Path, MTime, Size, IsDir}` 与 `ComputeFingerprint(res)` / `FingerprintsEqual(a, b)`。指纹 = 扫描结果中全部测量顶层路径（二进制 real、dataDirs、cleanables、孤儿路径）的 `mtime + size + isDir`，**仅 stat 不递归**（毫秒级）；按路径排序保证序列化稳定；不存在的路径不产生条目（条目缺失 = 变更）。
- **指纹持久化（独立文件）**：新增 `SaveFingerprint` / `LoadFingerprint`，存 `last-scan.fp.json`（`internal/scanner/cache.go` 抽 `writeJSONAtomic` 公共原子写，`SaveCache` 改用它）；`cache --clear` 同时清除指纹文件。**不改 `last-scan.json` 格式**——`LoadCache` 与全部消费方零改动，规避缓存迁移风险。
- **`ScanIfUnchanged(opts)`（新）**：`internal/scanner/scanner.go` 的 `Scan` 主体抽为内部 `scan(opts, skipIfUnchanged)`；`ScanIfUnchanged` 先读缓存 + 指纹，无变化直接返回缓存（不扫描、不写历史），否则走全量扫描；指纹文件缺失（首次运行）保守走全量。缓存写入时同写指纹（基于**未过滤**的 `cached` 结果）。
- **GUI 接入**：`gui/service.go` 新增绑定 `ScanIfChanged()`（指纹命中返回缓存结果、跳过历史与重探测；仅真实扫描追加历史/探测）；`main.ts` 启动路径改用 `ScanIfChanged()`，手动"重新扫描"按钮仍走 `Scan()`（强制全量）。wails 绑定重新生成。
- **CLI 接入**：`internal/cli/scan.go` 非 `--refresh` 路径改用 `ScanIfUnchanged`（`--no-cache` 时跳过捷径）；数据变化后 `scan` 自动重扫（从"可能陈旧"变为"按需新鲜"）。
- **已知盲区（文档化）**：文件内容原地修改不改变父目录 mtime → 漏检，但占用大小通常不变；`--refresh` / GUI"重新扫描"永远强制全量。

## Capabilities

### New Capabilities
- `scan-change-detection`：扫描结果变更检测（mtime 指纹），驱动 GUI 启动与 CLI 非刷新路径跳过全量扫描

### Modified Capabilities
<!-- 无能力修改：指纹文件独立、缓存格式不变、扫描 JSON 契约不变 -->

## Impact

- **代码**：`internal/scanner/`（fingerprint.go 新增、cache.go 抽公共原子写 + 指纹存取、scanner.go 抽 scan 内部函数）、`internal/cli/scan.go`、`gui/service.go`、`frontend/src/main.ts`、`frontend/wailsjs/`（绑定再生成）
- **测试**：`fingerprint_test.go`（采集/比较语义）、`cache_test.go`（指纹往返 + ClearCache 联动）、`unchanged_test.go`（缓存命中/指纹缺失回退两个集成用例，隔离 XDG_CACHE_HOME + LOCALAPPDATA）
- **行为**：GUI 启动在数据未变化时秒开（无全量 IO、无"扫描中…"闪烁）；安装/删除工具后启动自动全量；`scan` 无 `--refresh` 在数据变化后自动重扫；`cache --clear` 连指纹一起清
- **不引入**：不改 `last-scan.json` 格式与扫描 JSON 契约；不引入新依赖；不改变手动"重新扫描"/`--refresh` 的强制全量语义
