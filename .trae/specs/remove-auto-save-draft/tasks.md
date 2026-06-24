# Tasks

- [x] Task 1: 删除后端草稿相关代码
  - 删除 `internal/models/draft.go`（Draft 模型）
  - 删除 `internal/services/draft_service.go`（DraftService）
  - 修改 `internal/database/db.go`：从 `AutoMigrate` 中去掉 `&models.Draft{}`
  - 修改 `app.go`：删除 `draftService` 字段/初始化/三个绑定方法
- [x] Task 2: 删除前端自动保存功能（main.js）
  - 删除 `startAutoSave()` 函数
  - 删除 `autoSaveTimer` 全局变量
  - 删除 `onEditorInput()` 中的 `startAutoSave()` 调用
  - 删除 `openEditor()` 中 autoSaveIndicator 重置和 autoSaveTimer 清理代码
  - 删除 `closeEditor()` 中 autoSaveTimer 清理代码
- [x] Task 3: 删除前端草稿恢复及 ClearDraft 
  - 删除 `loadNotes()` 中的草稿检测弹窗（if (window.go && ... GetDraft) 块）
  - 删除 `createNote()` 中的 `ClearDraft()` 调用
  - 删除 `saveEditorContent()` 中的 `SaveDraft` 分支（仅保留 UpdateNote）
  - 删除 `handleAppExit()` 中的 `ClearDraft` 调用（保存后/放弃时各一处）
  - 删除取消按钮中的 `ClearDraft()` 调用
  - 删除蒙层点击/放弃中的 `ClearDraft()` 调用
- [ ] Task 4: 删除前端 HTML/CSS 草稿指示器
  - 删除 `els.autoSaveIndicator` 定义
  - 删除 index.html 中 `#autoSaveIndicator` 元素
  - 删除 style.css 中 `.auto-save-indicator` 相关规则

## Task Dependencies

- Task 1 无依赖
- Task 2 无依赖
- Task 3 无依赖
- Task 4 无依赖
