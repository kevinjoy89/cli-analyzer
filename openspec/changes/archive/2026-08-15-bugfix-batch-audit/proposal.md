## Why

一次覆盖 15 个模块的 bug 审计（24 轮、47 条发现）修复了多类正确性缺陷：归因错位（共享 bin 目录整体计入安装根、nvm 布局误判、自定义 GOPATH/GOBIN/PYENV_ROOT/RUSTUP_HOME/CARGO_HOME 漏归因、symlink-to-dir 二进制解析、备份误列 symlink 真实目标）、数据风险（回收站元数据写失败成永久孤儿、Purge 复用过期配置未真正永久删除、降级删除忽略错误、原子写固定 .tmp 名并发互相覆盖、probe 超时路径数据竞态）、安全缺口（卸载黑名单缺 pip/python@/go@ 形态、GUI OrphanTrash 接受任意路径、cleaner 缺 Windows 系统根守卫、应用自身缓存被列为 SAFE 可清理）、行为错误（过滤扫描污染历史趋势、TopGrowers 用窗口外旧扫描、时钟回拨后更新检查永不刷新、下载取消不可靠、checksums BOM 解析失败、C/POSIX locale 回退中文、子命令 --help 退出码不一致、排序非确定性）。

## What Changes

- **scanner 归因**：node/git 安装根仅当目录为"纯运行时目录"（不含家族外命令）时整体计入；nvm 的 `versions/<家族>/<版本>` 布局不再误判为版本化安装器；symlink 指向目录时解析到 `<dir>/<name>` 真实文件；go 工具按真实 GOPATH（`go env GOPATH`）归因 pkg/mod；PYENV_ROOT/RUSTUP_HOME/CARGO_HOME/GOBIN 环境变量优先；gitk/git-gui 并入 git 家族；rustup 本体归类 InstRustup；工具/清理项/回收站列表/增量排行统一确定性次键排序；应用自身从归因与展示中剔除
- **trash/uninstall 语义**：Purge 为显式永久删除（CLI `trash empty` 需确认、GUI 有确认弹窗），自动过期 Sweep 仍遵循 ExpireAction 配置；writeInfo 原子写 + 失败回滚；Restore 重建已删除的父目录；降级删除失败保留 info.json 待重试；卸载黑名单前缀匹配 python@/python3./node@/go@ 并补 pip/pip3
- **updater**：缓存限流窗口按规格为 4h；时钟回拨（LastCheckAt 在未来）视为缓存失效；错误链 %w 保留 context.Canceled；checksums.txt 剥 UTF-8 BOM；下载循环逐轮检查 ctx.Err()；Windows `cmd start` 路径加引号（空格与 cmd 元字符）
- **cli/gui 一致性**：过滤扫描不写历史；无缓存时 uninstall 报可区分的错误；CLI runOfficial 与 GUI 同享增强 PATH；OrphanTrash 白名单校验；子命令 --help 统一 exit 0；未知横线参数走 CLI usage 而非 Wails 分支；cleaner 用户可见消息与 scan 失败消息 i18n 三语
- **并发与健壮性**：config/cache/probe/trash 原子写改用唯一临时名；probe Start/Wait 分离消除 cmd.Process 数据竞态；子进程 stdout/stderr 双写加锁；history rows.Err() 检查并向上传播；humanBytes 后缀表覆盖 int64 全范围

## Capabilities

### New Capabilities
<!-- 无新能力 -->

### Modified Capabilities
- `app-update`: 限流缓存窗口明确为 4 小时（实现与文档此前 24h 不一致，按实现收敛）

## Impact

- **代码**：internal/{scanner,trash,uninstall,updater,history,cleaner,cli,config,probe,rules,platform,i18n}/、gui/、frontend/、main.go
- **行为**：`trash empty` / GUI 彻底删除变为真实永久删除（此前转系统回收站，磁盘空间未释放）；扫描结果排序可复现；GUI 启动缓存与趋势数据不再被过滤扫描污染
- **不引入**：无新命令行参数（`trash empty --yes` 为跳过确认的脚本入口）；无数据格式变更
