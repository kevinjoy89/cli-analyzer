## Why

清理工具的核心承诺是 **safely**，但现有 cleaner 用 `os.RemoveAll` 永久删除，误删不可逆——SAFE 门禁再严格也无法挽回一次手滑。把"清理"从瞬时事件变成"7 天可反悔的窗口"，才能真正兑现安全承诺，也才能让用户敢于一键清理。

## What Changes

- **延迟删除**：cleaner 的删除流程从 `os.RemoveAll` 改为"先移入内置回收站"（同文件系统内 `os.Rename` 瞬时完成），磁盘空间在回收站保留期内不立即释放
- **内置回收站**：新增应用数据目录下的回收站（`<data-root>/cli-analyzer/trash/`），每个被删项携带元数据（原路径、所属工具、类型、字节数、移入时间、到期时间）
- **恢复**：回收站内未过期项可一键还原到原路径（GUI + CLI），原路径被占用时改名还原
- **自动清除**：到期项默认**移动到系统回收站**（macOS Trash / Windows 回收站 / Linux `gio trash`），可配置为**彻底删除**；清除动作在扫描时顺带执行
- **可选直接删除**：用户可对特定项选择"立即彻底删除"（跳过内置回收站）
- **诚实展示**：GUI 常驻显示"回收站占用 X · 到期释放时间"，把"清理"与"空间释放"两个概念在 UI 上分开
- **自我保护**：scanner 排除回收站目录（防止自我归因/自我清理）；cleaner guard 将回收站路径加入 forbidden 列表

## Capabilities

### New Capabilities
- `trash-recycle-bin`: 延迟删除、恢复、自动过期与过期策略（系统回收站/彻底删除）、可选直接删除、回收站占用展示与自我保护

### Modified Capabilities
<!-- 无：openspec/specs/ 目前为空，且清理语义属新增能力，不改动现有 spec 要求 -->

## Impact

- `internal/cleaner`：`del()` 从 `os.RemoveAll` 改为 trash 流程；`guard` 增加回收站 forbidden
- `internal/scanner`：排除回收站目录；扫描结果新增回收站占用统计（供功能趋势联动）
- **新增** `internal/trash`：跨平台 rename、元数据持久化、过期扫描与清除、恢复
- `internal/platform`：新增应用数据根 `DataRoot()`（macOS App Support / Linux XDG_DATA_HOME / Windows LocalAppData），供回收站与趋势共用
- `gui/service.go`：新增 `TrashInfo` / `Restore` / `PurgeNow` / `SetTrashConfig` 等 Wails 绑定
- `internal/cli`：`clean` 支持立即彻底删除选项；新增 `trash` 子命令（list / restore / empty）
- `frontend`：回收站面板（占用、到期列表、恢复、立即清除、配置）
