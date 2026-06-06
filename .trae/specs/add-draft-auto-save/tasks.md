# Tasks

- [x] Task 1: 后端 — Draft 数据模型 + Service + AutoMigrate
  - 新建 `internal/models/draft.go`，定义 `Draft` 结构体（ID/Title/Content/CreatedAt/UpdatedAt）
  - 新建 `internal/services/draft_service.go`，实现 `SaveDraft`（upsert ID=1）、`GetDraft`、`ClearDraft`
  - `database/db.go` 中 AutoMigrate 追加 `&models.Draft{}`

- [x] Task 2: 后端 — app.go 绑定 3 个 Draft 方法
  - 在 `app.go` 的 `App` 结构体上新增 `SaveDraft(title, content string) error`
  - 新增 `GetDraft() (*models.Draft, error)` — 无记录时返回 nil
  - 新增 `ClearDraft() error`

- [x] Task 3: 前端 — 修改 `startAutoSave()` 支持新建笔记草稿保存
  - 当 `state.editingNoteId` 为 null 时，调用 `App.SaveDraft(title, content)`
  - 标题和内容均为空时不保存
  - 保存成功指示器显示「草稿已保存 ✓」（复用现有 autoSaveIndicator 元素）

- [x] Task 4: 前端 — 草稿检测与恢复弹窗
  - 在 `loadNotes()` 末尾，延迟 1s 后调用 `App.GetDraft()`
  - 如果有非空草稿 → 使用现有 `showConfirmDialog()` 弹窗：「发现未保存的草稿，是否恢复？」
  - 「恢复」按钮 → `App.ClearDraft()` → `openEditor(null)` → 填入草稿标题和内容
  - 「放弃」按钮 → `App.ClearDraft()`

- [x] Task 5: 前端 — 草稿清除时机对接
  - `createNote()` 成功后 → `App.ClearDraft()`
  - `closeEditor()` 中**不**清除草稿（草稿保留用于下次恢复）
  - 编辑器打开时（`openEditor` 执行期间）不触发草稿检测

# Task Dependencies
- [Task 3] depends on [Task 1, Task 2]
- [Task 4] depends on [Task 1, Task 2]
- [Task 5] depends on [Task 3]
