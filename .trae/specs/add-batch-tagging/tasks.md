# Tasks

- [x] Task 1: 后端新增 `BatchRemoveTagFromNotes` 方法
  - 在 `tag_service.go` 新增 `BatchRemoveTagFromNotes(noteIDs []uint, tagID uint) error`
  - 在 `app.go` 新增绑定方法 `BatchRemoveTagFromNotes(noteIDs []uint, tagID uint) error`

- [x] Task 2: 批量操作栏新增按钮
  - `index.html` 批量操作栏新增「批量添加标签」和「批量移除标签」按钮
  - 在 `main.js` 中新增 `batchAddTag()` 和 `batchRemoveTag()` 函数
  - 在 `init()` 中注册按钮事件绑定
  - `style.css` 新增标签选择弹窗样式（~30 行）

# Task Dependencies
- [Task 2] depends on [Task 1]
