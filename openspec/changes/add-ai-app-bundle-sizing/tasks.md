## 1. 应用包测量

- [ ] 1.1 `platform.AppBundlePaths(apps)`：darwin 返回实际存在的 .app 路径（/Applications、~/Applications、/System/Applications），其他平台返回空
- [ ] 1.2 `scanner`：AI 条目（category 非空且有 Apps）在 metadata 阶段为存在的应用包追加 Kind install 归因 + measure；包级变量注入便于测试
- [ ] 1.3 scanner 测试：注入临时 .app 路径 → 安装占用包含应用包、footprint 相应增长；应用缺失时不归因

## 2. 验证

- [ ] 2.1 go test 全绿；实机重扫：workbuddy/zcode/code 等安装占用 = 应用包实际大小，trae-cn/marscode 维持 —
