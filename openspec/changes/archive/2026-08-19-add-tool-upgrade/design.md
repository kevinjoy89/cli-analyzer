## Context

参见 proposal.md - Why：为被扫描工具补上「官方升级」，与已有的 tool-uninstall（来源→命令→代跑→收尾）构成镜像。

现状约束：
- `internal/updater/` 只管 cli-analyzer **自身**的更新（GitHub Releases + 4h 缓存 + 校验 + 打开安装包），带状态持久化，与本功能无关。
- `internal/uninstall/` 已有完整范式：`Official{Command, Runnable, Bin, Args}` 命令表、`ResolveCommand`（增强 PATH 下解析可执行文件）、`AugmentedPathEnv`/`WithPath`（子进程继承完整 PATH）、GUI 异步代跑 + `GetUninstallStatus` 轮询、CLI 交互确认。本功能直接复用其执行管道与 UI 节奏。
- scanner 已给出每个工具的 `Installer` 与 `Name`（brew 公式名 / npm 包名 / pipx 名…），是检测与命令的键。
  - **npm 例外**：`npmToolID` 会把 scoped 包映射成短工具名（如 `pi` ←
    `@earendil-works/pi-coding-agent`），`Tool.Name` 只剩短名。npm 查询/升级
    按**真实包名**寻址，用短名会静默误报「已最新」（实测 `npm outdated -g
    --json <不存在的包>` 也返回 `{}`）或代跑失败。
    - 修复（code review #2）：`scanner.NpmPackageFor(name)` 逆映射回真实包名。
    - 再修复（code review #3）：逆映射是**启发式**，7 个映射短名（pi/codex/
      claude/omp/dsh/openspec/codegraph）在 npm 上都有**同名真实包**，仅凭短名
      无法区分——若用户装了真实 `pi` 包会被误当成 `@earendil-works/pi-coding-agent`。
      故扫描结果新增 `Tool.Package`（真实包名，`omitempty`），`CheckTool`/
      `OfficialFor` 优先用 `t.Package`，旧缓存回退 `NpmPackageFor`。同一根因的
      `uninstall` 也同步改为 `OfficialFor` 按 `t.Package` 寻址。
- 项目明确处理过国内网络（updater 在 GitHub 不可达时静默失败）——检测查询必须走用户自己的包管理器配置（镜像友好），这是约束而非可选项。

## Goals / Non-Goals

**Goals:**
- 无状态：不缓存、不持久化、无忽略版本——每次点击全新查询，出错静默降级。
- 检测与命令都按安装来源走包管理器自身口径，镜像友好。
- CLI 向后兼容：`update check` 无参行为不变，工具检测靠位置参数进入。

**Non-Goals:**
- 不做批量检测/批量升级（一次一个工具，独立确认）。
- 不拦系统关键工具（无黑名单）。
- 不做自动检查（启动、进详情页都不触发）。
- 不复用 updater 的下载/校验/安装引导——那是应用自身的更新，不是工具升级。

## Decisions

**D1: 新增 `internal/upgrade/` 包，与 `internal/updater/` 严格分离。**
updater 语义是"更新应用自身"（GitHub release、下载、校验、打开安装包），upgrade 语义是"升级被扫描工具"（包管理器查询、官方命令、代跑）。合并会让"更新"概念混乱。备选：放进 updater——否决，职责不同且会拖入缓存/校验逻辑。

**D2: 检测一律调用包管理器自身的查询命令，注册表 HTTP 直连仅作后备。**
```
brew    brew outdated --json=v2 <公式名>        单命令: current+wanted, 空=已最新
npm     npm outdated -g --json <包名>            单命令: current+latest
pipx    pipx list --json                          installed
        + pip index versions <包名>               latest (走 pip 配置)
cargo   cargo install --list                      installed
        + cargo search <名> --limit 1             max_version (走 registry 配置)
go/versioned/pyenv/other/local-bin-未知          无法检测 → 仅命令/提示
```
选包管理器命令而非 HTTP 直连（PyPI/crates.io/registry.npmjs.org）：直连绕开用户镜像与代理配置，与项目"国内网络静默降级"的既有约束冲突。备选已考虑：PyPI JSON API 更"标准"但同样绕开 pip 镜像；`pip index versions` 为实验性命令，输出（`Available versions: ...`）用前缀解析，解析失败按 D3 降级。

**D3: installed 与 latest 均为包管理器口径，绝不与 probe 的 `--version` 值交叉比较。**
`v14.1.1` vs `14.1.1` 之类的格式差会制造误报。probe 版本仅用于详情页顶部展示与升级后刷新，不参与"是否有更新"判定。检测/degraded 语义：查询命令缺失或解析失败 → `detected=false`，横幅显示"无法检测"但仍给出命令；网络失败同理。**不编造版本号，不编造命令。**

> GUI 降级路径（code review #4 修复）：`CheckResult.Error` 仅供 CLI 非 JSON
> 输出与调试，GUI 前端必须按 `detected=false` 降级展示「无法检测 + 官方升级
> 命令」，不得把 `error` 当致命错误提前 return——否则网络失败（国内网络常见）
> 时用户看不到升级命令，违反 D3 的降级契约。

**D4: 升级命令表镜像 uninstall 的 `Official` 结构，但代跑面更窄。**
```
brew   brew upgrade <公式名>           Runnable
npm    npm update -g <包名>            Runnable
pipx   pipx upgrade <包名>             Runnable
cargo  cargo install <包名> --force    Runnable
local 命中官方脚本表 → 展示脚本命令      仅展示 (uv/poetry/rye...)
go    "重新 go install（需模块路径）"   提示
versioned/other/pyenv                  提示
```
local-bin 脚本（`curl … | sh` 形态）比 uninstall 的 `rm -f <path>` 风险等级不同：**只展示不代跑**。已知脚本表为小表，URL 会腐烂（astral 迁移史），未知 local-bin 回退通用提示。

**D5: 执行管道复用 uninstall，抽共享小包 `internal/cmdexec/`。**
`ResolveCommand` / `AugmentedPathEnv` / `WithPath` 从 `internal/uninstall/` 移入 `internal/cmdexec/`，uninstall 与 upgrade 共同调用。避免 upgrade→uninstall 的异味依赖方向；uninstall 调用方（cli/uninstall.go、gui/service.go）同步改 import。这是纯移动+改导入的小重构，不做行为变更。

**D6: GUI 页面守卫是纯前端义务，后端只做两件事：查询、代跑。**
- `CheckToolUpdate(name) → CheckResult{name, current, latest, detected, hasUpdate, command, runnable}`：单次异步调用（Go method，promise 返回），不轮询。
- 前端点击 → 按钮禁用（防连点）→ promise resolve 时若当前详情页工具 ≠ name 则丢弃。
- 代跑沿用 uninstall 的轮询模式：`RunToolUpgrade(name)` 起 goroutine + `GetUpgradeStatus()` 轮询 `{running, done, output, error}`，输出流式展示。

**D7: CLI 参数化收敛在 `update` 家族，不新建动词。**
`runUpdate` 扩展：`check` 后的位置参数 = 工具名时走 upgrade 检测（`--json` 输出 CheckResult），无参保持 `updater.CheckForUpdates` 原行为；新增 `run <工具> [--yes]` 子命令：展示命令 → 确认（`--yes` 跳过）→ 代跑，交互文案复用 uninstall 的确认节奏。退出码约定沿用现有 `updateExitCode`（检测到更新返回非零）。

**D8: 升级完成后轻量重探测版本，不动全量扫描。**
复用 `internal/probe` 对该工具重跑版本探测，刷新详情页顶部版本号与横幅状态。占用数字保持原样，由用户下一次显式全量扫描刷新——升级的即时反馈是"版本变了"，与 uninstall"卸载后必查残留"（残留是其核心承诺）的紧耦合不同。

## Risks / Trade-offs

- [brew outdated 慢（秒~十秒级）] → 纯异步 + 页面守卫 + 进行中状态；用户主动点击才触发，不阻塞其它 UI。
- [`pip index versions` 实验性、输出格式可能漂移] → 前缀解析 + 解析失败走 D3 降级（detected=false 仍给命令）；实现时可换 PyPI JSON（见 Open Questions，不影响规格/任务划分）。
- [cargo search 依赖注册表 index 同步，新发布可能滞后] → 包管理器口径本身如此，与用户手动 `cargo search` 所见一致，可接受。
- [local-bin 脚本表 URL 腐烂] → 小表 + 仅展示不代跑 + 未知回退通用提示，最坏情况是脚本 URL 失效，用户仍可去官网。
- [升级后 re-probe 版本可能不准（shim/别名）] → 仅展示，不参与判定。

## Migration Plan

- 无数据迁移、无配置变更（不碰 `internal/config/`）。
- CLI 向后兼容：`update check` 无参行为不变，`update run` 为新增子命令。
- cmdexec 抽取为纯移动：测试随包迁移，行为断言不变。
- 发布：随下一次 release 上线；GUI 与 CLI 功能同时发布。

## Open Questions

- ~~cmdexec 抽取时 `uninstall.ResolveCommand` 等是否保留薄转发别名以最小化 diff~~（已定：**不留别名**，调用方统一走 `cmdexec`，消除单向依赖抽干后的空壳）。
- ~~pipx latest 获取最终定格在 `pip index versions` 还是 PyPI JSON~~（已定：**`pip index versions`**；前缀固定名解析失败亦同步走整体降级，PyPI JSON 留作未来的备选实现）。

## Code Review #5 修复记录

- **GUI 并发检测竞态（`CheckToolUpdate`）**：原实现先跑网络查询（最长 3min）再记录 `ugTool/ugCommand`。用户查 A 后离开再查 B 时，先发起的 A 查询后返回会覆盖 B 的记录，导致 B 的代跑被误拒（`ugTool != B` → 「不支持代跑」）。修复：**先记录再查询**，保证「最近发起」的检测胜出，陈旧检测不会事后覆盖。新增 `TestCheckToolUpdateStaleDoesNotOverwrite`。
- **前端按钮复位竞态（`startToolUpgrade`）**：`.finally()` 无条件复位当前页按钮。查 A 后离开再查 B（B 检测在途）时，A 的 `.finally()` 会把 B 的「检查中…」按钮误复位成可用，导致重复点击。修复：引入 `upgradeCheckSeq` 序号，仅「最近发起」检测的 `.finally()` 复位按钮。
- **brew 多 keg current 展示（`detectBrew`）**：`installed_versions` 是过时 keg 列表，brew 源码按 `scheme_and_version` 升序（旧→新）。原取 `[0]`（最旧），多 keg 时把已装 2.18.1 的用户展示成 2.17.1（fontconfig 实测）。修复：取最后一个（最新过时 keg）作为 current。新增 `TestDetectBrewMultiKeg`。`HasUpdate` 不受影响（brew 仅列出真正过时的公式）。

## Code Review #6 修复记录

- **cargo 二进制名 ≠ crate 名（`detectCargo`）**：扫描器按二进制名归类 cargo 工具（`rg`），而 cargo 按 crate 名寻址（`ripgrep`）。原实现用二进制名查 `cargo install --list`/`cargo search`，导致 `rg`/`fd`/`delta` 等常见工具永远「无法检测」（测试用 crate 名掩盖了此缺陷）。修复：`parseCargoInstallList` 改为经二进制列表反查 crate 名，`detectCargo` 用 crate 名做 `cargo search`。新增/更新 `TestDetectCargo`、`TestParseCargoInstallList`。
- **cargo 代跑命令用错名（`officialCommand`/`OfficialFor`）**：同一根因，`cargo install <二进制名> --force` 会装错/装不到。修复：新增 `cargoCrateOf`（本地 `cargo install --list` 反查 crate 名），`CheckTool`/`OfficialFor` 对 cargo 用它生成命令；`CheckTool` 分离「命令用 crate 名 / 检测用二进制名」两个参数。新增 `TestOfficialForCargoMapped`、`TestCheckToolCargoMapped`。
- **pipx venv 名 ≠ 发行名（`detectPipx`）**：pipx venv 名 = 发行名 + 可选 `--suffix`（`pipx install --suffix foo uv` → venv `uv-foo`）。原用 venv 名查 `pip index versions` 会查错包。修复：用 `main_package.package`（真实发行名）查询，回退 venv 名。新增 `TestDetectPipxSuffixVenv`。
- **cargo git 安装修订串进 current（`parseCargoInstallList`）**：git 安装条目带修订后缀（`v0.1.0 (git+…):`），原会把修订串进 current 展示。修复：只取版本号本身。新增 `TestDetectCargoGitInstall`。

## Code Review #7 修复记录

- **升级完成后的重探测目标错位（`reprobeToolVersion`）**：原实现升级完成时读 `s.ugTool` 决定重探测哪个工具。代跑 A（brew 最长 5 分钟）期间用户检查 B（`CheckToolUpdate` 覆盖 `ugTool=B`），A 升级完成后会重探测 B（未变）而非 A（刚被升级）——用户看不到 A 的版本刷新。修复：`RunToolUpgrade` 在发起时（校验 `name == ugTool` 处）记录本次代跑目标工具名，经 `runToolUpgrade(name, cmd)` 传给 `reprobeToolVersion(name)`，完成时按发起时的工具重探测。新增 `TestReprobeUsesRunStartTool`。
## Code Review #8 修复记录

- **cargo 检测错误处理不一致（`detectCargo`）**：`detectBrew`/`detectNpm` 遵循「退出码非零但输出有效（`execError`）仍继续解析」的约定（round-1 原则），而 `detectCargo` 对 `cargo install --list`/`cargo search` 的任何错误一律降级，即使输出有效。当 cargo 因 registry 配置/网络以非零码退出却仍打印有效列表时，本可检测却降级。修复：两处均改为 `if err != nil && !execError(err)` 才降级。新增 `TestDetectCargoExit1StillDetects`（用真实 `*exec.ExitError` 验证退出码 1 + 有效输出仍检测成功）。
- **`TestReprobeUsesRunStartTool` 断言强化 + 去 flake**：原 round-7 测试只用不存在的二进制验证「不 panic」，未真正验证重探测命中 A 而非 B。强化为：A 为真实可探测脚本、B 路径不存在，断言 A 被探测（版本被赋值）而 B 保持空；并临时回退实现验证该测试在 bug 存在时确实失败（判别力确认）。同时把探测改为「种子化 probe 缓存 + 真实脚本回退」双路径，消除 `go test ./...` 并行负载下子进程探测偶发 3s 超时的 flake（连续多轮全量通过）。
