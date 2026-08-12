## 1. 官方卸载命令表

- [ ] 1.1 新建 `internal/uninstall/` 包；实现来源 → 官方命令映射（brew/npm/pipx/cargo 可代跑；go/pyenv/versioned/other 仅提示），命令参数（formula/pkg/crate）从扫描归类信息解析
- [ ] 1.2 系统关键工具黑名单（python/node/git/docker/go/brew/cli-analyzer 自身等）+ 单测（命中拒绝、未命中放行）
- [ ] 1.3 命令表单元测试：各来源映射正确、参数解析、解析不到时的模板回退

## 2. 残留检测（双源）

- [ ] 2.1 规则表目录源：按当前平台 roots 解析 dd() 规则为绝对路径
- [ ] 2.2 扫描快照源：读取卸载前归因到该工具的 dataDirs（scanner 暴露所需信息）
- [ ] 2.3 合并去重 + 存在性过滤 → 残留项 {path, tier, kind, bytes}；USER 级标注"含登录凭证"
- [ ] 2.4 残留检测单元测试：双源合并、存在性、级别标注、无残留场景

## 3. 残留清理（回收站硬约束）

- [ ] 3.1 复用 `trash.Trash()` 移入残留项；无 `--permanent` 变体、`--yes` 不豁免（单测断言 USER 项不可能被直删）
- [ ] 3.2 移入失败（跨文件系统/进程占用）→ 返回失败项与原因，不静默丢弃

## 4. CLI 子命令

- [ ] 4.1 `internal/cli/uninstall.go`：`uninstall <tool>`（显示命令 → 询问代跑 → 残留检测 → 询问清理）、`--residue`（仅列出）、`--yes`（跳过交互）、`--json`（契约保护）
- [ ] 4.2 退出码：0 成功 / 1 错误 / 2 黑名单或无该工具；接入 `run.go` 分发与 usage
- [ ] 4.3 CLI 测试：mock 扫描结果 + 各分支（显示命令/代跑/仅残留/黑名单/无工具）+ 退出码

## 5. GUI 集成

- [ ] 5.1 `gui/service.go` 绑定：`UninstallStart(tool)`（返回官方命令与占用摘要）、`UninstallRunOfficial()`（代跑，输出事件/轮询）、`UninstallResidue()`、`UninstallTrash(ids)`；黑名单拦截在服务层
- [ ] 5.2 详情页「卸载」按钮（danger 样式）；确认弹窗（命令 + 复制 + 代跑/自行执行）
- [ ] 5.3 残留列表弹窗：全选默认、凭证标红、一次确认；完成后 rescan + 回收站状态更新
- [ ] 5.4 输出展示：代跑输出经事件或轮询（参考 update 进度经验：macOS 事件不可靠则轮询）；失败明确回退"复制命令自行执行"

## 6. i18n

- [ ] 6.1 新增 `un.*` 键域三语（confirmTitle/officialCmd/runOfficial/copyCmd/residueTitle/residueNone/residueCredential/trashConfirm/blockedSystem/done/failed 等）；parity 测试通过

## 7. 端到端验证

- [ ] 7.1 三语走查：GUI 卸载流程（确认 → 代跑/复制 → 残留 → 回收站）、CLI 各分支
- [ ] 7.2 安全验证：黑名单工具被拒；USER 残留只能进回收站（恢复面板可还原）；`--yes` 不直删
- [ ] 7.3 回归：默认 zh-CN 下全量 go test + 前端 tsc/vitest 通过；README 补充卸载说明
