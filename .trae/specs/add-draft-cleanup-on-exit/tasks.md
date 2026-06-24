# Tasks

- [x] Task 1: 修改 `handleAppExit()` 添加草稿清除逻辑
  - 在 `save` 分支中，`saveEditorContent()` 执行完毕后，若为新建笔记（`!state.editingNoteId`），调用 `window.go.main.App.ClearDraft()`
  - 在 `discard` 分支中，若为新建笔记（`!state.editingNoteId`），调用 `window.go.main.App.ClearDraft()`
  - 保持 `cancel` 分支不变

- [x] Task 2: 更新 `AGENTS.md` — 已实现列表 + 关键记忆点
