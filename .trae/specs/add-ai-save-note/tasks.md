# Tasks

- [ ] Task 1: 后端 `SaveAIMessageAsNote` 绑定 — `app.go` 新增绑定函数
  - 函数签名：`func (a *App) SaveAIMessageAsNote(content string) (*models.Note, error)`
  - 空内容校验返回错误
  - 自动生成标题：取第一行/首段，截断到 50 字符加 "..."
  - 调用 `a.noteService.Create(title, content, ".md")` 创建笔记
  - 返回创建后的笔记对象

- [ ] Task 2: 前端保存按钮 — `ai-chat.js` 中 `createMsgActions` 为 assistant 增加"保存为笔记"按钮
  - SVG 保存图标
  - 按钮位于复制和重新生成之间
  - 点击事件调用 `window.go.main.App.SaveAIMessageAsNote(content)`
  - 成功后显示 `window.showNotification?.('笔记已创建', 'success')`，通知中附带"查看"按钮
  - 失败时显示 `window.showNotification?.('保存失败: ...', 'error')`
  - "查看"按钮点击后：`window.switchView('grid')` 并通过 `openEditor(note.id, true)` 打开该笔记

# Task Dependencies
- Task 1 和 Task 2 无依赖，可并行实施
