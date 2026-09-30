## 1. 合计口径

- [x] 1.1 `internal/scanner/attribute.go` finalize：cleanSum 改为极大路径求和（复用 sumMaximal）
- [x] 1.2 scanner 测试：finalize 后全部工具 `cleanableBytes ≤ footprintBytes`；marscode 可处置 ≈ .marscode 单份（不再 7.9G）

## 2. 清理去重

- [x] 2.1 `internal/cleaner/clean.go`：选中集合中严格位于另一选中路径之内的条目跳过，原因 `cln.subsumed`
- [x] 2.2 i18n `cln.subsumed` 三语；cleaner 测试：父子同选 → Deleted 仅父项、子项入 Skipped

## 3. GUI 同口径

- [x] 3.1 `lib/grouping.ts` 新增 pathInside + 测试；`selectedItems` 过滤被父项包含的选择，确认弹窗合计同口径

## 4. 验证

- [x] 4.1 go test 全绿；实机重扫：AI 段 opencode/marscode/kimi 的 可处置 ≤ 总占用、安装占用列恢复非零
