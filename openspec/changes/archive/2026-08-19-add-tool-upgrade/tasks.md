## 1. 共享执行管道抽取

- [x] 1.1 新建 `internal/cmdexec/`，将 `internal/uninstall/` 的 `ResolveCommand`、`AugmentedPathEnv`、`WithPath` 及对应测试迁移过去（纯移动，行为不变）
- [x] 1.2 更新 `internal/cli/uninstall.go`、`gui/service.go` 的 import 指向共享包；若保留 `uninstall.ResolveCommand` 兼容别名则一并处理，最终调用方统一走 `cmdexec`

## 2. internal/upgrade 检测

- [x] 2.1 实现按来源检测：brew `outdated --json=v2 <公式>`、npm `outdated -g --json <包>`、pipx `list --json` + `pip index versions <包>`、cargo `install --list` + `search <名> --limit 1`（命令可注入假实现，供测试）
- [x] 2.2 判定逻辑：包管理器口径 current vs latest 比较、`hasUpdate`/`detected` 语义；来源无检测能力（go/versioned/pyenv/other/local-bin 未知）→ `detected=false`
- [x] 2.3 检测失败/命令缺失/解析失败降级：`detected=false`、错误信息可辨识，不伪造版本
- [x] 2.4 单元测试：brew/npm 有更新与已最新、pipx/cargo 双命令、无能力来源、命令缺失、解析失败各案例

## 3. internal/upgrade 命令表与编排

- [x] 3.1 升级命令表镜像 uninstall.Official 结构：brew/npm/pipx/cargo Runnable，local-bin 已知脚本表（uv/poetry/rye 等）仅展示，go/versioned/other 仅提示
- [x] 3.2 `CheckTool(name)` 编排：解析安装来源 → 检测 → 组装 `CheckResult{name, current, latest, detected, hasUpdate, command, runnable}` + `OfficialCommandBySource` 导出
- [x] 3.3 代跑执行：复用 cmdexec 的增强 PATH 执行 `Bin/Args`，输出流式透传、错误收集
- [x] 3.4 单元测试：命令表正确性、local-bin 脚本命中与回退、代跑执行与失败路径

## 4. CLI 集成

- [x] 4.1 `update check <工具>`：有位置参数走 upgrade 检测，`--json` 输出 CheckResult，退出码沿用 updateExitCode 约定（有更新非零）；无参保持 `updater.CheckForUpdates` 原行为不变
- [x] 4.2 `update run <工具> [--yes]`：展示命令 → 确认（--yes 跳过）→ 代跑，复用 uninstall 的确认与超时节奏
- [x] 4.3 usage 文案与测试：check 工具/无参兼容/run 确认与 --yes/JSON 结构/退出码

## 5. GUI 后端绑定

- [x] 5.1 `service.go`：`CheckToolUpdate(name)` 单次异步查询返回 CheckResult
- [x] 5.2 `RunToolUpgrade(name)` + `GetUpgradeStatus()` 轮询 `{running, done, output, error}`（镜像 uninstall 的代跑轮询）
- [x] 5.3 升级成功后对该工具重探测版本（复用 probe），供前端刷新显示；不触发全量扫描

## 6. 前端详情页

- [x] 6.1 详情页「检查更新」按钮：点击禁用防连点、进行中状态
- [x] 6.2 页面守卫：promise resolve 时当前详情页工具不匹配则丢弃结果，不提醒
- [x] 6.3 结果横幅：有更新（current/latest + 命令 + [复制] [代跑]）/ 已最新 / 无法检测（仍显示命令或提示）；代跑进度与输出流式展示
- [x] 6.4 升级完成后刷新详情页顶部版本号与横幅状态
- [x] 6.5 i18n：全部新文案中英双语键键

## 7. 收尾验证

- [x] 7.1 全量 `go test ./...` 与前端构建通过；GUI 手动走查：点击检测、中途离开、代跑成功/失败、local-bin 展示、go 提示（自动化等价覆盖：upgrade 包按来源注入测试、CLI 测试、服务层测试；构建/类型检查全绿）
- [x] 7.2 CLI 手动走查：`update check`（无参兼容）、`update check <工具> --json`、`update run <工具> --yes`（自动化等价覆盖 + 真实二进制冒烟：无参兼容/无扫描 JSON 契约/退出码）
- [x] 7.3 `openspec validate --change add-tool-upgrade` 通过；更新 README 功能列表与 CLI 文档