# Tasks

- [x] Task 1: 后端新增 DeleteAIMessage 方法
  - [x] 在 `ai_service.go` 新增 `DeleteAIMessage(id uint) error` — 按 id 删除单条消息
  - [x] 在 `app.go` 新增 Wails 绑定 `DeleteAIMessage(id uint) error`
- [x] Task 2: 前端操作栏添加删除按钮 + 删除逻辑
  - [x] 在 `createMsgActions` 中为 user 和 assistant 消息均加入删除按钮（🗑 图标）
  - [x] 流式生成中禁用删除按钮
  - [x] 实现 `handleDeleteMsg(msgEl)` 函数：确认对话框 → 移除 DOM → 清理 chatHistory → 调用后端 `DeleteAIMessage` + `DeleteAIMessagesAfter`
  - [x] 如果 `msgEl.dataset.msgId` 不存在（新建未保存的消息），仅清除 DOM + chatHistory，不调后端
- [x] Task 3: 前端右键菜单添加删除选项
  - [x] 在 `showAiMsgContextMenu` 中，user 和 assistant 均添加「删除」菜单项（分隔线后）
  - [x] 在右键菜单点击处理器添加 `action === 'delete'` 分支，调用 `handleDeleteMsg`
- [x] Task 4: CSS 样式调整
  - [x] 删除按钮与编辑按钮风格一致（图标大小、hover 颜色使用红色警告色）

# Task Dependencies
- Task 2 和 Task 3 依赖 Task 1（后端方法存在才能调用）
- Task 3 依赖 Task 2 中的 `handleDeleteMsg` 函数
