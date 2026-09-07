# Tasks: harden-concurrency-and-data-layer

## 1. GUI 并发边界

- [x] 1.1 `gui/service.go`：`ScannerService` 增加 `scanning` 标志，`Scan`/`ScanIfChanged` 共用 single-flight 闸门（丢弃语义，ADR-0001），验证：并发触发测试 + `-race` 下 `go test ./gui/` 通过
- [x] 1.2 `gui/service.go`：`probeAll` 改为锁内快照拷贝探测目标，合并时锁内二次校验 `s.last` 未变，验证：`go test -race ./gui/` 通过且 probe:done 契约不变

## 2. 数据层自愈

- [x] 2.1 `internal/history/history.go`：`open` 拆出 `openDB`，corruption 错误时备份库文件与 -wal/-shm 侧车为 `.corrupt-<时间戳>` 后重建，验证：`TestOpenRecoversFromCorruptDB`（备份留档 + 自愈后可写入查询）通过
- [x] 2.2 `internal/history/history.go`：`isCorruptErr` 按错误消息（malformed / not a database / corrupt）宽泛匹配，验证：单测覆盖非 SQLite 文件路径

## 3. 代跑命令健壮性

- [x] 3.1 `internal/upgrade/command.go` + `internal/uninstall/uninstall.go`：npm 包名为空时降级为不可代跑提示，代跑命令永不携带空参数，验证：`TestOfficialCommandNpmUnknownPkg`（uninstall）与 upgrade 侧空参数契约断言通过
- [x] 3.2 `gui/service.go`：升级/卸载代跑 PATH 解析失败时写入明确错误（`up.resolveFailed` / `un.resolveFailed`）并返回，不静默裸名执行，验证：三语 locale 新键齐全且 i18n parity 测试通过
- [x] 3.3 `internal/uninstall/uninstall.go`：`IsBlocked` 循环剥离多层可执行扩展名，验证：`TestIsBlocked` 新增 `python.exe.exe` / `npm.cmd.exe` 用例通过

## 4. 前端转义

- [x] 4.1 `frontend/src/flows.ts`：`upd.versions`、`up.guiFound` 等动态值插值前经 `esc()`（保留文案 `<b>` 标记），验证：`npx tsc --noEmit` + `npm test` 通过

## 5. 流程固化与文档

- [x] 5.1 `scripts/test-all.sh` 加入 `go test -race ./gui/`；docs/release-process.md 验证清单补充 race 要求，验证：`bash -n` 语法检查通过、清单条目可见
- [x] 5.2 CONTEXT.md（领域术语表）与 docs/adr/0001（single-flight 决策）随变更入库，验证：文件存在于仓库根与 docs/adr/
- [x] 5.3 全量回归：`go vet ./...`、`gofmt -l .` 为空、`go test -race ./...` 全绿、前端 `tsc --noEmit` + vitest 通过，验证：全部命令退出码 0
