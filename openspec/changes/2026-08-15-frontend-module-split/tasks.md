# 前端 main.ts 拆模块

> 变更目录：2026-08-15-frontend-module-split；无能力变更（纯重构）

> **执行环境（必读）**：本仓库在 DSH 沙箱下工作，所有 `pwsh` 命令（npm / npx / tsc / git 等）当前被沙箱拦截（`SetNamedSecurityInfoW failed: grantWrite`）。实施时每一步验证/提交命令经 `pwsh` 执行会弹出授权请求——批准 `workspace-write` 即可放行；授权为会话级，批准后后续命令通常不再重复弹窗。本变更无文件删除操作。

## 1. dom.ts + state.ts

- [ ] 1.1 新建 `frontend/src/dom.ts`：剪切 main.ts 的 `el` / `esc` / `showToast`（约 106-118、562-564 行）
- [ ] 1.2 新建 `frontend/src/state.ts`：剪切全部顶层状态与状态类型（result/probing/selected/appVersion/filterText/selectedCleanIds/expandedCleanIds/sortKey/sortDir/themeMode/isMac/panelView/orphanSel/trashItems/updateState/lastUpdateResult/downloadPoll/lastShownPct/uninstallPoll/reminderTools + interface CleanReport/TrashItem/…），采用命名空间写法（方案 A）；main.ts 内重复 interface 迁入 state.ts 或改 import lib/types（D4）
- [ ] 1.3 main.ts 改 `import * as dom from './dom'; import * as state from './state';`
- [ ] 1.4 验证：`npx tsc --noEmit`（0 error）+ `npm test` + `npm run build`
- [ ] 1.5 提交：`refactor(frontend): extract dom primitives and global state into modules`

## 2. render.ts

- [ ] 2.1 新建 `frontend/src/render.ts`：剪切渲染函数（renderSummary/renderToolList/renderPanelTabs/filteredOrphans/renderOrphanView/renderDetail/subRows/subIdsOf/selectedItems/renderBellPanel/renderTrendChart/renderGrowers + COLUMNS/KIND_TONE/SUB_CAP/TRASH_ICON/RESTORE_ICON）
- [ ] 2.2 `renderDetail` 的卸载回调改为 `uninstallHandler?.(tool.name)`（D2 回调注册）；`export let uninstallHandler: ((name: string) => void) | null = null`
- [ ] 2.3 main.ts 删除已移动函数，`import * as render from './render'`
- [ ] 2.4 验证：`npx tsc --noEmit` + `npm test` + `npm run build`
- [ ] 2.5 提交：`refactor(frontend): extract render functions into render module`

## 3. flows.ts + menu.ts + main.ts 瘦身

- [ ] 3.1 新建 `frontend/src/flows.ts`：剪切业务流程（trashPaths/confirmDialog/showOrphanConfirm/showConfirmModal/refreshTrashInfo/openTrashPanel/refreshTrashList/更新流程全部/卸载流程全部/openPrefs/refreshReminder/openTrends/refreshTrends/openAbout/rescan/setScanning）
- [ ] 3.2 新建 `frontend/src/menu.ts`：剪切 closeMenus/initMenuBar（依赖 flows 的 openPrefs/manualCheck）
- [ ] 3.3 main.ts 瘦身：保留 import 汇总 + systemDark/applyTheme + resolveLocale/initI18n/applyI18n + init() 接线；init() 补 `render.uninstallHandler = (n) => flows.startUninstall(n);`
- [ ] 3.4 验证：`npx tsc --noEmit`（重点：无循环依赖报错）+ `npm test` + `npm run build`
- [ ] 3.5 真机冒烟（release-process.md 第 7 步清单）：扫描/过滤/排序/详情勾选清理；回收站打开-恢复-清空（确认弹窗样式与层级）；更新面板手动检查-下载取消-失败面板保留；卸载起始信息-代跑-残留列表；首选项语言即时切换；Windows 菜单条下拉；**重点验证事件不重复绑定**（弹窗开-关-再开，操作不叠加）
- [ ] 3.6 提交：`refactor(frontend): extract flows and menu modules; slim main.ts to wiring`

## 4. 收尾验证

- [ ] 4.1 全量：`gofmt -l .` + `go vet ./...` + `go test ./... -cover` + `(cd frontend && npm test && npm run build)` + 三平台 `GOOS=windows|linux|darwin go build ./...`
- [ ] 4.2 GUI 冒烟（真机，必须）：按 release-process.md 清单逐项；若有修复，追加提交 `fix(frontend): smoke-test fixes after module split`
