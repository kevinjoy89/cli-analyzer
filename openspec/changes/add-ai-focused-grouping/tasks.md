## 1. 分组纯逻辑

- [x] 1.1 新增 `frontend/src/lib/grouping.ts`：matchesFilter（名称/别名）、groupTools（AI 段 = category 非空，其他段 = 其余，含两段合计）
- [x] 1.2 `grouping.test.ts`：分段正确性、过滤联动、合计、空段

## 2. 视图与状态

- [x] 2.1 `state.ts`：othersExpanded（默认 false）+ setter
- [x] 2.2 `render.ts` renderToolList：AI 段头（始终展开）→ 列头 → AI 行；其他段头（折叠/展开切换）→ 展开时列头 + 行；排序键段内生效
- [x] 2.3 搜索联动：filterText 非空时两段过滤，其他段有命中自动展开（不持久化自动展开）
- [x] 2.4 `style.css`：段头样式（置顶条、可折叠箭头、悬停）

## 3. i18n 与收尾

- [x] 3.1 ui.group.ai / ui.group.others（{n} · {size} 占位）三语补齐
- [x] 3.2 前端 vitest + vite build + go test 全绿；实机核对：AI 段 16 行置顶、其他段默认折叠、搜索 "npm" 自动展开
