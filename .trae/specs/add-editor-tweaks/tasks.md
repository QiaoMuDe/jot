# Tasks

- [ ] Task 1: Ctrl+L 快捷键切换编辑器模式
  - 在 `handleKeyboardNavigation` 中添加 `Ctrl+L` 检测
  - 编辑器未打开时不响应（`return`）
  - 调用 `switchEditorMode()` 切换 `edit` ↔ `preview`
- [ ] Task 2: 新建笔记默认时间标题
  - 在 `createNewNote()` 中生成 `YYYY-MM-DD HH:mm ☺️` 格式时间标题字符串
  - 填入 `els.editorNoteTitle.value`
  - 确保可通过后端接口正常创建
- [ ] Task 3: 置顶不更新 `updated_at`
  - 修改 `TogglePin` 方法：使用 `UpdateColumn` 或 `Omit("UpdatedAt")` 仅更新 `is_pinned` 字段

## Task Dependencies
- 三个任务完全独立，可并行处理

## Verification
- 每个任务完成后运行 `npx vite build` 确保前端构建通过
- Task 3 额外运行 `golangci-lint run ./...` 确保后端 lint 通过
