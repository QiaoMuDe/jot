# Tasks

## Task 1: 后端新增 UpdateAIMessageContent 方法

实现在 `AIService` 中通过消息 ID 更新 `content` 字段的数据库操作。

- [x] SubTask 1.1: 在 [ai_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go) 中新增 `UpdateAIMessageContent(id uint, content string) error` 方法，用 `db.Model(&models.AIMessage{}).Where("id = ?", id).Update("content", content)`
- [x] SubTask 1.2: 在 [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go) 中新增 `UpdateAIMessageContent(id uint, content string) error` 绑定方法，委托 `aiService.UpdateAIMessageContent`

## Task 2: 后端新增 DeleteAIMessagesAfter 方法

实现在 `AIService` 中删除某条消息之后所有消息的数据库操作。

- [x] SubTask 2.1: 在 [ai_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go) 中新增 `DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error)` 方法：
  - 先查询出 `messageID` 的 `created_at`
  - 然后用 `db.Where("session_id = ? AND created_at > ?", sessionID, createdAt).Delete(&models.AIMessage{})` 删除后续消息
  - 返回被删除的记录数
- [x] SubTask 2.2: 在 [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go) 中新增 `DeleteAIMessagesAfter(sessionID uint, messageID uint) (int64, error)` 绑定方法

## Task 3: 前端 — 用户消息增加编辑按钮

在 `createMsgActions()` 函数中为 `role === 'user'` 添加编辑按钮。

- [x] SubTask 3.1: 在 [ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js) 的 `createMsgActions()` 中，为 `role === 'user'` 分支在复制按钮之后添加编辑按钮（铅笔 SVG 图标，`title: '编辑'`）
- [x] SubTask 3.2: 编辑按钮的 SVG 使用 `M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z` 铅笔图标

## Task 4: 前端 — 实现内联编辑模式

核心编辑交互逻辑。

- [x] SubTask 4.1: 新增 `enterEditMode(msgEl, originalContent)` 函数：
  - 获取消息内容 container（`.msg-content`）
  - 保存原内容到 `msgEl.dataset.originalContent`
  - 创建 `<textarea>` 填入原内容，自动聚焦全选
  - 隐藏原内容 div，插入 textarea
  - 创建确认（✓）和取消（✕）按钮容器，插入到 action-buttons 位置
  - 隐藏原有的 action-buttons
- [x] SubTask 4.2: 新增 `confirmEdit(msgEl)` 函数：
  - 从 textarea 读取新内容
  - 空内容校验：触发按钮红色闪烁动画，不保存
  - 与原内容对比，不同则调用 `applyEdit(msgEl, newContent)`，相同则调用 `cancelEdit(msgEl)`
- [x] SubTask 4.3: 新增 `cancelEdit(msgEl)` 函数：删除 textarea 和编辑按钮，恢复 `msg-content` 和原始 action-buttons 显示
- [x] SubTask 4.4: 新增 `applyEdit(msgEl, newContent)` 函数：
  - 更新 `msgEl` 中 `.msg-content > div:first-child` 的文本内容（如果是纯文本）或整体替换（Markdown 渲染的内容）
  - 更新 `_contextMsgContent = newContent`
  - 从 DOM 中删除 `msgEl` 之后的所有 `.ai-msg` 兄弟节点
  - 计算 `msgEl` 在 `chatHistory` 中的索引：删除该索引及之后的所有项
  - 更新 `updateContextSize()`
  - 异步调用后端 `App.UpdateAIMessageContent(msgId, newContent)` 和 `App.DeleteAIMessagesAfter(activeSessionId, msgId)`
  - 调用 `cancelEdit(msgEl)` 恢复正常显示
  - 触发 `startStreaming(false)` 自动开始新的回复

> **注意**：需要给消息气泡 DOM 元素存储对应的数据库 `messageId`。当前 `addMessage()` 不传递 message ID。编辑按钮需要知道哪条消息被编辑。方案：在 `addMessage()` 的 msgEl 上挂 `dataset.msgId`，或使用 `chatHistory` 索引查找。由于编辑确认后要关联回数据库，建议在消息加载/创建时在 DOM 上存储 `data-msg-id` 属性。

> **SubTask 4.5**：修改 `addMessage()` 函数，在 `.ai-msg` 元素上设置 `dataset.msgId` 属性：
> - 在 `switchSession()` 的历史消息加载中（`msgs.forEach`），每个 msg 对象有 `.id` 字段（来自 `AIMessage.ID`），在 `addMessage` 调用时传入
> - 在 `startStreaming()` 的 stream-done 回调中（新建的 assistant 消息），目前没有 msg ID，保存后可以通过 `SaveAIMessages` 返回值获取 ID，或在编辑前通过 `chatHistory` 索引（同等轮次的消息位置一致）
> - **简化方案**：`applyEdit` 中不用数据库 ID，用 `chatHistory` 索引来定位消息。编辑时的 `UpdateAIMessageContent` 可以先查询 session 消息列表找到第 N 条 user 消息的 ID。

- [x] SubTask 4.6: 绑定键盘事件：Ctrl+Enter 确认、Escape 取消
- [x] SubTask 4.7: 流式进行中（`isStreaming`）忽略编辑按钮点击

## Task 5: 前端 — 编辑按钮事件绑定

- [x] SubTask 5.1: 在 `createMsgActions()` 中的编辑按钮 `addEventListener('click', ...)` 调用 `enterEditMode(msgEl.parentElement, content)`（`msgEl.parentElement` 是 `.ai-msg`）

## Task 6: CSS 样式

- [x] SubTask 6.1: 在 [ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css) 中新增编辑模式样式：
  - `.ai-msg-edit-textarea` — textarea 样式（继承字体、背景半透明、边框圆角、最小高度）
  - `.ai-msg-edit-actions` — 确认/取消按钮容器（flex row，gap 4px）
  - `.ai-msg-edit-btn` — 编辑按钮通用样式 (icon button, 透明背景)
  - `.ai-msg-edit-confirm` — 确认按钮（绿色 hover）
  - `.ai-msg-edit-cancel` — 取消按钮（红色 hover）
  - `.ai-msg-edit-error` — 空内容校验时的红色闪烁动画（`@keyframes`）

## Task 7: 验证和测试

- [x] SubTask 7.1: 手动验证编辑流程
- [x] SubTask 7.2: 验证边界情况（空内容、超长内容、流式进行中编辑被忽略）
