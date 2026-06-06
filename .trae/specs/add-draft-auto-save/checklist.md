# Checklist

- [x] Draft 模型定义包含 ID(uint)/Title(string)/Content(string)/CreatedAt/UpdatedAt 字段
- [x] `database/db.go` 中 AutoMigrate 包含 `&models.Draft{}`
- [x] `DraftService` 实现了 `SaveDraft`（upsert ID=1）、`GetDraft`、`ClearDraft`
- [x] `app.go` 绑定并暴露 `SaveDraft`/`GetDraft`/`ClearDraft` 三个方法给前端
- [x] 新建笔记时 `startAutoSave()` 调用 `App.SaveDraft()` 写入草稿表（而非跳过）
- [x] 标题和内容均为空时 `startAutoSave()` 不保存草稿
- [x] 编辑已有笔记时 `startAutoSave()` 仍然调用 `UpdateNote()`，不受影响
- [x] `loadNotes()` 完成后延迟 1s 检测草稿（编辑器未打开时才检测）
- [x] 检测到非空草稿时弹出恢复确认弹窗
- [x] 点击「恢复」→ 清除草稿 → 打开新建编辑器 → 填入标题和内容
- [x] 点击「放弃」→ 清除草稿 → 不做其他操作
- [x] `createNote()` 成功后调用 `App.ClearDraft()`
- [x] `closeEditor()` 时**不**清除草稿（草稿保留用于下次恢复）
- [x] 全局 lint (`golangci-lint run ./...`) 0 issues
- [x] 功能自测通过（新建打字 → 关闭 → 重新进入首页 → 弹窗恢复 → 保存后不再弹窗）
