# Tasks

- [x] Task 1: 后端数据模型 — 新增 `RecallNotebookIDs` 字段到 `AISessionConfig` 和 `SessionConfig`，更新 CRUD 函数和默认配置创建
  - `internal/models/ai_session_config.go` +1 行
  - `internal/services/ai_service.go` 同步 `SessionConfig`、`SaveSessionConfig`、`LoadSessionConfig`、`CreateDefaultSessionConfig`

- [x] Task 2: 后端搜索过滤 — `SearchFull` 加 notebook ID 过滤参数，`CardRecallSearch` 接收并传递 notebookIDs
  - `internal/services/note_service.go` 修改 `SearchFull`
  - `internal/services/recall_service.go` 修改 `CardRecallSearch`

- [x] Task 3: Wails 绑定层 — `CallAIStream` 新增 `recallNotebookIDs []uint` 参数，从配置读取后传入 `CardRecallSearch`
  - `app.go` 修改 `CallAIStream` 签名和内部调用

- [x] Task 4: 前端 HTML — 在 AI 聊天工具栏添加笔记本选择菜单结构（参考联网搜索菜单）
  - `frontend/index.html` 添加 `.ai-chat-recall-wrap` 包裹容器和下拉菜单

- [x] Task 5: 前端 CSS — 笔记本选择菜单样式（参考联网搜索菜单，复用或扩展）
  - `frontend/src/css/components/ai-chat.css` 添加菜单相关样式

- [x] Task 6: 前端 JS 交互 — 菜单打开/关闭、checkbox 选中逻辑、全选/全取消、配置保存、设置页联动
  - `frontend/src/js/ai-chat.js` 更新保存配置、加载配置、发送消息、设置页同步等

## Task Dependencies
- Task 4（HTML）和 Task 6（JS）依赖 Task 1-3（后端必须先支持新字段的读写）
- Task 5（CSS）无依赖，可与其他任务并行
- Task 6 依赖 Task 4（先有 HTML 元素才能绑定 JS 事件）
