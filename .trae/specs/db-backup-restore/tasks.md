# Tasks

## Task 1: 后端导出 — VACUUM INTO 备份

将 `NoteService.ExportAll()` 重构为 DB 文件备份方案。

- 新增 `NoteService.ExportBackup(destPath string) error`：
  - 使用 `s.db.Exec("VACUUM INTO ?", destPath)` 创建压缩副本
  - 返回错误
- 更新 `app.go:ExportDataWithDialog()`：
  - 调用 `ExportBackup(tempPath)` 导出到临时文件
  - 用 `runtime.SaveFileDialog` 获取用户保存路径
  - 将临时文件移到用户选择路径
  - 返回 `"导出成功：/path/to/file.db"`
- 清理：移除 `ExportAll()`、`ImportFromJSON()`、`ExportNoteItem`、`ExportTag`

## Task 2: 后端导入 — 文件覆盖 + 重连

新增 `App.ImportDatabaseWithDialog()` 方法实现导入还原。

- 用 `runtime.OpenFileDialog` 打开文件选择器筛选 `.db` 文件
- 实现导入流程：
  1. 备份当前 DB → `<dbPath>.bak`
  2. 关闭旧连接：`sqlDB, _ := a.db.DB(); sqlDB.Close()`
  3. 复制选定文件到 DB 路径
  4. 调用 `database.InitDB(dbPath)` 重开
  5. 重建 `a.db` / `a.noteService` / `a.tagService` / `a.settingService`
  6. 删除 `.bak`
- 出错时：从 `.bak` 恢复，尝试重开，返回错误 Toast
- 返回 `ImportResult{Message: "已还原数据库备份"}`

## Task 3: 前端更新 — 导入导出交互适配

- 更新 `exportData()` — 调用 `App.ExportDataWithDialog()`，Toast 展示结果
- 更新 `importData()`：
  - 不再用 `<input type="file">`，改为直接调 `App.ImportDatabaseWithDialog()`
  - 成功时：显示 Toast + 调用 `loadNotes() + loadDataStats() + loadTags()` 刷新
- 更新 HTML 按钮描述文案

## Task 4: 清理旧代码

- `types.go` — 移除 `ExportNoteItem`、`ExportTag`
- `note_service.go` — 移除 `ExportAll`、`ImportFromJSON`
- 检查是否还有其他引用这些类型的地方

# Task Dependencies

- Task 1, 2 可并行开发
- Task 3 依赖 Task 1, 2（需要先确认后端 API 签名）
- Task 4 依赖 Task 1, 2（需要在旧代码不再被引用后移除）
