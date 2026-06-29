# Tasks

- [x] Task 1: 新增 GORM 模型（AISession + AIMessage）并在 db.go 中注册 AutoMigrate
  - 创建 `internal/models/ai_session.go` — AISession 结构体
  - 创建 `internal/models/ai_message.go` — AIMessage 结构体
  - `database/db.go` 追加 `&models.AISession{}, &models.AIMessage{}` 到 AutoMigrate

- [x] Task 2: 新增后端会话 CRUD + 消息持久化方法
  - `ai_service.go` 新增：
    - `GetAISessions() []AISessionSummary` — 返回会话列表（含最后一条消息摘要）
    - `CreateAISession() uint` — 创建空会话
    - `DeleteAISession(id uint) error` — 删除会话及消息
    - `RenameAISession(id uint, title string) error` — 重命名
    - `LoadAISessionMessages(id uint) []Message` — 加载消息
    - `SaveAIMessages(sessionID uint, messages []Message) error` — 保存一轮对话消息
  - `app.go` 新增对应 7 个绑定方法（含 ClearAISessionMessages）

- [x] Task 3: 前端 HTML + CSS — 会话侧栏布局与样式
  - `index.html`：AI 助手视图改为左右分栏（`.ai-chat-layout`），左侧 `.ai-session-sidebar`，右侧现有 `.ai-chat-content`
  - 侧栏包含：新建按钮、会话列表容器、清空按钮
  - `ai-chat.css`：会话侧栏样式（宽度 220px、分割线、高亮态、删除按钮 hover 显示）

- [x] Task 4: 前端 ai-chat.js — 会话管理逻辑
  - 初始化时调用 `GetAISessions()` 加载侧栏
  - 切换会话 → `LoadAISessionMessages(id)` → 渲染消息列表
  - 新建会话 → `CreateAISession()` → 空消息列表
  - 删除会话 → `DeleteAISession(id)` → 刷新侧栏或切换到最近会话
  - 双击内联编辑 → `RenameAISession(id, title)`
  - 流输出完成后 → `SaveAIMessages(sessionID, messages)`
  - 会话项按 `updatedAt` 降序排列
  - 切回视图时自动恢复上次激活的会话

# Task Dependencies
- Task 1 → Task 2（模型必须先定义）
- Task 2 → Task 3, Task 4（后端 API 按先于前端调用）
