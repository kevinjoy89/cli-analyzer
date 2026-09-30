## 1. 编目与契约

- [x] 1.1 `DataDirRule` 新增 Form 字段；桌面端目录标注（opencode/codex/dsh/kimi/agy/gemini/zcode/workbuddy/u1s1/code/trae-cn/marscode 的 AppSupport 目录）
- [x] 1.2 `DataDir` JSON 契约透传 form；scanner 对 .app 结尾的 install 归因自动标 desktop
- [x] 1.3 rules 测试：桌面端目录标注正确

## 2. 展示

- [x] 2.1 `lib/grouping.ts`：formSubtotals（极大路径去重后按 form 合计）+ vitest
- [x] 2.2 详情页「形态占用」块：桌面端 / CLI 两行小计；types.ts 补 form
- [x] 2.3 i18n ui.form.desktop / ui.form.cli 三语

## 3. 验证

- [x] 3.1 go test + 前端 vitest 全绿；实机核对 opencode/dsh/kimi/zcode 的形态小计（归因目录口径：独立二进制与仅清理项不计入）
