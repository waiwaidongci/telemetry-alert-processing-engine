# Bug 复现

- Bug 是什么：默认路由配置没有初始化 map，禁用校验时又把 typed-nil 校验器当成有效校验器。
- 如何触发：运行 `TestDefaultsAreUsable`，使用空配置注册路由并提交空路由值。
- 错误信息：`default routes panic` 或空路由被接受。
