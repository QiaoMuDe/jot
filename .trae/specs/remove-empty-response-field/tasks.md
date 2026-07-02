# Tasks

- [x] Task 1: 后端 — 移除 `IsEmptyResponse` 字段及相关引用
  - [x] 1.1 从 `internal/models/ai_message.go` 移除 `IsEmptyResponse` 字段
  - [x] 1.2 从 `internal/services/ai_service.go` 的 `Message` 结构体移除 `IsEmptyResponse` 字段
  - [x] 1.3 从 `internal/services/ai_service.go` 的 `LoadAISessionMessages` 中移除 `IsEmptyResponse` 映射
  - [x] 1.4 从 `internal/services/ai_service.go` 的 `SaveAIMessages` 中移除 `IsEmptyResponse` 赋值

- [x] Task 2: 后端 — 添加数据库迁移脚本移除 `is_empty_response` 列
  - [x] 2.1 在 `internal/database/db.go` 的 `AutoMigrate` 后调用 `Migrator().DropColumn()`

- [x] Task 3: 前端 — 移除 TypeScript 模型中的 `is_empty_response`
  - [x] 3.1 从 `frontend/wailsjs/go/models.ts` 的 `Message` 类中移除 `is_empty_response` 属性及相关序列化

- [x] Task 4: 前端 — 修改空回复处理逻辑（ai-chat.js）
  - [x] 4.1 在流式完成处移除 `isEmptyMsg` 变量和占位文本替换逻辑，改为调用 `showNotification`
  - [x] 4.2 移除 `saveSessionMessages` 调用中的 `is_empty_response` 字段
  - [x] 4.3 移除 `addMessage` 函数中的 `isEmptyResponse` 参数及相关渲染逻辑
  - [x] 4.4 移除历史消息加载中的 `msg.is_empty_response` 引用

- [x] Task 5: 前端 — 移除 CSS 空回复占位样式
  - [x] 5.1 从 `frontend/src/css/components/ai-chat.css` 移除 `.ai-msg-empty` 相关样式
  - [x] 5.2 移除 `.ai-msg-assistant:has(.ai-msg-empty)` 相关样式

# Task Dependencies

- Task 1, 2 独立（可并行）
- Task 3 依赖 Task 1（接口对齐）
- Task 4 依赖 Task 1（接口对齐）
- Task 5 独立（可与其他任务并行）
