## 1. 基础：platform.DataRoot + config

- [x] 1.1 在 `internal/platform` 新增 `DataRoot()`：应用数据根（macOS `~/Library/Application Support` / Linux `XDG_DATA_HOME` / Windows `%LOCALAPPDATA%`）下的 `cli-analyzer` 子目录，复用 `Root()` 现有常量
- [x] 1.2 新增 `internal/config`：读写 `<DataRoot>/cli-analyzer/config.json`，字段含保留期（默认 7 天）、过期动作（system-trash/permanent）、默认走回收站（bool，默认 true）
- [x] 1.3 为 `internal/config` 编写单元测试（默认值、读写往返、非法值回退）

## 2. 核心：internal/trash

- [x] 2.1 实现 `trash.Trash(path)`：`stat` 比较 `Dev` 判断同文件系统；同 FS 用 `os.Rename` 移入 `trash/<ts>_<name>/_data` 并写 `info.json`（fsync 落盘）；跨 FS 按配置复制+删或拒绝
- [x] 2.2 实现 `trash.Restore(id)`：读 `info.json`，原路径不存在则 `rename` 还原并删元数据；原路径已存在则改名还原并返回新路径
- [x] 2.3 实现 `trash.Sweep()`：遍历过期项，按配置移系统回收站（macOS `osascript` / Linux `gio trash`，工具缺失时降级彻底删除）或彻底删除；无法解析的目录跳过并记录
- [x] 2.4 实现 `trash.Info()`：统计回收站总占用、项数、最近到期时间
- [x] 2.5 为 `internal/trash` 编写单元测试（同 FS rename 往返、跨 FS 拒绝、过期清除、恢复冲突改名），用临时目录隔离真实文件系统
- [x] 2.6 将 `systemTrash` 按 build tag 拆分为平台文件，Windows 用 `syscall` 调 `SHFileOperationW`（`FOF_ALLOWUNDO`）移入系统回收站，并交叉编译 `GOOS=windows` / `GOOS=linux` 验证

## 3. cleaner 集成

- [x] 3.1 `cleaner.del()` 改为调用 `trash.Trash()`，"立即彻底删除"模式仍走 `os.RemoveAll`；`dryRun` 语义不变
- [x] 3.2 `guard` 将回收站根目录加入 forbidden 列表；`guardSub` 不受影响
- [x] 3.3 scanner/disk 遍历时排除回收站目录，并核对现有 `cleaner`/`scanner` 单元测试不回归

## 4. GUI 绑定与前端

- [x] 4.1 `gui/service.go` 新增 `TrashInfo` / `Restore` / `PurgeNow` / `SetTrashConfig` 绑定
- [x] 4.2 frontend 底部状态栏常驻显示"回收站占用 X · 到期时间"，空回收站显示 0 B
- [x] 4.3 frontend 回收站面板：项目列表、单项恢复、立即清除、配置（保留期/过期动作/是否走回收站）
- [x] 4.4 `main.go` 菜单增加首选项入口：macOS App 菜单 About 之下"首选项…"（Cmd+,）；Windows/Linux File 菜单 Quit 上方"首选项…"（Ctrl+,）
- [x] 4.5 frontend 首选项模态面板：保留期 / 过期动作 / 默认走回收站，改动经 `internal/config` 落盘并即时生效

## 5. CLI

- [x] 5.1 `clean` 新增 `--permanent` 立即彻底删除选项
- [x] 5.2 新增 `trash` 子命令：`list` / `restore <id>` / `empty`

## 6. 收尾

- [x] 6.1 全量测试 `go test ./...`
- [x] 6.2 更新 README：清理语义（延迟删除 + 7 天保留）与 `trash` 命令用法
