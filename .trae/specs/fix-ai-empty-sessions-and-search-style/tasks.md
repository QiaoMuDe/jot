# Tasks

- [x] Task 1: 联网搜索下拉菜单样式统一为"更多技能"风格
  - [x] 步骤 1.1: 修改 HTML，将 `<label class="ai-chat-search-source-item">` 改为 `<div class="ai-chat-search-source-item">` 结构，去掉 `style="display:none"`，改用 `.open` class
  - [x] 步骤 1.2: 重写 CSS 样式，使用与 `.ai-chat-skills-dropdown` 一致的背景色、阴影、圆角、过渡动画、菜单项渐入效果
  - [x] 步骤 1.3: 修改 JS，将 `style.display = 'none'/'block'` 改为 `classList.toggle('open')`

- [x] Task 2: 修复 `switchSession` 空会话无法高亮的问题
  - [x] 步骤 2.1: 在 `switchSession` 中，将侧栏高亮更新代码移到空会话判断的 `return` 之前
  - [x] 步骤 2.2: 确保空会话切换时 `updateChatTitle()` 正确更新标题

- [x] Task 3: 数据库瘦身新增清理空 AI 会话逻辑
  - [x] 步骤 3.1: 在 `ai_service.go` 新增 `DeleteEmptyAISessions()` 方法，查询 `ai_sessions` 表中没有关联 `ai_messages` 的记录并删除
  - [x] 步骤 3.2: 在 `app.go` 的 `VacuumDatabase()` 中，VACUUM 之前调用 `DeleteEmptyAISessions()`，将删除数量合并到返回消息中

- [x] Task 4: 会话侧边栏展开时从数据库刷新会话列表
  - [x] 步骤 4.1: 在侧栏折叠/展开按钮的 `click` 事件中，当侧栏从折叠变为展开时，调用 `loadSessionList()` 获取最新列表

# Task Dependencies

- [Task 1] 无依赖
- [Task 2] 无依赖
- [Task 3] 无依赖（后端方法独立）
- [Task 4] 无依赖
