# Tasks

- [ ] Task 1: 后端模型 — `models/note.go` 移除 `NoteType` 字段
- [ ] Task 2: 后端服务层 — `note_service.go` 签名变更：`Create()` / `Update()` / `CreateWithNotebook()` 的 `noteType` 参数替换为 `fileExt`；删除 `fileExtFromNoteType()`；更新 `noteThinSelect`
- [ ] Task 3: 后端绑定层 — `app.go` 签名变更：`CreateNote(title, content, fileExt, notebookID)` / `UpdateNote(id, title, content, fileExt)`；`ImportFiles()` 移除 noteType 推导
- [ ] Task 4: 数据库清理 — `db.go` 移除 `note_type` 回填迁移（3 行）；测试阶段删除旧 DB 文件即可重建
- [ ] Task 5: 种子数据 — `tools/seed/main.go` 移除 `NoteType` 字段，改为 `FileExt` 赋值
- [ ] Task 6: 前端 HTML — `index.html` 删除 `#editorTypeToggle` 按钮
- [ ] Task 7: 前端 JS 重构 — `main.js` 移除 `state.noteType`、`fileExtFromNoteType()`、`switchNoteType()`；新增 `noteTypeFromFileExt()`；改写 `openEditor()`/`saveFileExt()`/`createNote()`/`updateNote()`/快捷键等所有引用
- [ ] Task 8: Wails 绑定同步 — `App.d.ts` / `App.js` / `models.ts` 移除 `note_type` 类型参数
- [ ] Task 9: 构建验证 — `go build ./...` + `npx vite build` 通过

# Task Dependencies

- [Task 1] ~ [Task 5] 无依赖，可并行
- [Task 6] 依赖 [Task 5]（HTML 删除按钮后 JS 清理引用）
- [Task 7] 依赖 [Task 3]（后端 API 签名变更后更新绑定）
- [Task 8] 依赖所有前置任务
