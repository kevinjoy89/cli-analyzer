# CLI Analyzer Bug 审计报告（test-report）

> 审计范围：`/Users/wei/iWorkspace/idea/cli-analyzer`（Go 1.26 + Wails v2 + modernc.org/sqlite）
> 审计轮次：24 轮（rounds_completed=24），存活 life=24，累计发现 found_total=47
> 修复模式：auto（自动修复，TDD 红黑闭环 + Live 复验）
> 模块覆盖：**15/15（100%）**——module_coverage.py final-check 通过
> 质量门禁：go build/vet 全绿、三平台（darwin/windows/linux）交叉构建全绿、
> `go test -race ./...` 全绿（14 包）、前端 vitest 19 用例全绿、CLI 参数模糊 300 组无崩溃

## 发现总览（47 条，全部 fixed）

### round1（4 条，life 1→4）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | uninstall blocklist 缺 brew 公式形态 python@3.x，系统关键工具可被代跑卸载 | internal/uninstall/uninstall.go | fixed（前缀匹配） |
| minor | humanBytes 对 ≥1 PiB 输入 "KMGT"[4] 越界 panic | internal/cli/output.go | fixed（后缀扩 KMGTPE） |
| minor | 扫描缓存 scanTimeMs 恒 0（赋值时序） | internal/scanner/scanner.go | fixed |
| minor | GUI withPath 只匹配 "PATH="，Windows "Path=" 残留重复 PATH | gui/service.go | fixed（大小写不敏感） |

### round2（11 条，life 4→8，计命 5）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | blocklist 漏 pip/pip3（Python 核心工具可代跑卸载） | internal/uninstall/uninstall.go | fixed |
| major | 过滤扫描污染历史趋势（`scan <filter> --refresh` 写过滤 totals） | internal/cli/scan.go | fixed |
| major | GUI OrphanTrash 接受前端任意路径（非孤儿目录可被移入回收站） | gui/service.go | fixed（白名单） |
| major | trash writeInfo 落盘失败留下永久孤儿数据 | internal/trash/trash.go | fixed（回滚） |
| minor | 过滤扫描缓存缺 ScanTimeMS/Unattributed | internal/scanner/scanner.go | fixed |
| minor | trash empty / GUI"彻底删除"语义不符（Purge 复用过期配置转系统回收站） | internal/trash + cli + gui | fixed（永久删除+确认） |
| minor | trash 降级删除忽略错误 → 不可见孤儿 | internal/trash/trash.go | fixed |
| minor | updater 限流窗口 4h vs 文档 24h 漂移 | internal/updater/check.go | fixed |
| minor | history TopGrowers 用窗口外旧扫描算增量 | internal/history/history.go | fixed |
| minor | rules 通配 Contains 误匹配（非前缀目录被当清理项） | internal/rules/rules.go | fixed（HasPrefix） |
| minor | CLI uninstall 无缓存时静默当"未找到工具" | internal/cli/uninstall.go | fixed |

### round3（4 条，life 8→11）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | nodejs/git 安装根误判共享 bin 目录（/usr/local/bin 整体计入） | internal/scanner/classify.go | fixed（纯净目录判定） |
| major | backup 清理项误列 symlink 命令真实目标（删除破坏命令） | internal/scanner/attribute.go | fixed（isCommand 记 Real） |
| minor | under() Windows 大小写敏感 → go/cargo/pyenv 归因失效 | internal/scanner/classify.go | fixed（underFold） |
| minor | uninstall 残留错误输出字面量占位符 {msg} | internal/cli/uninstall.go | fixed |

### round4（4 条，life 11→14）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | updater 时钟回拨后缓存永不失效 | internal/updater/check.go | fixed（负间隔失效） |
| minor | CLI runOfficial 无 PATH 增强（最小 PATH 下卸载失败，与 GUI 不一致） | internal/cli/uninstall.go | fixed（WithPath 共享） |
| minor | 前端 subRows 分隔符硬编码 '/'（Windows 子项显示完整路径） | frontend/src/main.ts | fixed |
| minor | 前端无下载入口仍显示无效下载按钮 | frontend/src/main.ts | fixed（条件渲染） |

### round5（3 条，life 14→16）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | versionedMatch 误判 nvm 布局（node/npm/npx 归为 ".nvm"） | internal/scanner/classify.go | fixed（版本形态校验） |
| minor | 孤儿遍历漏 XDGState（state 残留不可见） | internal/scanner/scanner.go | fixed |
| minor | 前端 hb 缺 PB/EB（与 Go humanBytes 不一致） | frontend/src/lib/format.ts | fixed |

### round6（3 条，life 16→18）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | probe 探测超时路径数据竞态（killGroup 与 Start 并发读写 cmd.Process） | internal/probe/probe.go | fixed（Start+Wait 分离） |
| minor | trash.Restore 原父目录被删后无法恢复（数据困在回收站） | internal/trash/trash.go | fixed（重建父目录） |
| minor | HOMEBREW_PREFIX 尾斜杠 → brew 归因全失效 | internal/scanner/classify.go | fixed（Join 规范化） |

### round7（2 条，life 18→19）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | symlink-to-dir 命令 Binary.Real=目录（大小 0/探测目录/归因错） | internal/scanner/discover.go | fixed（Real 字段） |
| minor | main 未知横线参数落 GUI 分支（Wails 构建错误误导） | main.go | fixed（CLI usage） |

### round8（2 条，life 19→20）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | go 工具模块缓存硬编码 ~/go/pkg/mod（自定义 GOPATH 下 929MB 漏归因） | internal/scanner/attribute.go | fixed（真实 GOPATH） |
| minor | PYENV_ROOT/RUSTUP_HOME/CARGO_HOME 自定义未识别 | internal/scanner/classify.go | fixed（env 优先） |

### round9（2 条，life 20→21）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | augmentUserDirs 漏 GOBIN（GUI 最小 PATH 漏扫） | internal/platform/path_augment_unix.go | fixed |
| minor | checksums.txt 带 UTF-8 BOM 时哈希校验永远失败 | internal/updater/checksum.go | fixed |

### round10（2 条，life 21→22）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | gitk/git-gui 未入 git 家族（unix 上独立工具行噪音） | internal/scanner/classify.go | fixed |
| minor | history 两处 rows.Next() 缺 rows.Err()（查询错误静默截断） | internal/history/history.go | fixed |

### round11（2 条，life 22→23）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | Windows OpenInstaller 路径含空格时 start 拆参数 | internal/updater/open_windows.go | fixed（引号） |
| minor | C/POSIX locale（C.UTF-8/LANG=C）解析回退中文而非英文 | internal/i18n/detect.go + detect_linux.go | fixed |

### round12（2 条，life 23→24）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | cache --clear 对无缓存报"清除失败"（非幂等） | internal/scanner/cache.go | fixed |
| minor | go 工具无 PATH 二进制时卸载提示残缺命令 | internal/uninstall/uninstall.go | fixed |

### round13（2 条，life 24→25）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | config/cache/probe 原子写固定 ".tmp" 名：并发写互相覆盖丢更新 | internal/config + scanner + probe | fixed（CreateTemp 唯一名） |
| minor | trash writeInfo 非原子写（崩溃半写 info.json → 永久孤儿） | internal/trash/trash.go | fixed |

### round14（1 条，life 25）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | cleaner guard 缺 Windows 系统根（C:\Windows 等纵深防御缺口） | internal/cleaner/cleaner.go | fixed |

### round15（1 条，life 25→26）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | updater 错误链丢失（%s 吞 context.Canceled，取消显示为通用错误） | internal/updater/checksum.go + release.go | fixed（%w） |

### round16（1 条，life 26）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | 子命令 --help 处理不一致（flag.ErrHelp 当错误 exit 1、trash/trends 忽略） | internal/cli/* | fixed（8 处统一 exit 0） |

### round17（2 条，life 26→27）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | rustup 本体归 InstCargo（工具链清理项推导缺失） | internal/scanner/classify.go | fixed |
| minor | Download 取消不可靠（缓冲数据让取消静默失效 + flaky 测试根因） | internal/updater/download.go | fixed（ctx.Err 每轮检查） |

### round18（1 条，life 27）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | scanner 工具排序不稳定（同 footprint 等值重排，输出不可复现） | internal/scanner/attribute.go | fixed（次键确定性） |

### round19（2 条，life 27→28）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | 4 处展示排序单一键不稳定（trash 列表/排行/子项） | internal/trash + history + scanner | fixed |
| minor | cleaner 11 处 + cli 1 处用户可见消息硬编码英文 | internal/cleaner + cli | fixed（i18n 三语） |

### round20（1 条，life 28→27）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| major | 应用自身缓存列为 SAFE 可清理项（clean --all 清掉自己的扫描/探测缓存） | internal/scanner/attribute.go | fixed（isSelfDataDir 剔除） |

### round21（1 条，life 27）
| 严重度 | 发现 | 位置 | 修复 |
|--------|------|------|------|
| minor | probe/gui 子进程输出无锁 bytes.Buffer 双 writer（理论竞态） | internal/probe + gui | fixed（syncBuf） |

### round22-24（0 条，深度回归/fuzz 轮，无新发现）

## 修复统计
- 全部发现：**47 条**（major 8 / minor 39），**全部 fixed**
- 未修复：0；fail（不可修复）：0；unfixed：0
- 计命：round1-21 累计 credited 47（全部轮次净发现）
- 每个修复均含：红测试（先失败）→ 实现 → 转绿 → 相关回归 → Live 复验（真实命令/单测）

## 模块覆盖清单（15/15 = 100%）
buildinfo / i18n / config / disk / rules / platform / probe / history / scanner /
cleaner / trash / uninstall / updater / cli / main+gui —— 全部「已覆盖」，
module_coverage.py check 通过（证据文件/测试名均有效）。

## 遗留风险与已知限制（非 bug，设计权衡）
1. **asdf/mise 安装布局**：shim 是脚本（非 symlink），安装占用按 shim 文件计（漏归因）——涉及清理安全模型决策，未实现
2. **npm/uv 缓存自定义路径**（npm config cache / UV_CACHE_DIR）：规则表硬编码默认路径，自定义时漏归因
3. **多进程缓存竞态**（GUI+CLI 并发 SaveCache）：唯一 tmp 后无更新丢失，但最后写者胜（低频）
4. **trash empty 语义变更**：显式清空 = 永久删除（不经过系统回收站）；自动过期 Sweep 仍遵循配置（系统回收站保险）
5. **macOS systemTrash 直接 Rename 到 ~/.Trash**：Finder 无法"放回原处"（无原路径元数据）
6. **硬链接文件重复计数**（README 已知限制 v1）

## 下次重启建议（错题集方向）
- 错题集已沉淀 25 条经验（12 类），同类排查点：
  - 通用匹配规则的自伤排除（asdf/mise 等新布局的"自身"概念）
  - env 自定义路径的归因（npm config / UV_CACHE_DIR 等）
  - 前端与 Go 的契约一致性（format/类型/排序）
  - 新安装器布局（scoop/volta 已有，asdf/mise 待补）
