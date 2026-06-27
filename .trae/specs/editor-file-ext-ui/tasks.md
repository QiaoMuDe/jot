# Tasks

- [x] Task 1: 后端新增 `UpdateNoteFileExt` API — 在 `note_service.go` 添加 `UpdateFileExt(id, fileExt)` 方法，在 `app.go` 添加 `UpdateNoteFileExt` Wails 绑定方法
- [x] Task 2: 状态栏后缀显示 — 在 `index.html` 的 `.editor-footer-left` 中添加 `.file-ext` 显示元素；在 `main.js` 的 `openEditor()` 中设置后缀文本
- [x] Task 3: 后缀编辑对话框 HTML + CSS — 在 `index.html` 中添加后缀编辑模态对话框（遮罩 + 面板 + 输入框 + 按钮）；在 `editor.css` 中添加对话框样式
- [x] Task 4: 后缀编辑 JS 交互 — 在 `main.js` 中绑定点击后缀弹出对话框、输入校验、调用 `UpdateNoteFileExt` 保存、取消关闭等逻辑
- [x] Task 5: 构建验证 — 前端 `npx vite build` + 后端 `go build ./...` 编译通过

# Task Dependencies

- [Task 1] 无依赖，可先执行
- [Task 2] 无依赖，可与 Task 1 并行
- [Task 3] 依赖 Task 2（在已有后缀显示基础上添加对话框）
- [Task 4] 依赖 Task 1 和 Task 3（需要后端 API 和对话框 DOM）
- [Task 5] 依赖所有前置任务完成
