# 扫描增量优化（mtime 指纹变更检测）

> 变更目录：2026-08-15-scan-change-detection；能力：scan-change-detection（新）

> **执行环境（必读）**：本仓库在 DSH 沙箱下工作，所有 `pwsh` 命令（go test / gofmt / go vet / git / npx / npm 等）当前被沙箱拦截（`SetNamedSecurityInfoW failed: grantWrite`）。实施时每一步验证/提交命令经 `pwsh` 执行会弹出授权请求——批准 `workspace-write`（个别命令需 `danger-full-access`）即可放行；授权为会话级，批准后后续命令通常不再重复弹窗。本变更无文件删除操作。

## 1. 指纹核心（TDD）

- [ ] 1.1 写失败测试 `internal/scanner/fingerprint_test.go`：TestComputeFingerprintAndEqual（临时目录构造结果；等值/顺序无关/条目缺失/mtime 变化/size 变化/真实 stat 一致性）
- [ ] 1.2 运行确认失败（undefined: ComputeFingerprint）
- [ ] 1.3 实现 `internal/scanner/fingerprint.go`：FingerprintEntry / statEntry / measurePaths / ComputeFingerprint（排序）/ FingerprintsEqual
- [ ] 1.4 运行确认通过 + gofmt 干净
- [ ] 1.5 提交：`feat(scanner): add mtime-based fingerprint for change detection`

## 2. 指纹持久化（TDD）

- [ ] 2.1 追加失败测试 `cache_test.go`：TestFingerprintRoundTrip（隔离 XDG_CACHE_HOME + LOCALAPPDATA；Save/Load 往返；ClearCache 联动清指纹）
- [ ] 2.2 运行确认失败（undefined: SaveFingerprint）
- [ ] 2.3 实现 `cache.go`：抽 `writeJSONAtomic`（SaveCache 改用它）；`SaveFingerprint`/`LoadFingerprint`（last-scan.fp.json）；`ClearCache` 连带清指纹
- [ ] 2.4 运行确认通过（含既有 cache 测试）+ gofmt 干净
- [ ] 2.5 提交：`feat(scanner): persist fingerprint file, clear it with cache --clear`

## 3. ScanIfUnchanged（TDD）

- [ ] 3.1 写失败测试 `internal/scanner/unchanged_test.go`：TestScanIfUnchangedCacheHit（预置缓存+指纹，返回缓存且 ScannedAt 不变）；TestScanIfUnchangedNoFingerprintFallsBack（无指纹走全量，ScannedAt 变化）
- [ ] 3.2 运行确认失败（undefined: ScanIfUnchanged）
- [ ] 3.3 实现 `scanner.go`：`Scan` 主体抽为内部 `scan(opts, skipIfUnchanged)`；新增 `ScanIfUnchanged`；缓存写入处追加 `SaveFingerprint(ComputeFingerprint(cached))`（cached 为未过滤结果）
- [ ] 3.4 运行确认通过（含既有 scan/classify/attribute 测试）+ gofmt 干净
- [ ] 3.5 提交：`feat(scanner): ScanIfUnchanged skips full rescan when fingerprint is unchanged`

## 4. GUI 接入

- [ ] 4.1 `gui/service.go` 新增 `ScanIfChanged()`：内部 `scanner.ScanIfUnchanged`；仅真实扫描（prev==nil 或 ScannedAt 变化）才 history.Record + probeAll；事件契约与 Scan 一致
- [ ] 4.2 重新生成 wails 绑定（`wails generate module`）；确认 `frontend/wailsjs/go/gui/ScannerService.js` 与 `.d.ts` 含 ScanIfChanged；无法跑 wails 时手工补同构条目
- [ ] 4.3 `frontend/src/main.ts`：import 加 ScanIfChanged；`init()` 末尾 `rescan()` → `ScanIfChanged()`；rescanBtn onclick 不变（强制全量）
- [ ] 4.4 验证：`npx tsc --noEmit` + `npm test` + `go vet ./gui/` + `go build ./...`
- [ ] 4.5 真机冒烟：首启全量→再启秒开（无全量 IO/无扫描闪烁）；手动重扫仍全量；安装/删除工具后启动自动全量
- [ ] 4.6 提交：`feat(gui): startup scan uses change detection, manual rescan stays full`

## 5. CLI 接入

- [ ] 5.1 `internal/cli/scan.go`：`--refresh` 走 Scan；非 `--refresh` 走 ScanIfUnchanged（`--no-cache` 时直接 Scan）；历史仅真实扫描追加
- [ ] 5.2 验证：`go build ./...` + `go test ./internal/cli/` + 手动三步（首扫→秒回→touch 触发重扫）
- [ ] 5.3 提交：`feat(cli): scan without --refresh auto-rescans when fingerprint changed`

## 6. 文档

- [ ] 6.1 README.md / README.zh-CN.md：Known limitations 追加指纹盲区说明；`scan` 行注释更新（auto-rescans when files changed）
- [ ] 6.2 提交：`docs: document fingerprint-based skip of unchanged scans`
