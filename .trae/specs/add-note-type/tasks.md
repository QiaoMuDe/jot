# Tasks

- [x] Task 1: Note 模型新增 `NoteType` 字段 — 在 `internal/models/note.go` 中添加 `NoteType string` 字段；`internal/database/db.go` 中 AutoMigrate 自动增列；app.go 中 `CreateNote`/`UpdateNote` 新增 `noteType` 参数并透传
- [x] Task 2: 编辑器状态栏添加笔记类型切换器 — 在底部状态栏左侧（auto-save-indicator 右侧）新增「纯文本」「Markdown」双按钮选择器
  - [x] 子任务 2.1: index.html 新增 `.editor-note-type` 结构
  - [x] 子任务 2.2: main.js 中 `openEditor()` 新建默认 `state.noteType = 'text'`，编辑按 `note.note_type` 加载（空值→`"text"`）
  - [x] 子任务 2.3: 切换时控制 `els.editorModes`（底部模式按钮）显示/隐藏：text 隐藏，markdown 显示
  - [x] 子任务 2.4: `createNote()`/`updateNote()` 时携带 `state.noteType` 值
- [x] Task 3: 查看模式按类型选择渲染方式 — `openEditor(_, _, readonly)` 中根据 `state.noteType` 判断：`"text"` 用 `<pre>` 显示原始文本跳过 marked；`"markdown"` 走现有渲染（`.md-rendered` + marked）
- [x] Task 4: CSS 样式 — 类型切换器按钮样式（与底部模式切换胶囊按钮风格一致），选中态高亮
- [x] Task 5: AGENTS.md 更新 — 添加笔记类型功能记忆条目
- [x] Task 6: lint 验证 — 运行 `golangci-lint run ./...` 确保无问题（0 issues）
- [x] Task 7: Checklist 验证 — 用 checklist.md 逐项验证所有检查点（15/15 通过）

# Task Dependencies
- 无，所有任务可顺序执行
