# Bug 复现

- Bug 是什么：通知状态机缺少 retrying 到 sent 的转换，成功重试回写旧状态且活动查询漏掉 retrying。
- 如何触发：运行 `TestSuccessfulRetryReachesSent`，验证重试成功的终态和活动列表。
- 错误信息：`transition denied` 或 `wrong final state`。
