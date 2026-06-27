# Checklist

- [x] Note 模型 `FileExt` 字段已添加（`.txt`/`.md`）
- [x] AutoMigrate 自动处理新增列，无需手动 SQL
- [x] 存量数据迁移：`note_type='markdown'` → `file_ext='.md'`，其余→`.txt`
- [x] 创建 `note_type='markdown'` 笔记时 `FileExt` 自动设为 `.md`
- [x] 创建 `note_type='text'` 笔记时 `FileExt` 自动设为 `.txt`
- [x] 更新笔记类型时 `FileExt` 同步更新
- [x] `ExportNoteAsMarkdown` 使用 `note.FileExt` 拼接默认文件名
- [x] `go build ./...` 编译通过
