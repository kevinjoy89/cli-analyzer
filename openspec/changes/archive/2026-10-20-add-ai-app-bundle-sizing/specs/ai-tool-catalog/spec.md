## ADDED Requirements

### Requirement: 桌面端应用包计入安装占用

对声明了桌面端宿主应用（Apps）的 AI 编目条目，系统 SHALL 将实际存在的应用包目录（macOS 的 .app 包）作为 Kind install 的归因目录纳入测量：安装占用列 SHALL 包含 CLI 安装根与桌面端应用包之和，总占用 SHALL 相应包含应用包。应用包不存在时（桌面端未安装）SHALL 不归因——与「已卸载」标注口径一致。非 macOS 平台的桌面端应用包测量不属于本能力。

#### Scenario: 已安装桌面端的应用包计入

WHEN WorkBuddy.app 实际存在于 /Applications 且 WorkBuddy 条目声明 Apps 包含 WorkBuddy

THEN workbuddy 的安装占用包含该应用包大小，总占用相应包含，不再显示 0 B / —

#### Scenario: 已卸载桌面端不归因

WHEN MarsCode 应用包不存在且 marscode 条目声明 Apps 包含 MarsCode

THEN marscode 不产生应用包归因目录，安装占用维持 —（数据残留照常按语义展示）

#### Scenario: CLI 安装根与桌面端应用包并存

WHEN dsh 既有 npm 安装根（node_modules/@deepseek-ai/dsh）又有 DeepSeek Harness.app

THEN 安装占用为两者之和（不相交路径，各自计入）
