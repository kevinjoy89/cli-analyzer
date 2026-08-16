# Design: 指纹缺口修复（PATH 覆盖 / 探测解耦 / 历史一致）

## Context

`2026-08-15-scan-change-detection` 的指纹只 stat 上次扫描的测量路径。审查确认三个功能缺口（见 proposal Why），本变更在保持"仅 stat 不递归、毫秒级"约束下补齐。

## Goals / Non-Goals

**Goals**
- 新装工具 / PATH 目录新增二进制时，GUI 启动与 CLI 非 `--refresh` 自动触发全量扫描
- 缓存命中启动时版本列有值（恢复旧版行为）
- CLI 非 `--refresh` 真实扫描与 `--refresh` / GUI 一样记录历史；消除 GUI 重复记录
- 指纹采集仍仅 stat（PATH 目录顶层条目变化由目录 mtime 承载，不 ReadDir、不递归）

**Non-Goals**
- 不做 fsnotify / 不读 PATH 目录内容（mtime 语义已覆盖增删）
- 不改变 `last-scan.json` 格式
- 不改变手动"重新扫描"/`--refresh` 的强制全量语义
- 不改变 probe 缓存格式

## Decisions

### D1: 指纹追加 PATH 发现目录条目（stat，不 ReadDir）

`ComputeFingerprint` 在结果测量路径之外，追加 `platform.PathDirs(true)` 中**存在**的目录的 `FingerprintEntry`（与发现逻辑同一来源，含 `augmentUserDirs` 补齐目录——GUI 最小 PATH 场景同样覆盖）。

- 新增二进制进入既有 PATH 目录 → 目录 mtime 变化 → 指纹不一致 → 全量（核心场景：`npm i -g` 新包、新脚本放入 ~/.local/bin）
- 新目录加入 PATH → 新条目出现 → 全量
- PATH 目录被移除 → 条目缺失 → 全量（保守方向）
- 不存在的目录不产生条目（与既有语义一致）；创建后出现条目 → 全量
- 成本：PathDirs（env split + 少量 stat/glob）+ 每个目录一次 stat，毫秒级

PATH 字符串本身不单独入指纹：目录条目集合已编码"哪些目录参与发现"，重排/重复等外观变化不影响发现结果，不值得触发重扫。

### D2: mtime 精度纳秒化

`statEntry` 的 `MTime` 改为 `st.ModTime().UnixNano()`。与 probe 缓存一致；同一秒内同大小替换可检测。旧指纹（秒级）与当前计算不一致 → 保守一次全量后自愈。目录 mtime 在 NTFS/APFS/ext4 均以子项增删更新，纳秒精度只增敏感度，方向安全。

### D3: `ScanIfUnchanged` 返回真实扫描标志

签名 `(*ScanResult, error)` → `(*ScanResult, bool, error)`（bool = 是否执行了全量扫描）。缓存命中 = false；指纹缺失/不一致走全量 = true。调用点：

- **GUI `ScanIfChanged`**：用 `scanned` 替代 `prev.ScannedAt != res.ScannedAt` 启发式。CLI `--refresh` 先扫描时，GUI 命中新缓存 → `scanned=false` → 不重复记录（修复重复历史）。
- **CLI 非 `--refresh`**：`scanned && len(filters)==0` 时 `history.Record`（过滤扫描不写历史，与 `--refresh` 一致）；`--no-cache` 走 `Scan`，视为真实扫描同样记录。

### D4: GUI 版本探测与历史解耦

`ScanIfChanged`：`probeAll` 无条件执行（probe 缓存按 path+size+mtime 秒回；真实扫描后探测结果自然更新）；仅 `scanned` 时 `history.Record`。缓存命中启动：`scan:done`（版本空）→ `probe:done`（版本补齐），恢复旧版行为。探测结果仍不写回 `last-scan.json`（与旧版一致，重启靠 probe 缓存秒回补齐）。

### D5: GUI 启动 busy 状态

`main.ts` 启动 `ScanIfChanged()` 前 `flows.setScanning(true, t('ui.scanning'))`；`scan:done` 处理器首行 `setScanning(false)` 已存在，错误路径 catch 同样复位。按钮禁用 + 转圈与手动重扫一致，防并发扫描。

### D6: 代码卫生

- 删 `Options.Refresh`（无调用点）与 `scan()` 的 `!opts.Refresh` 分支
- `service.go` Startup 注释改为描述新流程（前端驱动 ScanIfChanged + 无全量动效差异说明）
- `flows.ts` 中段 `import {KIND_TONE, RESTORE_ICON, TRASH_ICON} from './render'` 合并进顶部（render 不 import flows，无循环）
- `menu.ts` 用 `document.getElementById('menuBar')` + 空值守卫替代 `el()`（el 缺失即 throw，死守卫）

### D7: 测试策略

- `fingerprint_test.go`：PATH 条目断言改为动态计数（结果条目 + 当前存在的 PATH 目录数）；mtime 断言改 UnixNano
- `unchanged_test.go`：适配新签名；新增场景——PATH 临时目录加入新文件 → 指纹变化 → 自动全量（t.Setenv PATH 指向临时目录）
- 前端无需新单测（行为由既有 vitest 守护），tsc 验证

## Risks / Trade-offs

- [PATH 目录误触发] 用户向 PATH 目录拷入任意文件 → 多一次全量扫描，自纠正，无害
- [旧指纹格式] 无 PATH 条目 / 秒级 mtime → 一次保守全量后自愈（安全方向）
- [探测耗时] 冷 probe 缓存启动探测有 3s 单条上限，与旧版一致；热缓存秒回
- [PATH 目录 mtime 粒度] FAT 类 2s 粒度：扫描与下次判定间隔通常远大于 2s；纳秒精度下不增加盲区

## Migration Plan

无持久化格式变更（`last-scan.json` 不动）；指纹文件增量扩展，旧文件经一次全量自愈。
