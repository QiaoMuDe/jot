# Tasks

- [x] Task 1: Go 模型层 — AIMessage 和 Message 结构体新增 `IsEmptyResponse` 字段
  - [x] `internal/models/ai_message.go`: 在 AIMessage 结构体末尾添加 `IsEmptyResponse bool` 字段
  - [x] `internal/services/ai_service.go`: 在 Message 结构体末尾添加 `IsEmptyResponse bool` 字段
- [x] Task 2: Go 服务层 — SaveAIMessages 和 LoadAISessionMessages 传递新字段
  - [x] `internal/services/ai_service.go`: `SaveAIMessages` 中将 msg.IsEmptyResponse 赋值给 m.IsEmptyResponse
  - [x] `internal/services/ai_service.go`: `LoadAISessionMessages` 中将 m.IsEmptyResponse 赋值给 result[i].IsEmptyResponse
- [x] Task 3: 前端 JS — 保存时设标记，加载时读标记渲染占位样式
  - [x] `frontend/src/js/ai-chat.js`: `startStreaming` 的 `ai:stream-done` 中，检测空回复时在发送给 saveSessionMessages 的消息对象中加 `is_empty_response: true`
  - [x] `frontend/src/js/ai-chat.js`: `startStreaming` 的 `ai:stream-done` 中，正常非空回复在发送给 saveSessionMessages 的消息对象中加 `is_empty_response: false`
  - [x] `frontend/src/js/ai-chat.js`: `addMessage` 新增第 6 个参数 `isEmptyResponse`，为 true 时渲染 `.ai-msg-empty` 占位结构
  - [x] `frontend/src/js/ai-chat.js`: `switchSession` 加载 assistant 消息时，传递 `msg.is_empty_response` 给 `addMessage`
- [x] Task 4: 验证
  - [x] `wails build` 编译通过
  - [ ] 空回复场景：气泡渲染占位样式，切换会话后样式保持
  - [ ] 正常回复场景：样式不变，不受影响

# Task Dependencies

- Task 1 ← Task 2 ← Task 3（Go 层面完成后前端才能用）
- Task 4 在所有 Task 之后
