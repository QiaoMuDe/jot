# 移除草稿自动保存功能（全链路）

## Why

移除整个草稿（Draft）相关功能，包括前端自动保存、草稿恢复弹窗、以及后端 Draft 模型/Service/数据库迁移。原因：
1. 已有手动保存（按钮/Ctrl+S）保障，草稿自动保存带来冗余
2. 草稿表 `drafts` 仅 1 行（ID=1），设计上就是临时方案
3. 前后端合计约 150 行维护成本，收益不匹配

## What Changes

### Backend — 删除
- **`internal/models/draft.go`** — 整个文件（`Draft` 结构体）
- **`internal/services/draft_service.go`** — 整个文件（`DraftService`，含 SaveDraft/GetDraft/ClearDraft）
- **`internal/database/db.go`** — `AutoMigrate` 参数中去掉 `&models.Draft{}`
- **`app.go`** — 删除：
  - `draftService` 字段
  - `NewDraftService(db)` 初始化
  - `SaveDraft`/`GetDraft`/`ClearDraft` 三个绑定方法

### Frontend — 删除
- **`main.js`**：
  - `startAutoSave()` 函数
  - `autoSaveTimer` 全局变量
  - `els.autoSaveIndicator` 定义
  - `onEditorInput()` 中的 `startAutoSave()` 调用
  - `openEditor()` 中 autoSaveIndicator/autoSaveTimer 清理代码
  - `closeEditor()` 中 autoSaveTimer 清理代码
  - 草稿恢复弹窗（`loadNotes()` 中 `setTimeout` 检测草稿+弹窗，约 L768-L790）
  - `createNote()` 中的 `ClearDraft()` 调用
  - `saveEditorContent()` 中的 `SaveDraft` 分支
  - `handleAppExit()` 中的 `ClearDraft` 调用（保存后/放弃时）
  - 取消按钮中的 `ClearDraft()` 调用
  - 蒙层点击/放弃中的 `ClearDraft()` 调用
- **`index.html`** — 删除 `<span class="auto-save-indicator" id="autoSaveIndicator">`
- **`style.css`** — 删除 `.auto-save-indicator` 相关规则

### Frontend — 保留（修改）
- `saveEditorContent()` — 保留函数，仅保留 `UpdateNote` 分支（编辑态退出前保存）
- `handleAppExit()` — 保留确认对话框，删除草稿清除逻辑

## Impact

- Affected code: `frontend/src/main.js`（删除约 70 行）
- Affected code: `frontend/index.html`（删除 1 行）
- Affected code: `frontend/src/style.css`（删除约 15 行）
- Affected code: `internal/models/draft.go`（删除整个文件）
- Affected code: `internal/services/draft_service.go`（删除整个文件）
- Affected code: `internal/database/db.go`（修改 1 行）
- Affected code: `app.go`（删除约 15 行）
