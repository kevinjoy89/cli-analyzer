## 1. cleaner 拆除硬门槛

- [x] 1.1 `internal/cleaner/cleaner.go`：删除整项与子项两处 `Tier != SAFE → Skipped` 分支；包注释更新为"用户决策 + 完整性守卫"
- [x] 1.2 guard/guardSub/forbiddenRoots/windowsForbiddenRoots 原样保留（防灾难性错误）
- [x] 1.3 cleaner_test：`TestCleanRejectsUserTier` → `TestCleanDeletesUserTier`（USER 项经回收站删除）；`TestCleanSubUserParentRejected` → `TestCleanSubUserParentDeleted`；`TestCleanMessagesLocalized` 改为守护 guard 消息本地化

## 2. scanner 语义降级

- [x] 2.1 `attribute.go`：数据目录（安装根除外）全部生成可处置项，按物理路径去重；`kindDesc` 辅助函数
- [x] 2.2 全局路径去重 pass（backups 之后、sizing 之前）
- [x] 2.3 子项 breakdown 对所有可处置项计算（去掉 TierSafe 过滤）
- [x] 2.4 `finalize`：`cleanableBytes` = 全部可处置项；`userBytes = footprint − cleanable`
- [x] 2.5 `types.go`：Tier/Cleanable/Tool 注释更新（标签语义）
- [x] 2.6 测试：`TestFinalize` totals 断言更新；新增 `TestAttributeAllDataDirsActionable`（cache+config 可处置、安装根不可处置）

## 3. uninstall 残留双选项

- [x] 3.1 `residue.go`：新增 `RemoveResidues`（永久删除，不可恢复）；包注释更新
- [x] 3.2 `gui/service.go`：新增 `UninstallDeleteResidues` 绑定（与 trash 版相同的路径白名单）
- [x] 3.3 `frontend/wailsjs/go/gui/ScannerService.js` + `.d.ts`：同步生成绑定
- [x] 3.4 `frontend/src/main.ts`：残留面板新增「永久删除」按钮（`confirmDialog` 强确认 → `UninstallDeleteResidues` → toast + rescan）
- [x] 3.5 `internal/cli/uninstall.go`：新增 `--permanent` 标志与确认文案分支；`un.trashedN`/`un.deletedPermanentN` 插值输出
- [x] 3.6 测试：`TestRemoveResiduesPermanently`（删除生效且不经过回收站）

## 4. CLI 批处理白名单

- [x] 4.1 `clean.go`：`--all` 默认 {cache, old-version, backup, download}；`--include-data` 追加数据类；未命中输出 `cli.cleanAllEmpty` 提示
- [x] 4.2 `output.go`：`printCleanables` 移除 Tier 过滤（列出全部可处置项）
- [x] 4.3 测试：`TestRunCleanAllKindsWhitelist`（默认 1 项 / `--include-data` 2 项）、`TestRunCleanAllDataOnlyHintsIncludeData`

## 5. 前端详情面板

- [x] 5.1 `main.ts`：可处置项 = 全部 cleanables（移除 `tier === 'safe'` 过滤）；数据目录区只展示安装根（`ui.sectionInstallRoot`）
- [x] 5.2 `selectedItems` 移除 Tier 过滤
- [x] 5.3 前端 `npm test`（19 用例）与 `tsc --noEmit` 通过

## 6. i18n 与文档

- [x] 6.1 三语言 locale：键集一致（345 键，`i18n.KeysEqual` 语义由 node 脚本验证）；移除 `cln.userSkipped`、`ui.sectionDataDirs`；新增 `--include-data`/永久删除相关文案；`ui.safeOnly`/`cli.usage`/表头/趋势文案改为"可处置/安装占用"口径
- [x] 6.2 `README.md` / `README.zh-CN.md`：安全模型 → 处置模型；用法与示例表头更新
- [x] 6.3 本 change 的 proposal/design/tasks

## 7. 全量验证

- [x] 7.1 Go 侧：gofmt 合规（全部修改文件）+ `go vet ./...` 通过 + `go test ./...` 全绿（唯一例外：预存在的 `TestCollectBackupsSkipsSymlinkTarget` 在本机 Windows 无开发者模式权限下无法创建符号链接而失败，属环境限制、与本次改动无关；GitHub CI 的 windows runner 默认开启符号链接权限，不受影响）。验证工具链临时解压于 `.go-toolchain/`（已加入 .gitignore）
- [x] 7.2 前端：`npm test` 19/19 通过；`tsc --noEmit` 通过
- [x] 7.3 i18n 键集三语一致（345 键，node 脚本比对）

## 8. 同步与归档

- [x] 8.1 能力规格同步（apply）：`tool-attribution`（新增"归因目录均为可处置项，Tier 为信息标签"需求）、`trash-recycle-bin`（SAFE 措辞 → 处置；永久删除为显式选择）、`tool-uninstall`（残留处置默认回收站 + 永久删除显式选择 + 白名单约束）、`usage-trends`（cleanable → 可处置口径）
- [x] 8.2 归档：本 change 移入 `openspec/changes/archive/2026-08-15-remove-safety-gate/`
