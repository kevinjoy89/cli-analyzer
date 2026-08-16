# Tasks: fix-scan-change-detection-gaps

## 1. 指纹：PATH 发现目录条目 + 纳秒精度

- [x] 1.1 `internal/scanner/fingerprint.go`：`statEntry` 的 `MTime` 改为 `st.ModTime().UnixNano()`
- [x] 1.2 `internal/scanner/fingerprint.go`：`ComputeFingerprint` 追加 `platform.PathDirs(true)` 中存在的目录的 stat 条目（与结果条目同一列表，排序后序列化）
- [x] 1.3 `internal/scanner/fingerprint_test.go`：PATH 条目断言改为动态计数（结果条目 + 当前存在 PATH 目录数）；mtime 断言改 UnixNano；新增 `TestFingerprintDetectsPathDirChange`

## 2. ScanIfUnchanged 返回真实扫描标志

- [x] 2.1 `internal/scanner/scanner.go`：`ScanIfUnchanged` 签名改为 `(*ScanResult, bool, error)`；缓存命中返回 `(cached, false, nil)`，全量扫描路径返回 `(res, true, err)`；`Scan` 包装保持不变
- [x] 2.2 `internal/scanner/scanner.go`：`scan()` 移除 `!opts.Refresh` 分支；`internal/scanner/types.go` 删除 `Options.Refresh` 字段
- [x] 2.3 `internal/scanner/unchanged_test.go`：适配新签名（两处调用）；新增 `TestScanIfUnchangedDetectsNewPathBinary`

## 3. CLI 历史记录一致

- [x] 3.1 `internal/cli/scan.go`：非 `--refresh` 路径接收 scanned 标志（`--no-cache` 视为真实扫描），`scanned && len(filters)==0` 时 `history.Record`

## 4. GUI：探测解耦 + 历史标志 + 注释

- [x] 4.1 `gui/service.go` `ScanIfChanged`：改用 scanned 标志替代 `prev.ScannedAt` 启发式；`history.Record` 仅真实扫描；`probeAll` 无条件执行（缓存命中启动也补齐版本列）
- [x] 4.2 `gui/service.go`：更新 Startup 过期注释（描述 ScanIfChanged 流程）

## 5. 前端：启动 busy 状态 + 代码卫生

- [x] 5.1 `frontend/src/main.ts`：启动 `ScanIfChanged()` 前 `flows.setScanning(true, t('ui.scanning'))`
- [x] 5.2 `frontend/src/flows.ts`：中段 `import {KIND_TONE, RESTORE_ICON, TRASH_ICON} from './render'` 合并进顶部 import
- [x] 5.3 `frontend/src/menu.ts`：`el('menuBar')` 改为 `document.getElementById('menuBar')` + 空值守卫（消除死守卫），移除未用的 `el` import

## 6. 测试与文档

- [x] 6.1 `internal/scanner/unchanged_test.go` / `fingerprint_test.go`：PATH 新增二进制场景（`t.Setenv("PATH", ...)` 指向临时目录），含粗粒度文件系统（FAT 2s）Chtimes 兜底
- [x] 6.2 `README.md` / `README.zh-CN.md` 已知限制：更新指纹盲区描述（新装工具已由 PATH 目录条目检测；仅保留原地修改盲区）
- [x] 6.3 前端验证：`npx tsc --noEmit` 通过、`npm test`（vitest 19 用例）通过
- [x] 6.4 Go 验证：`gofmt -l .` 为空、`go vet ./...` 通过、`go test ./internal/... ./gui/...` 全绿（本机 .go-toolchain go1.26.6）
