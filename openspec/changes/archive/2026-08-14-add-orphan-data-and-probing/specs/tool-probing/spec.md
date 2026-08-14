## Purpose

对扫描出的 CLI 工具后台探测真实版本与一句话描述（`--version`/`-V`/`--help`），填充版本列，带超时、缓存与编码处理，失败降级且不阻塞扫描。

## ADDED Requirements

### Requirement: 探测编排

对扫描出的工具 SHALL 按顺序尝试 `--version`、`-V`、`--help`，取首个成功输出；每个命令探测超时 MUST 上限 3 秒，超时后 MUST 终止进程组；探测 MUST 后台并行执行，不得阻塞扫描主流程与界面渲染。

#### Scenario: 探测填充版本

WHEN 工具 `uv` 存在且 `uv --version` 输出 `uv 0.4.10`

THEN 该工具的版本字段显示为 `0.4.10`

#### Scenario: 探测超时不阻塞

WHEN 某工具执行 `--version` 超过 3 秒未返回

THEN 该命令被终止，该工具版本保持 `—`，其余工具探测不受影响

### Requirement: 结果缓存

探测结果 MUST 以二进制（real path + size + mtime）为缓存键；文件未变化时重扫 SHALL 直接复用缓存结果，不重复执行命令；缓存 MUST 存放于可清理缓存区，与数据区语义分离。

#### Scenario: 缓存命中

WHEN 扫描两次且某工具二进制未发生变化

THEN 第二次扫描不重新执行探测命令，直接使用首次结果

#### Scenario: 二进制变化触发重探

WHEN 工具二进制内容或大小发生变化

THEN 重新探测并更新缓存

### Requirement: 失败降级

探测失败、超时或输出不可解析时，该工具版本 SHALL 保持 `—`，MUST NOT 报错、阻塞或影响其他工具。

#### Scenario: 探测失败保持占位

WHEN 某工具探测失败（非零退出或无有效输出）

THEN 该工具版本显示 `—`，扫描无错误提示

### Requirement: Windows 输出编码

Windows 上探测输出可能为系统代码页（如 GBK）；非 UTF-8 输出 MUST 转码或剥离非打印字符，展示 MUST NOT 出现乱码。

#### Scenario: GBK 输出转码

WHEN Windows 上某工具 `--version` 输出为 GBK 编码

THEN 界面展示的版本/描述为可读文本，无乱码

### Requirement: 输出契约

探测得到的版本与描述 SHALL 进入扫描结果 JSON 契约：版本 MUST 填充工具 `version` 字段；描述可作为可选字段。探测字段 MUST 仅增强既有契约，不破坏现有键结构。

#### Scenario: JSON 契约包含探测版本

WHEN 执行 `cli-analyzer scan --json` 且某工具探测成功

THEN 该工具 JSON 对象的版本字段为探测到的版本，既有键结构不变
