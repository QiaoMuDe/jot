# Tasks

- [x] Task 1: 修改 `startAutoSave()` 跳过编辑模式的自动保存
  - 在 `startAutoSave()` 回调函数开头，检查 `state.editingNoteId` 不为 null 时直接 return（不执行保存逻辑）
  - 新建笔记（`state.editingNoteId` 为 null）的草稿保存逻辑不变
  - 编辑模式下 `autoSaveTimer` 仍然会启动（3s 后回调执行），但回调内检测到编辑模式直接返回，不调 `UpdateNote()` 也不更新指示器
