# 磁盘占用不做 inode 去重；APFS clone 的高估作为已知局限记录

## 背景

`internal/disk` 逐文件累加 `st_blocks`（物理块数）得到目录大小，注释里已写明 "Hard-linked files are counted once per path (v1 accepted limitation)"。`fix-cleanable-sum-overlap` 修正的只是**路径包含**层面的重复（父子目录并存），没有触及**物理块共享**层面。

macOS 上物理块共享有两条来源：

1. **硬链接**：同一 `(st_dev, st_ino)` 出现在多个路径下 —— inode 去重可以完全消除。
2. **APFS clonefile**：`cp -c`、Finder 复制、Xcode DerivedData、`clonefile(2)` 产生的副本**各有独立 inode，但共享物理块** —— inode 去重**完全无效**，且用户态没有廉价的探测手段（`st_blocks` 仍按完整大小报告，`du` 也一样）。

在决定是否引入全量 inode 集合（内存与扫描时长都会涨）之前，先实测偏差幅度。

## 测量

新增 `scripts/measure-dedup.py`：对同一批目录同时给出「逐路径求和」与「按 `(dev, ino)` 去重」两个数，并统计重复 inode 数。

本机实测（2026-10-20，macOS，APFS）：

| 目录 | 文件数 | 路径求和 | inode 去重 | 重复量 | 占比 | 重复 inode |
|---|---|---|---|---|---|---|
| `~/.npm` | 102,560 | 18.0 G | 18.0 G | 10.1 M | 0.05% | 1 |
| `~/.cache` | 582,790 | 19.9 G | 19.8 G | 68.3 M | 0.34% | 2 |
| `~/Library/Caches` | 200,172 | 20.9 G | 20.8 G | 93.1 M | 0.43% | 344 |

结论：**硬链接造成的重复在 0.05%–0.43% 之间**，绝对值 10–93 MB，相对本应用动辄数十 GB 的统计口径可忽略。

APFS clone 的验证：用 `cp -c` 复制一份 200MB 文件后，脚本的两个数**完全相同**（都是 400M），`du -sh` 也报 400M —— 证实 inode 去重对 clone **无任何效果**，而 `st_blocks` 无法区分共享块与独占块。

## 决定

**不引入 inode 去重。** 理由：

- 收益上限约 0.4%，而代价是每个扫描都要维护一个全量 `(dev, ino)` 集合（本机 `~/Library/Caches` 单目录就 20 万文件，全盘扫描是百万级），内存与 CPU 都会明显上涨。
- 对 clone 无效 —— 真正可能造成大偏差的那一类（Xcode DerivedData、Finder 复制的大文件）恰好是 inode 去重抓不到的，做了也只解决次要的那一半。

**APFS clone 的重复计入作为已知局限写进 README**，与已有的「符号链接不跟随」「硬链接按路径计」并列。用户在树形下钻里看到的单目录字节数可能与 Finder「显示简介」不一致，这是可解释的差异，不是 bug。

## Considered Options

- **inode 去重**：能精确消除硬链接重复，但实测收益 0.05%–0.43%，对 clone 无效。
- **clonefile extent 探测**（`F_LOG2PHYS` / `fcntl` 私有接口）：能真正识别共享块，但依赖非公开 API，跨平台无解，且每次扫描都要对每个文件发 ioctl —— 扫描时长会翻数倍。
- **不做，记录局限**（已选）：零成本，口径稳定，用户在 README 里能看到解释。
- **改用 `du` 等外部工具**：直接违反「无外部依赖」的设计前提（`internal/disk` 包注释明确 "no `du`"）。

## 后果

- `internal/disk/size.go` 的注释需要补一句 APFS clone 的局限（原来只提了硬链接）。
- `scripts/measure-dedup.py` 保留在仓库里，作为将来重新评估的测量工具（若哪天用户报告「数字比 Finder 大很多」，先用它量化再决定）。
- 若将来真的需要精确到物理块，应当作为一次独立的、带性能预算的变更立项，而不是顺手改。
