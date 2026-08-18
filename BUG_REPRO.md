# Bug 复现

- Bug 是什么：遥测窗口过滤和合并共享原始读数切片，后续追加会覆盖旧窗口的数据。
- 如何触发：运行 `TestWindowDoesNotRewriteReadings`，先筛选正值，再追加下一批点位。
- 错误信息：`source changed`。
