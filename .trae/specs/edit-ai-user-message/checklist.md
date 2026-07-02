# 用户消息编辑功能 — 验证清单

## 后端

- [x] `UpdateAIMessageContent(id, content)` 在 `AIService` 中实现，正确更新数据库 `content` 字段
- [x] `UpdateAIMessageContent` 在 `app.go` 中绑定为 Wails 方法
- [x] `DeleteAIMessagesAfter(sessionID, messageID)` 在 `AIService` 中实现，正确删除 `created_at > message.created_at` 的后续消息
- [x] `DeleteAIMessagesAfter` 在 `app.go` 中绑定为 Wails 方法

## 前端 — 编辑按钮

- [x] 用户消息气泡的 `.action-buttons` 中渲染了编辑按钮（铅笔图标）
- [x] assistant 消息没有编辑按钮
- [x] 流式进行中点击编辑按钮无反应

## 前端 — 编辑模式

- [x] 点击编辑按钮，消息文本原地变为 `<textarea>` 且自动聚焦全选
- [x] textarea 样式正确（继承字体、合适大小、边框圆角）
- [x] 确认按钮（✓）和取消按钮（✕）正确显示
- [x] 编辑模式下原有的 action-buttons 被隐藏
- [x] 按 `Ctrl+Enter` 确认编辑
- [x] 按 `Escape` 取消编辑
- [x] 点击取消按钮恢复原文

## 前端 — 编辑确认后

- [x] 编辑后消息内容在气泡中正确更新
- [x] 该消息之后的所有消息（DOM + chatHistory）被清除
- [x] 后端 `UpdateAIMessageContent` 被调用
- [x] 后端 `DeleteAIMessagesAfter` 被调用
- [x] `_contextMsgContent` 已同步更新
- [x] `updateContextSize()` 已调用
- [x] 编辑确认后自动触发 `startStreaming(false)`

## 前端 — 边界情况

- [x] 编辑内容与原内容相同时仅退出编辑模式，不做任何数据库操作
- [x] 编辑为空内容时被阻止，按钮红色闪烁，编辑模式保持
- [x] 超长内容在 textarea 中正常滚动
- [x] 对话历史加载后（`switchSession`），右键菜单能正确使用编辑后的 `_contextMsgContent`
