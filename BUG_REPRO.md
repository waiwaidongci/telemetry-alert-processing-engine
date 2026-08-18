# Bug 复现

- Bug 是什么：告警快照和缓存直接暴露仓库内部 map，更新仓库或修改缓存结果会污染历史统计。
- 如何触发：运行 `TestSnapshotsRemainStable`，读取快照后更新仓库并改写缓存返回值。
- 错误信息：`snapshot changed` 或 `cache leaked mutation`。
