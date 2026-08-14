# Design: Windows 测试可移植性（多开发者维护）

## Context

CI 矩阵 ubuntu+macos，Windows 开发者本地的 `go test` 失败无人发现。实机归类 23 个失败为四类（详见 proposal.md）。本设计统一处理策略：**真实 bug 修生产、unix 专属语义显式跳过、夹具平台化、结构性保障（Windows CI + LF 归一）**。

## Goals / Non-Goals

**Goals**
- `go test ./internal/...` 在 Windows 与 unix 上全部通过（除本会话沙箱 fsync 伪影——真实环境无此问题）
- 顺带修复 Windows 上真实存在的归因 bug（versioned/npm/pipx/rustup）
- CI 三平台矩阵 + Go 文件 LF 归一，防止回归

**Non-Goals**
- 不弱化 unix 专属语义测试（跳过带原因注释，断言保持完整）
- 不引入新依赖（分隔符处理用标准库）
- 不改变产品行为（除 Windows 归因 bug 的修复本身）

## Decisions

### D1: 分隔符无关（生产修复）

`versionedMatch`/`npmPkgMatch`/`pipxMatch` 原 `strings.Split(real, "/")` 只认正斜杠；`toolchainVersion` 同理。Windows 上 real path 为反斜杠 → 全部失配 → versioned/npm/pipx/rustup 归因在 Windows 上失效（测试用 filepath.Join 构造路径即暴露）。统一改为 `splitPath`（把 `\` 替换为 `/` 后切分），重建路径用 `filepath.FromSlash` 还原本机分隔符（下游 filepath.Join 亦会归一，双保险）。`under()` 已用 `filepath.Separator`，天然平台正确，不动。

### D2: unix 专属语义显式跳过

- `TestRootXDGEnv`（XDG 根不存在于 Windows）、`TestPathDirsAugmentUserDirs`（Windows augment 是无操作）、`TestResolveCleanableGlob`（p10k 缓存锚定 XDG cache 根）、`TestProbeDPKGManaged`（fake dpkg 是 sh 脚本，Windows 无解释器）、`TestResolveCommandViaAugmentedPath`（unix PATH 增强）→ `runtime.GOOS == "windows"` 时 `t.Skip` + 原因注释。
- `TestPathDirsDedup` 不跳过而是平台分支：Windows 用 `;` 分隔 PATH、`%WINDIR%\System32` 作为系统目录断言——该用例的语义（去重 + 系统目录跳过）在双平台都成立，只是夹具不同。

### D3: 夹具平台化

- 路径字面量 → `filepath.Join`（TestUnder/TestSumMaximal/TestFootprintOf）：不改变断言值，只让分隔符随平台。
- HOME 依赖 → 同时设置 `USERPROFILE`：Windows 的 `os.UserHomeDir` 优先读 USERPROFILE，HOME 只在 unix 生效。涉及 pyenv/rustup derive 用例、uninstall `fakeRoots`、cli `setupUninstallEnv`。
- uninstall 残留用例的 config 根按平台选择：unix `XDGConfig`、Windows `AppData`（generic 规则在 Windows 映射到 AppData）——测试创建夹具的根与 Residues 解析的根一致。
- `TestCacheRoundTrip`/`setupUninstallEnv` 隔离 `LOCALAPPDATA`：Windows 上 CacheRoot 回退到 `%LocalAppData%\cli-analyzer`，不隔离会读写真实缓存（污染 + 慢）。

### D4: 结构性保障

- CI 测试矩阵 + `windows-latest`：本应用是 Wails Windows 应用，Windows 是一等开发/发布平台；任何 Windows 回归在 PR 即失败。
- `.gitattributes` += `*.go text eol=lf`：gofmt 需要 LF；CRLF 会让 `gofmt -l` 全库误报（本会话实测）。属性只影响检出，git 仓库内本就是 LF，无需 renormalize 提交。
- 沙箱伪影（trash/cleaner 的 `f.Sync()` Access denied）：经独立探针确认是 DSH 沙箱拒绝 FlushFileBuffers，非产品问题；GitHub Actions Windows runner 与真实开发机 fsync 正常，不改代码。

## Risks / Trade-offs

- [unix 专属测试在 Windows 上不再执行] → 其语义在 Windows 不成立（根/机制不存在），跳过比"假通过"诚实；Windows 专属行为由 platform 层 Windows 测试覆盖
- [`.gitattributes` 改变工作树换行] → 仓库内本就是 LF，只影响检出（Windows 工作树 .go 变为 LF）；gofmt 本地检查随之恢复正常
- [Windows CI 增加矩阵成本] → 与多开发者正确性收益相比可接受；且能立即捕获本次修复的回归

## Migration Plan

无持久化格式变化。`.gitattributes` 生效后，Windows 工作树 Go 文件将在下次检出时归一为 LF；CI 下一次运行即三平台矩阵。
