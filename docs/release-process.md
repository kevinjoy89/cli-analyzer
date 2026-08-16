# 发布流程（Release Runbook）

> 从「代码就绪」到「GitHub Release 正式发布」的完整操作步骤与检查清单。
> 打包细节（架构、注入、前置条件）见 [packaging.md](packaging.md)。

## 流程总览

```
代码就绪 → 本地全量验证 → 版本号 → commit + tag + push → 等 CI/Release 构建
        → 双语 release notes → 发布（draft → published）→ 验证
```

## 0. 前置确认

- 工作区干净（`git status`），本地与 `origin/main` 同步（先 `git pull`）
- 新功能/修复已按 OpenSpec 流程归档（`openspec/changes/archive/`），specs 已同步
- 决定版本号：`0.3.x` 补丁 / `0.3.x.y` 小补丁（bugfix 常用四位）

## 1. 本地全量验证（提交前 MUST，CI 会原样跑一遍）

```bash
gofmt -l .                    # 必须为空！CI "Check formatting" 步骤会失败
go vet ./...
go test ./... -cover          # 全量（含 i18n parity、孤儿过滤等集成测试）
cd frontend && npm test       # vitest
npx tsc --noEmit              # 类型检查
cd .. && for os in windows linux darwin; do GOOS=$os go build ./...; done  # 三平台编译
```

> **血泪教训**：v0.3.7 曾因本地没跑 `gofmt -l .`，提交后 CI 直接红——5 个文件格式问题。
> 新增/修改 Go 文件后养成习惯：`gofmt -w` 再提交。

## 2. 版本号（单一来源）

`wails.json` 的 `productVersion` 是唯一版本来源（与 tag 保持一致）：

```bash
sed -i '' 's/"productVersion": "[^"]*"/"productVersion": "0.3.7.2"/' wails.json
git add wails.json && git commit -m "chore: bump version to 0.3.7.2"
```

> CI 构建前会用 tag 自动覆写 wails.json（防漂移），但本地提交保持一致是规范。
> 不升级版本号、只发 tag 会导致安装包元数据与 tag 不符（v0.3.4 曾因此产物缺失）。

## 3. 打 tag 并推送

```bash
git tag v0.3.7.2
git -c http.version=HTTP/1.1 push origin main --tags
```

> **网络血泪教训**：直连 GitHub 的 HTTP/2 握手不稳定（`Empty reply from server`），
> 强制 HTTP/1.1 稳定得多。失败时重试即可（`git status -sb` 看 `ahead N`）。
> tag 一旦推送即触发 Release 构建，**无法撤回重建同名 tag**——推之前确认代码状态。

## 4. 等待构建并核对

```bash
gh run list --limit 2        # 应有两条：CI（push main）+ Release（push tag）
```

- **CI**：Test (ubuntu/macos/windows) + Wails GUI build —— 必须全绿
- **Release**：三平台矩阵构建（macos/windows/linux），完成后自动创建 **draft** release

Release 构建约 3~5 分钟。轮询：

```bash
for i in $(seq 1 14); do sleep 30; gh release view v0.3.7.2 2>&1 | head -2 | grep -q "title:" && break; done
```

## 5. 双语 release notes（MUST，勿忘！）

Release 自动创建的是**空 notes 草稿**——必须手写双语说明并正式发布。

格式参照历史版本（`gh release view v0.3.7.1`）：

1. 中文部分：`## CLI Analyzer v<版本> — <一句话主题>`
   - 新功能 / 修复 / 改进 分节，每条列用户可感知的变化
   - 效果量化（如"孤儿 34 → 4 项"）
   - **下载**：三平台产物文件名逐一列出
   - 未签名提示（macOS 右键打开 / Windows 未知发布者）
   - 更新说明（v0.3.0+ 启动时提示）
2. `---` 分隔后附完整 **English** 版（对应翻译）

```bash
gh release edit v0.3.7.2 --notes-file /tmp/release-notes.md --draft=false
```

> **血泪教训**：v0.3.7 曾只推 tag 就以为完成——release 是空 notes 的 draft，
> 用户看不到任何说明。**draft=false（发布）+ notes 文件（双语）缺一不可**。

## 6. 发布后验证

```bash
gh release view v0.3.7.2 | head -8
```

- `draft: false`、`prerelease: false`
- 资产 7 个：`checksums.txt` + 6 个平台产物
- 产物命名与版本一致（`CLI-Analyzer-0.3.7.2-*`）
- 分享链接 `https://github.com/kevinjoy89/cli-analyzer/releases/tag/v<版本>`

## 7. 本地冒烟（MUST：任何触碰 GUI/回收站/更新面板的发布）

发布前在**真机 GUI** 上逐项验证（CI 只跑单测与构建，覆盖不到渲染/弹窗/平台行为）：

- **回收站**：单条删除与清空回收站的确认弹窗（macOS + Windows 各验一遍）——
  必须是应用内样式弹窗（不能用原生 `window.confirm`：macOS WKWebView 不支持、
  Windows WebView2 显示网页风格）；确认后真正删除、取消不删除
- **更新面板**：手动「检查更新」→ 下载中取消（面板关闭、toast 提示）；
  断网模拟下载失败 → **面板必须保留**并显示失败信息 + 「打开 Release 页面」可跳转
- **更新面板**：校验失败路径（面板保留、仅提供 Release 链接）；下载完成路径（立即安装/稍后）
- **孤儿移入回收站**：确认弹窗 → 确认后移入、取消不动
- **清理确认**：可处置项删除（入回收站）与永久删除的确认弹窗样式与行为
- **弹窗层级**：确认弹窗必须盖在其他面板（回收站/首选项/更新）之上，可点可关

```bash
./scripts/build-dmg.sh arm64    # 本地 dmg（create-dmg 收尾 eject 失败时手动
                                # 重命名 rw.<pid>. 前缀产物并清理残留挂载卷）
```

> **血泪教训**：v0.3.7.3 之后发现回收站删除/清空的"确认"用了原生 `window.confirm`——
> macOS 上点击无反应、Windows 上样式不统一；且确认弹窗 z-index 低于回收站面板被盖住。
> 这些只有真机 GUI 冒烟能拦住，单测/构建全绿都覆盖不到。

## 检查清单（发布前逐项勾）

- [ ] `gofmt -l .` 为空
- [ ] `go vet ./...` + `go test ./... -cover` 全绿
- [ ] 前端 `npm test` + `tsc --noEmit` 通过
- [ ] 三平台交叉编译通过
- [ ] `wails.json` productVersion 与 tag 一致
- [ ] 本地 = `origin/main`，无未提交改动
- [ ] CI run 全绿（含 windows-latest）
- [ ] Release run 成功、6 产物 + checksums 齐全
- [ ] **GUI 冒烟通过**（第 7 步清单：回收站确认弹窗/更新失败面板/弹窗层级，双平台）
- [ ] 双语 notes 已写、`draft=false` 已发布
- [ ] 发布链接可访问
