## ADDED Requirements

### Requirement: 形态占用小计

对同时具备桌面端与 CLI 形态的 AI 编目条目，系统 SHALL 在工具详情页展示「形态占用」小计：桌面端（应用包 + 桌面端数据目录）与 CLI（dotdir、npm 安装根、XDG 目录）各自合计。小计 SHALL 按极大路径口径计算（父目录与子目录并存时不重复计数），两段小计之和 SHALL 等于该工具全部归因目录的合计；独立二进制文件与不在归因目录中的仅清理项（如安装暂存目录）MAY 不计入小计。纯单形态条目（仅 CLI 或仅桌面端）SHALL 照常展示小计（另一形态为 0）。

#### Scenario: 双形态条目分形态小计

WHEN 用户打开 opencode 详情页（CLI 数据 `~/.local/share/opencode` 8G、桌面端数据 `ai.opencode.desktop` 1.1G、CLI 安装根 `.opencode` 347M、桌面端应用包 OpenCode.app）

THEN 形态占用小计展示桌面端合计（应用包 + 桌面端数据）与 CLI 合计（CLI 数据 + CLI 安装根），两者相加等于总占用

#### Scenario: 应用包自动归为桌面端

WHEN 归因 OpenCode.app（.app 结尾的 install 目录）

THEN 该目录按桌面端形态计入小计，无需编目显式标注

#### Scenario: 单形态条目照常展示

WHEN 用户打开 claude 详情页（仅 CLI 形态）

THEN 形态占用小计展示 CLI 合计与桌面端 0
