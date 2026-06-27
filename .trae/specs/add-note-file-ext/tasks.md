# Tasks

- [x] Task 1: Note 模型新增 `FileExt` 字段 — 在 `internal/models/note.go` 中添加 `FileExt string gorm:"size:10;default:.txt" json:"file_ext"`；`internal/database/db.go` 中 AutoMigrate 自动增列
- [x] Task 2: 启动时迁移存量数据 — 在 `App.startup()` 或 AutoMigrate 后执行一次性 SQL：`note_type='markdown'` → `file_ext='.md'`，其他（含 NULL/空）→ `'.txt'`（需判断列是否存在，避免重复迁移报错）
- [x] Task 3: 创建/更新笔记时自动设置 FileExt — `app.go` 中 `CreateNote` 调用 `noteService.CreateWithNotebook` 后在 service 层自动按 `noteType` 设置 `FileExt`；`UpdateNote` 同理
- [x] Task 4: 导出方法改用 `note.FileExt` — `ExportNoteAsMarkdown` 中的 `defaultName` 拼接改为 `sanitizeFilename(note.Title) + note.FileExt`
- [x] Task 5: 构建验证 — 运行 `go build ./...` 确保编译通过

# Task Dependencies
- 无，可顺序执行
