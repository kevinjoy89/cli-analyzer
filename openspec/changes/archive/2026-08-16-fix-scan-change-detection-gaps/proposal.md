# Proposal: fix-scan-change-detection-gaps

## Why

3c3ff54..HEAD 审查（变更扫描指纹 + 前端模块化）发现指纹特性的三个功能缺口：

1. **新装工具失明**：指纹只覆盖上次扫描的测量路径（二进制 real、dataDirs、cleanables、孤儿路径）。新安装的工具产生的是"全新路径"，不在指纹内，且通常不改变任何既有测量路径 → GUI 启动 / CLI 非 `--refresh` 返回陈旧缓存，新工具不显示。旧版 GUI 每次启动都全量重扫，新工具下次启动必现——这是指纹特性引入的回归，且未文档化。
2. **版本列空**：`probeAll` 被 `realScan` 门控，而 `last-scan.json` 从不包含探测版本（`SaveCache` 在 `scan()` 内、探测之后只改内存）→ 缓存命中启动时版本列全空，直到手动重扫（旧版每次启动都探测）。
3. **历史不一致**：CLI 非 `--refresh` 的自动重扫不记录历史，而 `--refresh` 与 GUI 真实扫描都记录；GUI 用 `prev.ScannedAt` 启发式判断真实扫描，在 CLI 先扫描的情况下会对同一扫描重复记录。

## What Changes

- **指纹纳入 PATH 发现目录**：`ComputeFingerprint` 追加 `platform.PathDirs(true)` 中存在的目录的 stat 条目（仅 stat，不递归）。新二进制/新工具出现在 PATH 目录时目录 mtime 变化 → 指纹不一致 → 自动全量。指纹格式增量扩展，旧指纹（无 PATH 条目）与当前计算不一致 → 保守走一次全量，安全方向。
- **指纹精度统一为纳秒**：`MTime` 由 `ModTime().Unix()`（秒）改为 `UnixNano()`，与 probe 缓存一致；同一秒内同大小的替换也可检测。
- **`ScanIfUnchanged` 返回是否真实扫描**（`(*ScanResult, bool, error)`）：GUI 用该标志替代 `ScannedAt` 启发式（消除 CLI 先扫描导致的重复历史记录）；CLI 非 `--refresh` 真实扫描（含 `--no-cache`）时记录历史，与 `--refresh` 一致。
- **GUI 启动总是版本探测**：`ScanIfChanged` 中 `probeAll` 不再被 `realScan` 门控（probe 缓存按 path+size+mtime 键控、秒回），仅历史记录受门控 → 缓存命中启动版本列也有值。
- **GUI 启动扫描显示 busy 状态**：`main.ts` 启动调 `ScanIfChanged` 前 `setScanning(true)`，与手动重扫一致（按钮禁用/转圈，防止并发扫描）。
- **代码卫生**：移除死字段 `Options.Refresh` 及 `scan()` 中的对应分支；更新 `service.go` Startup 过期注释；`flows.ts` 中段 import 合并到文件顶部；`menu.ts` 死代码守卫改为防御式 `getElementById`。

## Capabilities

### New Capabilities

<!-- 无新能力：全部为既有能力的需求变更 -->

### Modified Capabilities

- `scan-change-detection`: 指纹采集范围扩展（PATH 发现目录）、mtime 精度纳秒化、变更判定返回真实扫描标志、盲区收敛（新装工具可检测）、GUI 启动探测与历史记录语义。

## Impact

- **Go**: `internal/scanner/fingerprint.go`（PATH 条目 + UnixNano）、`scanner.go`（`ScanIfUnchanged` 签名、移除 `opts.Refresh` 分支、扫描结果的历史记录语义不变）、`types.go`（删 `Options.Refresh`）；`gui/service.go`（`ScanIfChanged` 改用 scanned 标志 + 总是探测 + 注释）；`internal/cli/scan.go`（非 `--refresh` 真实扫描记录历史）。
- **前端**: `frontend/src/main.ts`（启动 busy 状态）、`flows.ts`（import 整理）、`menu.ts`（防御式守卫）。wails 绑定签名不变，无需重新生成。
- **测试**: `internal/scanner/fingerprint_test.go`（PATH 条目断言）、`unchanged_test.go`（新签名 + PATH 场景）、新增 PATH 变更检测测试；`cache_test.go` 不受影响。
- **文档**: `README.zh-CN.md` 已知限制更新（新装工具盲区收敛说明）。
- **不破坏**: 指纹文件格式为增量扩展（旧文件 → 一次保守全量，自愈）；`last-scan.json` 格式不变；Wails 绑定契约不变。
