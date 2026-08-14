## 1. 生产修复：分隔符无关

- [x] 1.1 `classify.go`：`versionedMatch`/`npmPkgMatch`/`pipxMatch` 用 `splitPath`（`\`→`/` 后切分），重建路径 `filepath.FromSlash`
- [x] 1.2 `attribute.go`：`toolchainVersion` 用 `strings.FieldsFunc` 识别 `/` 与 `\`
- [x] 1.3 验证：TestClassifyVersioned/TestClassifyNpmScoped/TestDeriveOldVersionsRustup 在 Windows 通过（归因修复）

## 2. 测试夹具平台化

- [x] 2.1 classify_test TestUnder：`/a/b` 字面量 → `filepath.Join`
- [x] 2.2 core_test TestSumMaximal/TestFootprintOf：字面量 → `filepath.Join`
- [x] 2.3 core_test pyenv/rustup derive：`t.Setenv("USERPROFILE", home)`（Windows HomeDir）
- [x] 2.4 uninstall fakeRoots + USERPROFILE；残留用例 config 根按平台（XDGConfig/AppData）
- [x] 2.5 cli setupUninstallEnv + USERPROFILE/LOCALAPPDATA/APPDATA 隔离；TestCacheRoundTrip 隔离 LOCALAPPDATA

## 3. unix 专属语义显式跳过

- [x] 3.1 platform_test：TestRootXDGEnv、TestPathDirsAugmentUserDirs skip Windows；TestPathDirsDedup 平台分支
- [x] 3.2 rules_test：TestResolveCleanableGlob skip Windows（XDG cache 根）
- [x] 3.3 updater_test：TestProbeDPKGManaged skip Windows（sh 脚本不可执行）
- [x] 3.4 uninstall_test：TestResolveCommandViaAugmentedPath skip Windows（PATH 增强仅 unix）

## 4. 结构性保障

- [x] 4.1 ci.yml 测试矩阵 + windows-latest
- [x] 4.2 .gitattributes += `*.go text eol=lf`

## 5. 回归验证（本机 Windows）

- [x] 5.1 gofmt：全部改动文件合规（含尾注释对齐）
- [x] 5.2 go vet ./internal/... 通过
- [x] 5.3 go test ./internal/...：platform/scanner/rules/updater/cli 全绿；仅 trash/cleaner/uninstall 的 fsync 用例因本会话沙箱拒绝 FlushFileBuffers 失败（独立探针确认，真实环境通过）
- [x] 5.4 linux/darwin 交叉编译通过（CI 平台不受影响）
