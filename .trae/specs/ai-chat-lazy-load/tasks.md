# Tasks

- [x] Task 1: 后端新增分页/截断/Token 查询方法
  - [x] Step 1.1: 在 `ai_service.go` 新增 `LoadAISessionMessagesPaginated(sessionID, limit, beforeID)` — 游标分页，基于 ID 降序取 N 条后反转返回
  - [x] Step 1.2: 在 `ai_service.go` 新增 `TruncateAISessionAtMessage(sessionID, msgID)` — 删除该消息及其之后的所有消息（事务内：先查 `created_at` → 删除 `>=` 该时间戳的所有消息）
  - [x] Step 1.3: 在 `ai_service.go` 新增 `TruncateAISessionAfterMessage(sessionID, msgID)` — 删除该消息之后的所有消息（事务内：先查 `created_at` → 删除 `>` 该时间戳的所有消息）
  - [x] Step 1.4: 在 `ai_service.go` 新增 `GetSessionContextTokens(sessionID)` — 返回 `ai_sessions.context_tokens`
  - [x] Step 1.5: 在 `app.go` 为上述 4 个方法添加 Wails 绑定（含日志）

- [x] Task 2: 重构 `CallAIStream` 签名——后端自主加载历史
  - [x] Step 2.1: 修改 `app.go` 中 `CallAIStream` 签名，移除 `messages []services.Message` 参数，改为 `sessionID uint, userText string, ...` 元数据参数
  - [x] Step 2.2: 函数内部先调用 `SaveAIMessage` 保存用户消息到 DB 并获取 userMsgID
  - [x] Step 2.3: 再调用 `LoadAISessionMessages` 从 DB 加载全部历史消息
  - [x] Step 2.4: 构建 8 步上下文、调用 AI、保存 assistant 消息的逻辑保持不变
  - [x] Step 2.5: `ai:stream-done` 事件扩展，发射 `userMsgID` 和 `assistantMsgID`（新入库消息的 ID）
  - [x] Step 2.6: 新增 `CallAIStreamRegenerate(sessionID, ...metadata)` — 不保存用户消息，直接从 DB 加载历史 + 最后一条 user 消息作为输入

- [x] Task 3: 前端 `chatHistory` 降级 + `onSend` 重构
  - [x] Step 3.1: `chatHistory` 声明改为仅渲染缓冲区，注释说明不再发送给后端
  - [x] Step 3.2: `onSend()` 中移除 `chatHistory.push` 和 `startStreaming` 传参，改为直接调用 `startStreaming(text, false)`
  - [x] Step 3.3: `startStreaming()` 中移除 `chatHistory` 参数，函数内改为接收 `userText`，根据 `isRegenerate` 分支调用 `CallAIStream`/`CallAIStreamRegenerate`
  - [x] Step 3.4: `stream-done` 回调中收到 `userMsgID`/`assistantMsgID` 后挂载到 DOM 上（`data-msg-id`）
  - [x] Step 3.5: `stream-done` 回调中更新 `chatHistory` 的逻辑改为仅更新渲染缓冲区（push 新消息到本地数组）
  - [x] Step 3.6: 删除 `_pendingTokenSync` 变量及相关逻辑

- [x] Task 4: 前端 `switchSession` 重构为分页加载
  - [x] Step 4.1: `switchSession()` 调用 `LoadAISessionMessagesPaginated(sessionID, limit=6, beforeID=0)` 代替 `LoadAISessionMessages`
  - [x] Step 4.2: 消息 DOM 渲染时每个消息气泡挂载 `data-msg-id` 属性
  - [x] Step 4.3: 新增滚动到顶部加载更多逻辑（`_loadingMore` 防重复，`_oldestMsgId` 追踪分页游标）
  - [x] Step 4.4: 加载更早消息时 prepend 到消息列表，`chatHistory.unshift()` 更新缓冲区

- [x] Task 5: 前端编辑/删除/重发/再生重构为 msgID 驱动
  - [x] Step 5.1: `applyEdit()` — 从 `msgEl.dataset.msgId` 读取 msgID → 调 `TruncateAISessionAfterMessage` → 调 `CallAIStream`
  - [x] Step 5.2: `handleDeleteMsg()` — 从 `msgEl.dataset.msgId` 读取 msgID → 调 `TruncateAISessionAtMessage` → 重新加载会话
  - [x] Step 5.3: `handleRegenerate()` — 找前一个用户消息的 `data-msg-id` → 调 `TruncateAISessionAfterMessage` → 调 `CallAIStreamRegenerate`
  - [x] Step 5.4: `handleResend()` — 从 `msgEl.dataset.msgId` 读取 msgID → 调 `TruncateAISessionAtMessage` → 调 `CallAIStream`
  - [x] Step 5.5: 移除 4 个函数中所有 `chatHistory.splice` 和 `ReplaceAISessionMessages` 调用

- [x] Task 6: 前端 `updateContextSize` 重构
  - [x] Step 6.1: `updateContextSize()` 从后端 `GetSessionContextTokens` 读取 tokens
  - [x] Step 6.2: `stream-done` 回调中移除旧的 token 更新逻辑，仅调用 `updateContextSize()`
  - [x] Step 6.3: 删除 `updateContextSize` 中遍历 `chatHistory` 的逻辑

- [x] Task 7: 构建验证 + 清理
  - [x] Step 7.1: 构建项目确认无编译错误（Go + 前端均通过）
  - [x] Step 7.2: 确认 `chatHistory` 不再出现在 `CallAIStream` 调用中
  - [x] Step 7.3: 确认 `_pendingTokenSync` 已删除
  - [x] Step 7.4: 确认 `estimateTokens` 已删除或不再引用