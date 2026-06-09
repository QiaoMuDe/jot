# Tasks

- [x] Task 1: 后端修改 — `GetAllByNotebook` 和 `Search` 使用 `Select` 截断 Content，新增 `GetNoteContent` 方法
  - [x] SubTask 1.1: `note_service.go` — `GetAllByNotebook` 增加 `.Select(...)` 排除 Content，替换为 `SUBSTR(content, 1, 200) AS content`
  - [x] SubTask 1.2: `note_service.go` — `Search` / `SearchByNotebook` 增加相同的 `.Select(...)`
  - [x] SubTask 1.3: `note_service.go` — 新增 `GetNoteContent(id uint) (string, error)` 方法（单行查询，仅返回 content 字段）
  - [x] SubTask 1.4: `app.go` — 新增 `GetNoteContent(id uint) (string, error)` 对外暴露方法
- [x] Task 2: 前端修改 — `openEditor` 按需加载完整 Content
  - [x] SubTask 2.1: `main.js` — `openEditor()` 中改为调用 `GetNoteContent(noteId)` 获取完整内容注入 CM6
  - [x] SubTask 2.2: 预览模式：查看页的 markdown 预览也使用 `GetNoteContent` 返回的完整内容渲染
- [x] Task 3: 构建验证
  - [x] SubTask 3.1: `go build ./...` 编译通过
  - [x] SubTask 3.2: `cd frontend && npm run build` 构建通过
  - [x] SubTask 3.3: `go vet ./...` 无告警

# Task Dependencies

- Task 2 依赖 Task 1（前端需要后端 `GetNoteContent` 方法就绪）
- Task 3 依赖 Task 1 和 Task 2
