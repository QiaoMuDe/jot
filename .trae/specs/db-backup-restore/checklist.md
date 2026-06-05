# Checklist

## 后端导出
- [x] `NoteService.ExportBackup()` 调用 `VACUUM INTO` 生成 DB 副本
- [x] `ExportDataWithDialog()` 弹出原生保存对话框，默认文件名为 `jot-backup-YYYY-MM-DD.db`
- [x] 导出成功后前端显示 Toast "导出成功：/path/to/file.db"
- [x] 导出失败时前端显示错误 Toast

## 后端导入
- [x] `App.ImportDatabaseWithDialog()` 弹出原生文件选择器，只显示 `.db` 文件
- [x] 导入前备份当前 DB 到 `.bak`
- [x] 关闭旧数据库连接
- [x] 复制选定文件覆盖当前 DB 文件
- [x] 调用 `database.InitDB()` 重开连接
- [x] 重建 `App` 中所有 Service 实例
- [x] 成功后删除 `.bak` 备份
- [x] 失败时从 `.bak` 恢复并返回错误
- [x] 返回 `ImportResult{Message: "已从备份文件恢复数据库"}`

## 前端
- [x] 导出按钮描述改为 `将当前数据库完整备份为 .db 文件`
- [x] 导入按钮描述改为 `从 .db 文件还原数据库备份`
- [x] 导入成功后自动刷新笔记/标签/统计
- [x] `npx vite build` 通过

## 代码清理
- [x] `types.go` 中移除 `ExportNoteItem` / `ExportTag`
- [x] `note_service.go` 中移除 `ExportAll` / `ImportFromJSON`
- [x] 无残留引用旧类型
- [x] `go build ./...` 通过
