## Context

现有 `internal/cleaner` 的 `del()` 用 `os.RemoveAll` 永久删除 SAFE 项，`guard`/`guardSub` 已做路径校验（绝对路径、无 `..`、非 forbidden 根、不删当前版本）。`internal/platform` 已有 per-OS 目录抽象（`Root(MacAppSupport/XDGData/LocalAppData)`）和 `CacheRoot()` 先例。内置回收站成立的前提是**同文件系统内 `os.Rename` 瞬时且不占双份空间**——本项目清理目标（`~/.cache`、`~/.npm`、`~/Library/Caches`、brew Cellar…）绝大多数与家目录同挂载点，前提基本成立。动机见 proposal.md - Why，行为契约见 specs/trash-recycle-bin/spec.md。

## Goals / Non-Goals

**Goals:**
- 把 `cleaner` 的永久删除改为"先入内置回收站"，保留 SAFE 门禁与路径守卫不变
- 7 天内可恢复，到期自动清除（默认移系统回收站、可配彻底删除）
- 诚实展示"已清理"与"空间已释放"的差异
- 回收站自我保护：扫描排除、guard forbidden
- 与 usage-trends change 共用 `platform.DataRoot()` 应用数据根

**Non-Goals:**
- 不实现系统回收站的原生 API（依赖平台现有工具/命令）
- 不做跨文件系统的复制+删除优化（降级处理即可）
- 不做回收站列表的分页/搜索/批量恢复（MVP 仅基础列表 + 单项恢复）
- 不做回收站占用超限的主动提醒（留给趋势 change 的阈值提醒延伸）

## Decisions

### D1: 回收站目录位置 → 平台应用数据目录

```
macOS:   ~/Library/Application Support/cli-analyzer/trash/
Linux:   ~/.local/share/cli-analyzer/trash/
Windows: %LOCALAPPDATA%\cli-analyzer\trash\
```

复用 `platform.Root()` 三个现有常量，新增 `platform.DataRoot()`（返回 `cli-analyzer` 应用数据根），trash 为其子目录。

- **理由**：回收站语义是"待恢复的高价值数据"，必须放在语义为"持久"的数据目录。放缓存目录（`~/Library/Caches`、`~/.cache`）会被系统维护任务或用户手动清理，直接违背 7 天保留承诺。
- **备选**：`CacheRoot()` 缓存目录（否决——语义冲突，见上）。

### D2: 每项一个独立目录 + info.json

```
trash/
  20260811-183000_npm___cacache/
    info.json        # { original, tool, kind, bytes, trashedAt, expiresAt }
    _data/           # rename 进来的实际内容
  20260811-183200_kimi_versions-0.3.1/
    ...
```

- **理由**：每项独立可恢复、可单独清除，`os.Rename` 原子性保证断电安全；`info.json` 每次搬移/恢复后落盘（fsync）。
- 目录名带时间戳前缀 + 清洗过的原始 basename，天然规避同名冲突。

### D3: 删除流程 → trash.Trash()，guard 原样保留

`cleaner.Clean` 的 `del()` 分支改为：

```
trash.Trash(path)：
  1. stat(path) 与 stat(trashDir) 比较 Dev —— 同文件系统？
     同 FS   → os.Rename(path, trash/<ts>_<name>/_data) + 写 info.json
     跨 FS   → 按配置：复制+删除（保留恢复能力） 或 拒绝并提示
  2. "立即彻底删除"模式 → 仍走 os.RemoveAll，但 guard 照旧先行
```

- **理由**：把新逻辑收敛在 `internal/trash`，`cleaner` 只多一个分支；SAFE 门禁和 `guard`/`guardSub` 完全不改动，安全模型不回退。

### D4: 恢复流程 → 原路径冲突时改名还原

读 `info.json.original`，目标不存在则 `rename(_data, original)` 并删 info.json；目标已存在则改名（追加 ` (restored)` 或时间戳）并提示新路径。

### D5: 过期清除时机 → 扫描时顺带执行

每次扫描（`scanner.Scan`）入口处先调用 `trash.Sweep()`：遍历 trash 目录，对 `expiresAt < now` 的项目按配置执行——默认移系统回收站，可配置彻底删除；系统回收站不可用时降级为彻底删除并记录。各平台系统回收站实现按 build tag 拆分：macOS `os.Rename` 到 `~/.Trash`、Linux `gio trash`、Windows 用标准库 `syscall` 调 `shell32.dll` 的 `SHFileOperationW`（`FOF_ALLOWUNDO` 进入回收站，**纯 Go 无需 cgo**，`//go:build windows` 文件隔离，不影响 macOS/Linux 交叉编译）。

### D6: 配置持久化 → `<DataRoot>/cli-analyzer/config.json`

新增 `internal/config`：保留期（默认 7 天）、过期动作（system-trash/permanent）、默认是否走回收站（默认 true）、立即彻底删除不受影响。与趋势 change 的阈值配置同文件。这些配置由 GUI 首选项面板（见 D8）读写，默认值在此落盘，面板修改即时生效。

### D7: GUI/CLI 暴露

- `gui/service.go` 新增绑定：`TrashInfo()`、`Restore(id)`、`PurgeNow(ids)`、`SetTrashConfig()`；底部状态栏常驻"回收站 X · 到期时间"
- `internal/cli`：`clean --permanent`（立即彻底删除）；新增 `trash list | restore <id> | empty` 子命令

### D8: 配置入口 → 首选项面板

配置统一收敛到 GUI 的"首选项"面板：

- **菜单入口**：macOS 在 App 菜单的 About 之下加"首选项…"（快捷键 Cmd+,，符合平台惯例——标准 App 菜单中 About 下方正是 Preferences 位）；Windows/Linux 在 File 菜单的 Quit 上方加"首选项…"（快捷键 Ctrl+,）
- **面板实现**：前端模态面板（原生 DOM，延续零依赖风格），集中展示并编辑回收站配置（保留期 / 过期动作 / 默认走回收站）；usage-trends 的 cleanable 阈值复用同一面板，不重复实现入口
- **读写**：经 `internal/config` 读写 `<DataRoot>/cli-analyzer/config.json`，修改后即时生效并落盘

## Risks / Trade-offs

- [跨文件系统目标无法瞬时搬移] → D3 按配置降级为复制+删除或拒绝，UI 提示
- [Windows `SHFileOperationW` 官方标记 deprecated（建议换 `IFileOperation`）] → 该 API 在所有 Windows 版本仍正常工作，回收站删除这一用途不受影响；若未来需要迁移到 `IFileOperation`（COM），改动仅局限于 `systemtrash_windows.go`
- [Linux 无 `gio`/`trash-cli` 命令] → 过期动作检测不到工具时降级为彻底删除并提示
- [回收站被用户手动改动、目录结构被破坏] → `Sweep()` 跳过无法解析的项并记录，不影响其余项
- [大文件在保留期内空间不释放，用户误以为清理无效] → UI 诚实展示回收站占用与到期时间（spec: 回收站占用可见）
- [rename 后原路径又被占用的极端情况] → D4 改名还原 + 提示，不覆盖

## Migration Plan

无数据迁移。旧版本无回收站；新版本首次运行时创建 trash 目录，`cleaner` 行为从"直接删"变为"入回收站"，SAFE 门禁不变，无破坏性变更。回滚：旧版本二进制可直接使用（回收站遗留目录不影响扫描，因其已被排除）。

## Open Questions

无。
