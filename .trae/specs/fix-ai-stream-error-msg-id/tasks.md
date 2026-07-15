# Tasks

## 任务依赖关系

- [Task 1] 无依赖（独立后端改造）
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 1]

---

- [x] Task 1: 后端提取独立保存消息方法 + 新增 SaveAIMessage Wails 导出
  - [x] 在 `app.go` 中已存在的 `aiService.SaveAIMessage`（internal/services/ai_service.go 第 703 行）基础上，新增 `SaveAIMessage` Wails 导出方法
  - [x] 修改 `CallAIStream` 签名，新增 `userMsgID uint` 参数，内部跳过保存用户消息的步骤
  - [x] `CallAIStreamRegenerate` 无需改动（已不创建用户消息）

- [x] Task 2: 前端 onSend 提前获取 msgId + 错误处理改用通知
  - [x] `onSend()` 中先调 `SaveAIMessage` 保存用户消息，拿到 `userMsgId`
  - [x] `addMessage(text, 'user', ..., userMsgId)` 带上 ID
  - [x] `startStreaming(text, userMsgId)` 传入 ID
  - [x] `startStreaming` 签名从 `(userText, isRegenerate)` 改为 `(userText, isRegenerate, userMsgID)`
  - [x] 所有调用 `startStreaming` 的地方适配新签名（`onSend`、`handleRegenerate`、`handleResend`、`confirmEdit`）
  - [x] `ai:stream-error` 处理器和 catch 块中移除 `addErrorMessage()` 调用，改用 `window.showNotification`
  - [x] `handleResend` 也改为先调 `SaveAIMessage` 拿到 ID 再创建气泡和流式
  - [x] `CallAIStream(myGen, ..., userMsgID)` 传入 `userMsgID` 参数

- [x] Task 3: 回归验证
  - [x] Go 编译通过
  - [x] 前端 JS 无诊断错误
  - [x] Wails JS binding 已重新生成，包含 `SaveAIMessage` 和更新后的 `CallAIStream` 签名
