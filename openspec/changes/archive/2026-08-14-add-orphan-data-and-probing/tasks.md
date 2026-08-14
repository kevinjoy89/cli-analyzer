## 1. 非 CLI 排除体系（前置地基）

- [x] 1.1 在 `internal/platform` 新增排除表：`VendorExclusion{Pattern, Allow}` 结构 + 预置厂商清单（microsoft/amazon/google/netsarang/adobe/tencent/wechat/qq/discord/slack/steam/obsidian/notion/telegram/zoom/teams/jetbrains），含各厂商纯 CLI 例外（az/aws/gcloud/gsutil/bq）
- [x] 1.2 实现路径片段精确匹配函数（按分隔符拆片段、小写比较、剥 `.exe/.cmd/.bat` 扩展名后做 allow 匹配）
- [x] 1.3 重构 `scanner.discoverExecs`：替换 `isVendorInstallDir` 子串匹配为片段精确匹配 + allow 例外；保留 dirHasGUI 整目录兜底
- [x] 1.4 结构性 GUI 信号判定（macOS Containers bundle-id 形态、Windows Packages `_<hash>` 形态）作为平台函数
- [x] 1.5 单测：片段匹配不误伤（code/opencode）、allow 例外保留纯 CLI、伴侣不例外、结构性信号判定
- [x] 1.6 三平台交叉编译 + vet + 全量测试通过

## 2. 孤儿数据过滤与扫描管线

- [x] 2.1 在 `findUnattributed` 增加过滤层：系统/OS 目录、本应用自身、结构性 GUI 信号、厂商排除表（目录名精确判例）
- [x] 2.2 将 `Unattributed` 计算从 `opts.Full` 门控改为默认执行（CLI `--full` 语义保留兼容）；结果随扫描缓存持久化
- [x] 2.3 孤儿体积扫描内并行计算（sizer.WalkAll，与工具数据共用一次 walk），结果随扫描持久化
- [x] 2.4 CLI `scan`/`scan --json` 输出孤儿数据字段（路径、大小、数据根类型）
- [x] 2.5 单测：认领排除、系统/结构信号/厂商排除、体积计算、CLI JSON 契约
- [x] 2.6 i18n 三语键：孤儿小节标题、条目说明、移入回收站按钮文案（parity 测试通过）

## 3. 孤儿数据 GUI

- [x] 3.1 GUI 工具面板"未认领数据"标签页（与工具列表并列）：按数据根分组条目（路径/大小），组标题展示真实根路径
- [x] 3.2 新绑定：孤儿条目移入回收站（复用 trash 移入路径，USER 级、可恢复、无永久删除；单条/批量均二次确认）
- [x] 3.3 处置后立即刷新右下角回收站占用 + 列表状态（本地移除已移入项，重扫兜底校准）
- [x] 3.4 空态处理（无孤儿时不显示标签页）+ 顶部汇总卡总占用展示

## 4. 健康探测

- [x] 4.1 探测模块：`--version`/`-V`/`--help` 依次尝试，首个 0 退出且有输出者胜出
- [x] 4.2 超时与进程组终止（unix Setpgid+kill、windows 终止）+ 并行 worker（限 N=8）
- [x] 4.3 结果缓存：键 (real, size, mtime)，存 CacheRoot；文件未变不重探
- [x] 4.4 Windows 输出解码：UTF-8 失败转 GBK（golang.org/x/text），剥离控制字符
- [x] 4.5 探测结果写入 `ScanResult.Tools[].Version`（既有 JSON 键填充，契约不变）；失败降级保持原值、无错误事件
- [x] 4.6 GUI 版本列"探测中…"→ 完成后更新；缓存命中直接显示
- [x] 4.7 单测：探测顺序、超时终止、缓存命中/失效、GBK 解码、降级不阻塞

## 5. 回归与收尾

- [x] 5.1 全量测试（Go 单测 + 前端 TS/test + i18n parity + 三平台交叉编译）
- [x] 5.2 手动验证：Windows 扫描无 NetSarang/系统噪音；孤儿小节展示与回收站恢复；版本列填充
- [x] 5.3 文档同步（README/README.zh-CN 如有 CLI 输出示例更新）
