# Tasks

- [x] Task 1: 后端新增 `ReplaceAISessionMessages` — AIService 层方法（事务内清空 + 批量写入 + auto-title）
  - [x] Step 1.1: 在 `ai_service.go` 中实现 `ReplaceAISessionMessages(sessionID uint, messages []Message) error`，使用 `gorm.DB.Transaction` 包裹 Clear + Create 逻辑，复用 `SaveAIMessages` 的 auto-title 代码片段
  - [x] Step 1.2: 在 `app.go` 中实现对应的 Wails 绑定方法 `ReplaceAISessionMessages(sessionID uint, messages []services.Message) error`，含日志

- [x] Task 2: 前端 4 处操作替换为单次调用
  - [x] Step 2.1: `applyEdit()` — 将 L3348-3350 的 Clear+Save 替换为 `App.ReplaceAISessionMessages`
  - [x] Step 2.2: `handleDeleteMsg()` — 将 L3399-3401 替换
  - [x] Step 2.3: `handleRegenerate()` — 将 L3438-3440 替换
  - [x] Step 2.4: `handleResend()` — 将 L3466-3468 替换

- [x] Task 3: 验证与清理
  - [x] Step 3.1: 构建项目确认无编译错误
  - [x] Step 3.2: 人工确认前端所有 `ClearAISessionMessages` 调用仅剩「清空会话」按钮一处（L388）+ 4 处 empty chatHistory 兜底

# Task Dependencies

- Task 2 依赖 Task 1
