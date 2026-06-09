# Checklist

- [x] `GetAllByNotebook` 的 GORM 查询链增加了 `.Select(...)`，Content 替换为 `SUBSTR(content, 1, 200) AS content`
- [x] `Search` 的 GORM 查询链增加了相同的 `.Select(...)`
- [x] `note_service.go` 新增了 `GetNoteContent(id uint) (string, error)` 方法
- [x] `app.go` 新增了 `GetNoteContent(id uint) (string, error)` 对外暴露方法
- [x] 列表和搜索的 `LIKE %keyword%` 仍能正确过滤 content（WHERE 子句不受 `Select` 影响）
- [x] Tags 的 `Preload` 在增加 `Select` 后仍正常工作
- [x] 前端 `openEditor()` 在打开笔记时调用 `GetNoteContent(noteId)` 加载完整内容
- [x] 预览模式的 markdown 渲染使用了完整 content（而非截断版本）
- [x] `go build ./...` 编译通过，无错误
- [x] `cd frontend && npm run build` 构建通过，无错误
- [x] `go vet ./...` 无告警
- [ ] 打开包含大内容笔记的列表页，内存不再急剧上升（需用户验证）
