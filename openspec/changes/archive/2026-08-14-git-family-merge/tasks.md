## 1. exe 发现环节 GUI 判定收紧

- [x] 1.1 `importsGUI` 从单库命中改为 ≥2 个 GUI 导入库（user32 单库是 Git for Windows 控制台工具的合法导入）
- [x] 1.2 测试：writePE 支持多 DLL 夹具（名字区移到描述符之后避免重叠）；user32 单库 → 保留（git 场景）、user32+gdi32 → 过滤（Xshell 场景）

## 2. git 家族归并

- [x] 2.1 `gitFamily` 名称表 + `gitFamilyRoot`（置于 brew 等具体来源之后）
- [x] 2.2 `gitInstallRoot`：目录含 git.exe/git 时作为安装根
- [x] 2.3 `probeOrder`：git 家族把 git 排到 Binaries[0]
- [x] 2.4 测试：TestClassifyGitFamily（家族命中/安装根/非家族独立/brew 独立）、TestNodejsProbeOrder 扩展 git 用例、旧断言更新（git.exe 现归 git 家族）

## 3. nodejs 辅助脚本并入

- [x] 3.1 nodejsFamily += install_tools / nodevars
- [x] 3.2 TestClassifyNodejsFamily 扩充

## 4. 回归与交付

- [x] 4.1 gofmt + go vet + go test ./internal/platform/ ./internal/scanner/（全绿）+ linux/darwin 交叉编译
- [x] 4.2 端到端：Git for Windows cmd/ 在 PATH 时，扫描输出一行 `git`（8 个二进制合并，含 ~/.gitconfig 与 Git\cmd 安装占用）；工具总数 15 → 12
- [x] 4.3 重建 NSIS 安装器 + 便携 zip（含本次修复），更新 SHA256
