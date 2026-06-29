# Tasks

- [x] Task 1: 停止生成 — 后端支持
  - [x] `internal/services/ai_service.go`: `CallAIStream` 新增 `ctx context.Context` 参数，在 SSE 读取循环中检查 `ctx.Done()`
  - [x] `app.go`: 在 `App` 结构中添加 `aiStreamCancel context.CancelFunc` 字段；`CallAIStream` 创建可取消的 context 并保存 cancel 函数；新增 `CancelAIStream()` 绑定方法调用 cancel
- [x] Task 2: 停止生成 — 前端 UI 与逻辑
  - [x] `frontend/index.html`: 在发送按钮旁或替换发送按钮，添加停止按钮元素（初始隐藏）
  - [x] `frontend/src/js/ai-chat.js`: 流式开始时显示停止按钮、隐藏发送按钮；点击停止按钮调用 `window.go.main.App.CancelAIStream()`；停止后保留已接收内容，恢复发送按钮
- [x] Task 3: 对话搜索 — 搜索输入框
  - [x] `frontend/index.html`: 在会话侧栏头部（`ai-session-sidebar-header`）新增搜索输入框
  - [x] `frontend/src/css/components/ai-chat.css`: 新增搜索输入框样式
- [x] Task 4: 对话搜索 — 前端过滤逻辑
  - [x] `frontend/src/js/ai-chat.js`: 新增 `sessionSearchQuery` 变量；监听搜索输入框的 `input` 事件；在 `renderSessionList()` 中根据搜索词过滤会话列表（标题匹配，忽略大小写）；匹配关键词高亮

# Task Dependencies

- Task 2 依赖 Task 1（前端需要后端的取消绑定）
- Task 3 和 Task 4 无依赖，可并行
- Task 4 依赖 Task 3（需要输入框元素）