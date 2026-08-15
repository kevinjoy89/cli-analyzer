## 1. scanner 归因正确性

- [x] 1.1 node/git 安装根纯净目录判定（nodeOnlyDir/gitOnlyDir：unix 按执行位、Windows 按可执行扩展名；官方安装器 .ps1/.man/git-bash 等附属放行）
- [x] 1.2 nvm 布局不再误判 versioned（versions 后必须是版本形态）
- [x] 1.3 symlink-to-dir 解析到真实命令文件（execEntry.Real）
- [x] 1.4 真实 GOPATH 归因 pkg/mod 与下载缓存；GOBIN 并入 GUI PATH 增强
- [x] 1.5 PYENV_ROOT/RUSTUP_HOME/CARGO_HOME 环境变量优先
- [x] 1.6 gitk/git-gui 并入 git 家族；rustup 归类 InstRustup
- [x] 1.7 确定性排序（工具/清理项/回收站列表/TopGrowers 次键）
- [x] 1.8 应用自身（isSelfDataDir）从归因与展示剔除

## 2. 回收站与卸载安全

- [x] 2.1 Purge = 永久删除（CLI empty 确认 + --yes；GUI 确认弹窗）；Sweep 仍按 ExpireAction
- [x] 2.2 writeInfo 原子写；元数据失败回滚数据到原路径
- [x] 2.3 Restore 重建缺失父目录
- [x] 2.4 降级删除失败保留 info.json 待重试
- [x] 2.5 卸载黑名单补 pip/pip3 + python@/python3./node@/go@ 前缀
- [x] 2.6 OrphanTrash 白名单（仅接受当前扫描的 Unattributed 路径）

## 3. updater

- [x] 3.1 限流窗口规格统一 4h（代码/README/docs/openspec spec/注释/测试命名）
- [x] 3.2 时钟回拨（负间隔）视为缓存失效
- [x] 3.3 FetchChecksums/LatestRelease %w 保留错误链
- [x] 3.4 checksums.txt 剥 UTF-8 BOM
- [x] 3.5 Download 循环逐轮 ctx.Err() 检查（取消可靠）
- [x] 3.6 Windows `cmd start` 路径引号（空格 + cmd 元字符）

## 4. cli/gui/历史/本地化

- [x] 4.1 过滤扫描不写 history；缓存写回补齐 ScanTimeMS/Unattributed
- [x] 4.2 uninstall 无缓存错误可区分；runOfficial 与 GUI 同享增强 PATH
- [x] 4.3 子命令 --help 统一 exit 0；未知横线参数走 CLI usage
- [x] 4.4 cleaner/scan 用户可见消息 i18n 三语；C/POSIX locale 映射 en
- [x] 4.5 TopGrowers 用窗口内记录；rows.Err() 传播
- [x] 4.6 rules 通配 HasPrefix（前缀匹配语义）

## 5. 并发与健壮性

- [x] 5.1 原子写唯一临时名（config/cache/probe/trash info）
- [x] 5.2 probe Start/Wait 分离消除 cmd.Process 竞态；输出缓冲加锁
- [x] 5.3 humanBytes 后缀覆盖 int64 全范围；前端 hb() 同步 PB/EB
- [x] 5.4 ClearCache 幂等；go 卸载提示兜底 binName 为空

## 6. 验证

- [x] 6.1 gofmt/vet 全绿；`go test ./... -cover` 14 包全绿
- [x] 6.2 前端 vitest 19/19 + tsc --noEmit
- [x] 6.3 darwin/windows/linux 交叉编译
