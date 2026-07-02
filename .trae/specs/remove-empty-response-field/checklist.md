# 验证清单

- [x] `internal/models/ai_message.go` 中 `AIMessage` 结构体不再包含 `IsEmptyResponse` 字段
- [x] `internal/services/ai_service.go` 中 `Message` DTO 不再包含 `IsEmptyResponse` 字段
- [x] `internal/services/ai_service.go` 中 `LoadAISessionMessages` 不再引用 `IsEmptyResponse`
- [x] `internal/services/ai_service.go` 中 `SaveAIMessages` 不再引用 `IsEmptyResponse`
- [x] `internal/database/db.go` 在 `AutoMigrate` 后调用 `Migrator().DropColumn()` 迁移旧列
- [x] `frontend/wailsjs/go/models.ts` 中 `Message` 类不再包含 `is_empty_response`
- [x] `frontend/src/js/ai-chat.js` 流式完成处检测空内容时调用 `showNotification` 而不是保存占位消息
- [x] `frontend/src/js/ai-chat.js` 中 `saveSessionMessages` 调用不再传递 `is_empty_response`
- [x] `frontend/src/js/ai-chat.js` 中 `addMessage` 函数不再有 `isEmptyResponse` 参数
- [x] `frontend/src/js/ai-chat.js` 中 `addMessage` 函数体内不再有空回复占位渲染逻辑
- [x] `frontend/src/js/ai-chat.js` 中历史消息加载不再引用 `msg.is_empty_response`
- [x] `frontend/src/css/components/ai-chat.css` 中不再包含 `.ai-msg-empty` 样式和 `:has(.ai-msg-empty)` 选择器
- [x] 后端代码编译无错误
