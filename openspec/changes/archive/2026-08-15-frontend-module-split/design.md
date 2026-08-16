# Design: 前端 main.ts 拆模块

## Context

main.ts（1448 行）是单体：状态、渲染、业务流、事件接线全部混在一个模块里。函数按"被谁调用"而非"属于哪个域"组织，新增功能时难以定位改动点，且整树 innerHTML 重建 + 手动事件绑定在长会话下有闭包累积风险。lib/ 已有纯逻辑模块化先例（clean/format/i18n/labels/trends/types），本次把 UI 层同样模块化。

## Goals / Non-Goals

**Goals**
- 按依赖方向拆出 5 个模块（dom/state/render/flows/menu），无循环依赖
- 纯移动：行为、样式、HTML 结构、i18n key 全部不变
- main.ts 瘦身为"入口 + 初始化 + 事件接线"
- 为未来可测性铺路（渲染与流程分离后，纯逻辑部分可进 vitest）

**Non-Goals**
- 不引入框架（React/Vue）或状态库——保持零依赖 vanilla TS
- 不重构渲染方式（仍字符串 innerHTML；拆模块不解决渲染模式本身）
- 不改 Go 侧 / wailsjs 绑定 / HTML / CSS

## Decisions

### D1: 模块划分与依赖方向

```
dom.ts (el/esc/showToast)
  ← state.ts (全部顶层状态)
    ← render.ts (渲染函数 + 常量)
      ← flows.ts (业务流程：更新/卸载/回收站/首选项/趋势/确认弹窗/rescan)
        ← menu.ts (Windows 菜单)
main.ts (import 汇总 + theme + i18n init + init() 接线) → 依赖以上全部
```

依赖单向、无环：render 只依赖 dom/state/lib；flows 依赖 render；menu 依赖 flows；main 依赖全部。

### D2: 循环依赖解法（renderDetail → startUninstall）

`renderDetail` 的"卸载"按钮回调调 `startUninstall`（flows）。让 render 依赖 flows 会成环（flows 也调 render 的 renderDetail/renderToolList）。解法：**回调注册**——

```ts
// render.ts
export let uninstallHandler: ((name: string) => void) | null = null;
// renderDetail 内: uninstallHandler?.(tool.name)

// main.ts init(): render.uninstallHandler = (n) => flows.startUninstall(n);
```

main.ts 作为组合根负责注入，render 保持对 flows 零依赖。

### D3: 状态模块两种写法（二选一，保持一致）

- 方案 A（推荐）：`state.ts` 导出 `let`，消费方 `import * as state` 后 `state.selected = x`（ESM 允许通过命名空间写 let 导出），避免为每个变量写 setter
- 方案 B：为每个状态写 setter（改动量大，仅当 A 遇到 tsc 限制时使用）

### D4: 类型契约单一来源

main.ts 内重复声明的前端 interface（CleanReport / TrashItem / UpdateResult / UninstallStatus…）迁移到 `state.ts`（或从 `lib/types.ts` 复用/扩展）；`lib/types.ts` 保持为与 Go JSON 契约对应的类型唯一权威，后续新增字段只改一处。

### D5: 分步迁移与守门

按 dom → state → render → flows/menu → main 的顺序，每步：剪切函数 → 新建模块 + import → `npx tsc --noEmit`（0 error）→ `npm test` → `npm run build` → 提交。每步都是可回退的独立提交。

### D6: 验证（冒烟必须）

拆模块是纯移动，单测/构建全绿不代表 GUI 正常——按 release-process.md 第 7 步清单真机冒烟：扫描/过滤/排序/详情勾选清理、回收站（恢复/清空/确认弹窗）、更新面板（手动检查/下载取消/失败面板保留）、卸载流程（起始信息/代跑/残留列表）、首选项语言切换、Windows 菜单条下拉。重点检查**事件重复绑定**（打开-关闭-再打开弹窗，操作不叠加）。

## Risks / Trade-offs

- [移动过程漏掉一个引用 → tsc 报错] → 每步 tsc 守门，0 error 才提交
- [循环依赖漏网] → 依赖方向固定 + tsc 的 import cycle 检测（vite/tsc 会报 "Cannot access before initialization" 类错误）
- [状态写法不统一] → D3 二选一后全程一致
- [冒烟覆盖不全导致回归] → 按 release-process.md 清单逐项执行；事件重复绑定重点验证

## Migration Plan

无数据/持久化变化。分步提交（5 个 commit），每步可单独回退；完成后再做一次全量 `scripts/release/check.sh`（或等效命令）+ 真机冒烟。
