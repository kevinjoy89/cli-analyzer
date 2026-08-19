## Why

工具已有「官方卸载」（识别安装来源 → 显示标准命令 → 可代跑 → 残留检测），但升级这个生命周期缺口仍在：用户想知道某个 CLI 工具有没有新版本，只能自己跑 `brew upgrade` / `npm update` 手动比对。应用已识别每个工具的安装来源并探测到当前版本——「官方升级」是卸载的天然镜像：检测新版本、给官方升级命令、可代跑。

## What Changes

- 新增 `internal/upgrade/` 包：逐工具版本检测（brew/npm 单命令、pipx/cargo 双命令）与升级命令表（按安装来源映射，镜像 uninstall 的命令表模式）。
- GUI：工具详情页新增「检查更新」按钮，点击异步查询（无缓存）；查询完成时用户仍停留在该工具页才提醒（否则静默丢弃）；有更新则横幅展示官方升级命令 + [复制] [代跑]，代跑走异步轮询状态；升级完成后对该工具重探测版本刷新显示。
- CLI：`update check <tool>`（带工具名参数 = 查工具新版本；无参数 = 现有应用自身检查，向后兼容）、`update run <tool> [--yes]`（代跑官方升级命令）。
- 安全约束：无黑名单（升级不拦截任何工具）；无缓存、无忽略版本（每次点击全新查询）；来源无法检测时降级为「仅给升级命令」，不编造命令（go/versioned/pyenv/other 只提示）；local-bin 按已知官方脚本表给「重跑官方脚本」，未知者给通用提示。
- 复用：uninstall 的 `ResolveCommand` / `AugmentedPathEnv` / 异步代跑轮询模式；probe 的版本探测。

## Capabilities

### New Capabilities
- `tool-upgrade`: 被扫描工具的版本检测与官方升级。覆盖：逐工具检测（按安装来源选择查询命令、镜像友好）、升级命令映射与代跑（brew/npm/pipx/cargo）、未知来源降级提示（不编造命令）、local-bin 官方脚本表、GUI 详情页按需检测与页面守卫提醒、CLI `update check/run`、升级后重探测版本。

### Modified Capabilities
<!-- 无：既有 capability（app-update 管应用自身更新、tool-uninstall 管卸载）的需求不变。 -->

## Impact

- 新增包：`internal/upgrade/`（检测 + 命令表 + 代跑编排，纯逻辑可单测）。
- 修改包：`internal/cli/update.go`（`check` 支持工具名位置参数 + 新增 `run` 子命令）、`gui/service.go`（新增绑定：`CheckToolUpdate` / `RunToolUpgrade` / 状态轮询）、`main.go`（分发，若需要）。
- 复用：`internal/uninstall` 的 `ResolveCommand`/`AugmentedPathEnv`（或将 PATH 增强逻辑抽为共享）、`internal/probe`（升级后重探测版本）、`internal/i18n`（全部新文案本地化，含中英）。
- 前端：`frontend/src/main.ts` / 详情页模板：检查按钮、页面守卫、结果横幅、代跑进度与状态。
- 无新增第三方依赖。
- 网络：检测一律走包管理器自身的查询命令（尊重用户镜像/代理配置，国内网络友好）；失败静默降级为「无法检测 + 仍给升级命令」，不影响正常使用。