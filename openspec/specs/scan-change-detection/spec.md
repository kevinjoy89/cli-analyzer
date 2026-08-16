# scan-change-detection Specification

## Purpose

为扫描结果提供廉价的变更检测（mtime 指纹，仅 stat 不递归），驱动 GUI 启动与 CLI 非刷新路径在数据未变化时跳过全量扫描、直接复用缓存，在数据变化时自动触发全量扫描。

## Requirements

### Requirement: 指纹采集

扫描结果的每个测量顶层路径（二进制 real、dataDirs、cleanables、孤儿路径）SHALL 产生一个指纹条目，包含路径、mtime、size 与 isDir；采集 MUST 仅 stat 不递归；结果 MUST 按路径排序（序列化稳定）；不存在的路径 MUST 不产生条目。**此外，当前 PATH 发现目录（与工具发现同一来源，含用户目录补齐）中存在的目录 SHALL 也产生 stat 条目**——新二进制/新工具进入 PATH 目录（目录 mtime 变化）MUST 判为变更。**mtime SHALL 采用纳秒精度（`ModTime().UnixNano()`）**，与健康探测缓存一致。

#### Scenario: 指纹覆盖测量路径

- **WHEN** 某工具含一个二进制文件与一个数据目录
- **THEN** 指纹包含该二进制与数据目录的条目，字段与 `os.Stat` 一致（路径 clean、mtime/size/isDir 对应）；PATH 发现目录另有条目

#### Scenario: 新工具安装自动检测

- **WHEN** 用户向某 PATH 目录安装新工具（如 `npm i -g` 新包、向 `~/.local/bin` 放入新脚本）或新增 PATH 目录
- **THEN** 目录 mtime 变化或条目集合变化，指纹不一致，下次 `ScanIfUnchanged` 执行全量扫描

#### Scenario: 同秒替换可检测

- **WHEN** 某二进制在同一秒内被同大小替换（mtime 纳秒级变化）
- **THEN** 指纹不一致（纳秒精度），执行全量扫描

### Requirement: 指纹比较

两份指纹相等 SHALL 要求：条目数一致、路径集合一致、每条目的 mtime/size/isDir 一致（顺序无关）。条目缺失、mtime 变化或 size 变化 MUST 判为变更。

#### Scenario: 顺序无关的等值

- **WHEN** 两份指纹内容相同但条目顺序不同
- **THEN** 判为相等

#### Scenario: 路径消失判为变更

- **WHEN** 某测量路径从扫描结果中消失（指纹条目缺失）
- **THEN** 判为变更

### Requirement: 变更判定驱动扫描

`ScanIfUnchanged` SHALL：缓存与指纹均存在且指纹一致时直接返回缓存结果（不执行扫描、不写历史）；指纹文件缺失或指纹不一致时执行全量扫描；全量扫描后 MUST 同写缓存与指纹（指纹基于未过滤结果）。**`ScanIfUnchanged` SHALL 向调用方返回是否执行了全量扫描（scanned 标志）**。

#### Scenario: 未变化秒回缓存

- **WHEN** 缓存与指纹存在且指纹一致
- **THEN** 返回缓存结果，无全量 IO、无历史记录，scanned 为 false

#### Scenario: 首次运行保守全量

- **WHEN** 有缓存但无指纹文件
- **THEN** 执行全量扫描并生成指纹，scanned 为 true

#### Scenario: 数据变化自动重扫

- **WHEN** 某数据目录的子项被增删（目录 mtime 变化）、某二进制被替换（mtime/size 变化）或 PATH 发现目录新增二进制
- **THEN** 指纹不一致，执行全量扫描并更新缓存与指纹，scanned 为 true

#### Scenario: 真实扫描记录历史

- **WHEN** 真实扫描发生（GUI 启动变更检测、CLI 非 `--refresh` 自动重扫、`--refresh`、GUI 手动重扫），且结果为未过滤
- **THEN** 记录历史快照；缓存命中不记录

### Requirement: 强制全量路径

`Scan`（GUI 手动"重新扫描"）与 `--refresh` SHALL 始终执行全量扫描，不受指纹影响。

#### Scenario: 手动重扫始终全量

- **WHEN** 用户在 GUI 点击"重新扫描"或 CLI 执行 `scan --refresh`
- **THEN** 无条件全量扫描

### Requirement: 指纹生命周期

指纹文件 SHALL 存放于缓存区（与 `last-scan.json` 并列）；`cache --clear` SHALL 同时清除指纹文件；指纹与缓存不一致时 MUST 保守走全量（安全方向）。

#### Scenario: 清缓存连指纹

- **WHEN** 执行 `cli-analyzer cache --clear`
- **THEN** 缓存与指纹文件均被清除，下次扫描走全量

### Requirement: 已知盲区

文件内容原地修改（不改变父目录 mtime）可能不触发变更检测，SHALL 在文档中说明；强制全量路径提供兜底。**新装工具/新增 PATH 二进制已由 PATH 目录条目覆盖，不再属于盲区**。

#### Scenario: 盲区兜底

- **WHEN** 用户怀疑数据变化未被检测
- **THEN** 使用"重新扫描"/`scan --refresh` 强制全量即可收敛
