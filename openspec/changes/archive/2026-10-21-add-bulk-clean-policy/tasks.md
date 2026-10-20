## 1. 策略与配置

- [x] 1.1 `internal/config/config.go`：新增 `CleanConfig{BulkKinds, BulkExclude []string}`，挂到 `Config.Clean`，默认两空
- [x] 1.2 新增 `internal/cleanpolicy/cleanpolicy.go`：`BulkEligible(kind string, cfg config.CleanConfig) bool` 与 `BulkKinds(cfg) []string`，合并规则 `(内置 ∪ bulkKinds) − bulkExclude`
- [x] 1.3 单测：默认集合等于内置 7 种；追加生效；剔除生效；追加与剔除同时生效；`runtime` 默认不在集合内；未知 Kind 默认不在集合内

## 2. CLI 与 GUI 接入

- [x] 2.1 `internal/cli/clean.go`：删除 `cleanKinds` 硬编码，改调 `cleanpolicy.BulkEligible`
- [x] 2.2 ~~`gui/service.go`：批量选择改调同一函数~~ —— **查证后确认 GUI 没有批量选择功能**：工具视图只能逐项勾选，「全选」只存在于「未认领数据」视图（与 Kind 无关）。因此没有第二份白名单可统一，ADR 0005 里「CLI 与 GUI 各有一份」的原判断已更正
- [x] 2.3 单测覆盖 CLI 侧全部策略分支（`internal/cli/bulkpolicy_test.go` 5 例）；GUI 侧无批量入口，无对应断言

## 3. 文档与文案

- [ ] 3.1 `docs/user-rules.md`：说明自定义 Kind 如何通过 `clean.bulkKinds` 进入批量清理
- [ ] 3.2 `README` / `README.zh-CN`：CLI 段落补 `bulkKinds` / `bulkExclude` 示例
- [ ] 3.3 三语 locale：`--include-data` 的说明文案改为只影响 user 类（不再暗示它影响 Kind 白名单）

## 4. 验证

- [x] 4.1 `gofmt` 干净、`go vet` 通过、`go test ./...` 17 包全绿
- [x] 4.2 实机：规则包声明 `kind: "pack-cache"` → 未配置时 `clean --all` 不选中；配置 `bulkKinds: ["pack-cache"]` 后选中（`TestUserRuleKindIntegratesWithPolicy` + 手工验证）
- [x] 4.3 实机：`bulkExclude: ["cache"]` → `clean --all --dry-run --json` 的 `deleted` 从 37 降到 12，正好移除 25 个 cache 项（与 `clean --list` 的 `cache: 25` 吻合）；恢复配置后回到 37
- [x] 4.4 ~~GUI 冒烟对照~~ —— 同上，GUI 无批量选择入口，无可对照项

## 5. 实施中发现并修复的契约违约

- [x] 5.1 `clean --json` 在**批量选择为空**和**没有任何可处置项**两条早退分支里直接打纯文本（「没有可处置的项目」/「没有可清理项」），违反「`--json` 下 stdout 只含一份 JSON 文档」的契约。脚本会拿到非 JSON 直接解析失败。两条分支均改为输出空 `CleanReport`
- [x] 5.2 新增 `internal/cli/clean_json_test.go`：三个入口（all-empty / all-include-data / list）断言 stdout 必定是带 `kind` 与 `schemaVersion` 的合法 JSON

## 6. 本地 CI 验证（发现真实 CI 会红的问题）

- [x] 6.1 新增 `scripts/verify-ci-local.sh`：逐条执行 `ci.yml` test job 的命令（npm ci / build / test、gofmt、vet、test -cover、契约校验、build），一条命令复现 CI
- [x] 6.2 该脚本抓到 `internal/scanner/attribute.go` 与 `types.go` **未格式化**——这两个文件在之前的提交里就是坏的，真实 CI 的 gofmt 步骤会直接失败。已 `gofmt -w` 修复
- [ ] 6.3 待复核：`go test ./... -cover` 在本地运行时超时未完成（`-cover` 给每个包加插桩，probe/scanner 这类慢包叠加后超过等待上限）。需单独确认是「只是慢」还是真有包挂住
