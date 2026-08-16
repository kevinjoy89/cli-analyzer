# 前端 main.ts 拆模块

## Why

`frontend/src/main.ts` 已膨胀到 1448 行，承载全部 UI 逻辑：状态、渲染、i18n、更新、卸载、回收站、首选项、趋势、菜单，全部用字符串 innerHTML 拼接 + 手动事件绑定。痛点：

- **可维护性**：新增功能要在一个文件里同时改状态、渲染、事件与流程；函数间引用靠"恰好在同一文件"维持，无法独立测试
- **事件堆积风险**：`refreshTrashList` 等每次整树重建按钮并重新绑定 onclick，长会话下旧闭包累积
- **双菜单漂移**：macOS 原生菜单（main.go）与 Windows HTML 菜单（index.html）各一份，i18n key 双份手写
- **类型契约漂移**：TS 接口手工镜像 Go JSON 契约（`lib/types.ts` + main.ts 内 interface），Go 侧加字段前端静默失效

本次是**纯结构重构**：按依赖方向拆成 5 个模块，不改变任何行为与样式。

## What Changes

- **`frontend/src/dom.ts`**（新）：DOM 原语 `el` / `esc` / `showToast`
- **`frontend/src/state.ts`**（新）：全部顶层状态（result / selected / filterText / selectedCleanIds / sortKey / themeMode / panelView / orphanSel / trashItems / updateState / uninstallPoll / reminderTools …）+ 状态类型
- **`frontend/src/render.ts`**（新）：渲染函数（renderSummary / renderToolList / renderOrphanView / renderDetail / subRows / renderBellPanel / renderTrendChart / renderGrowers…）+ COLUMNS / KIND_TONE / SUB_CAP / 图标常量；`renderDetail` 调 `startUninstall` 的循环依赖用**回调注册**解法（`export let uninstallHandler`，main.ts init 里赋值）
- **`frontend/src/flows.ts`**（新）：业务流程（更新下载/卸载/回收站/首选项/趋势/确认弹窗/trashPaths/rescan…）
- **`frontend/src/menu.ts`**（新）：Windows 菜单 `closeMenus` / `initMenuBar`
- **`frontend/src/main.ts`**（瘦身）：import 汇总 + 主题（systemDark/applyTheme）+ i18n 初始化（resolveLocale/initI18n/applyI18n）+ `init()` 事件接线

依赖方向（无环）：`dom.ts` ← `state.ts` ← `render.ts` ← `flows.ts` ← `menu.ts` / `main.ts`。

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
<!-- 无能力变更：纯前端结构重构 -->

## Impact

- **代码**：`frontend/src/`（新增 dom.ts / state.ts / render.ts / flows.ts / menu.ts；main.ts 瘦身）；`frontend/wailsjs/` 不动
- **测试**：既有 `lib/*.test.ts` 与 vitest 不受影响；`lib/types.ts` 保留为契约单一来源，main.ts 内重复 interface 迁入 state.ts 或改 import lib/types
- **行为**：零变化（纯移动）；验证 = `tsc --noEmit` 0 error + `npm test` 全过 + `npm run build` 通过 + 真机 GUI 冒烟（release-process.md 第 7 步清单）
- **不引入**：不引入框架/状态库（保持零依赖 vanilla TS）；不改样式与 HTML 结构；不合并/删除现有 lib/ 模块
