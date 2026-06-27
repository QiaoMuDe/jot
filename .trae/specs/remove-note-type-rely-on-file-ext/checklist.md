# Checklist

- [ ] Note 模型不再包含 `NoteType` 字段
- [ ] `note_service.go` 的 `Create`/`Update`/`CreateWithNotebook` 使用 `fileExt` 而非 `noteType`
- [ ] `note_service.go` 的 `fileExtFromNoteType()` 已删除
- [ ] `noteThinSelect` 不再包含 `note_type`
- [ ] `app.go` 的 `CreateNote`/`UpdateNote` 签名改为 `fileExt`
- [ ] `ImportFiles()` 直接设置 `fileExt` 而非 `noteType`
- [ ] `db.go` 中旧的 `note_type` 回填迁移已移除
- [ ] 种子数据使用 `FileExt` 而非 `NoteType`
- [ ] HTML 中 `#editorTypeToggle` 已移除
- [ ] JS 中 `state.noteType` 已移除
- [ ] JS 中 `fileExtFromNoteType()` 和 `switchNoteType()` 已移除
- [ ] JS 中 `noteTypeFromFileExt()` 已添加
- [ ] JS 中 `createNote()`/`updateNote()` 传 `fileExt`
- [ ] `go build ./...` 编译通过
- [ ] `npx vite build` 构建通过
- [ ] 修改后缀为 `.md` 可触发 Markdown 模式
- [ ] 修改后缀为 `.txt` 或 `.py` 进入纯文本模式
