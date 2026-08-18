# Bug 复现

- Bug 是什么：通知仓储错误包装丢失 sentinel 链，缺失通知被当成 500 并进入三次重试。
- 如何触发：运行 `TestMissingNotificationIsNotRetried`，让仓储返回 `ErrMissing`。
- 错误信息：`status=500 attempts=3`。
