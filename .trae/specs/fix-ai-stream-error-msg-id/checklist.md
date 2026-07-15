# Checklist

- [x] 后端 `SaveAIMessage` 方法正确保存在 DB 并返回 ID
- [x] `CallAIStream` 参数新增 `userMsgID`，内部跳过用户消息保存
- [x] `CallAIStreamRegenerate` 不再自建用户消息
- [x] 前端 `onSend` 先调 `SaveAIMessage` 拿到 ID 再创建消息气泡
- [x] 用户消息气泡 DOM 在发送后即带有 `data-msgId`
- [x] 流式出错时 `addErrorMessage` 不再被调用
- [x] 流式出错时仅通过 `showNotification` 通知用户
- [x] 出错后用户消息右键 → 重新发送正常工作
- [x] 出错后用户消息右键 → 删除正常工作
- [x] 重新发送后新流式正常完成
- [x] 切换会话后历史消息加载正常、右键正常
- [x] 正常流式完成场景无退化
