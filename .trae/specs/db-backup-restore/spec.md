# DB 文件备份恢复 Spec

## Why

现在到导出导入只导出 JSON 格式的笔记和标签，无法精确还原数据库完整状态（时间戳、关联关系、配置、回收站等）。改用 SQLite 原生文件备份实现完整的"备份→还原"功能。

## What Changes

- **导出**：从 JSON 导出改为使用 SQLite `VACUUM INTO` 创建当前数据库的压缩副本，保存为 `.db` 文件
- **导入**：关闭当前数据库连接 → 用选定 `.db` 文件覆盖当前数据库文件 → 重新初始化连接 → 重建所有 Service 实例
- **文件扩展名**：导出使用 `.db` 后缀（默认文件名 `jot-backup-YYYY-MM-DD.db`）
- **按钮描述更新**：
  - 导出：`将所有笔记导出为 JSON 文件保存到本地` → `将当前数据库完整备份为 .db 文件`
  - 导入：`从本地的 JSON 文件恢复笔记` → `从 .db 文件还原数据库备份`
- **移除**：旧的 `ExportNoteItem` / `ExportTag` / `ImportFromJSON` / `ExportAll` 相关代码
- **保留**：`ImportResult` 结构体继续用于导入结果展示

## Impact

- Affected specs: 数据管理功能
- Affected code:
  - `internal/services/types.go` — 移除 `ExportNoteItem` / `ExportTag`
  - `internal/services/note_service.go` — 移除 `ExportAll` / `ImportFromJSON`，新增 `ExportBackup()`
  - `internal/services/setting_service.go` — 新增 `ResetDB(db *gorm.DB)` 用于外层刷新
  - `app.go` — `ExportDataWithDialog` 改为 VACUUM INTO 写 DB 文件；`ImportData` 改为文件覆盖 + 重连
  - `frontend/src/main.js` — 更新导入文件选择器的 `accept` 类型为 `.db`
  - `frontend/index.html` — 导入/导出按钮描述文案更新

## ADDED Requirements

### Requirement: DB 文件导出

The system SHALL support exporting the entire SQLite database as a `.db` file.

- **WHEN** user clicks "导出数据"
- **AND** selects a save location in the native save dialog
- **THEN** system creates a compacted copy of the current database at the chosen path using `VACUUM INTO`

- **WHEN** the export succeeds
- **THEN** show a success Toast with the file path

- **WHEN** the export fails (disk full, permission denied)
- **THEN** show an error Toast

### Requirement: DB 文件导入（覆盖还原）

The system SHALL support importing a `.db` file to completely replace the current database.

- **WHEN** user clicks "导入数据"
- **AND** selects a `.db` file via the native file dialog
- **THEN** system shall:
  1. Backup the current database to `<dbPath>.bak`
  2. Close the existing SQLite connection
  3. Copy the selected file to the current database path
  4. Call `database.InitDB()` to reopen and re-migrate
  5. Recreate all Service instances with the new `*gorm.DB`
  6. Delete the backup file on success

- **WHEN** the import succeeds
- **THEN** show a success Toast "已还原数据库备份"
- **AND** frontend automatically refreshes all data (notes, tags, settings, stats)

- **WHEN** the import fails at any step (invalid file, copy error, reopen error)
- **THEN** system shall restore the backup `.bak` file
- **AND** attempt to reopen the database
- **AND** show an error Toast with the failure reason

## MODIFIED Requirements

### Requirement: 导出按钮交互

前端导出按钮改为调用后端 `ExportDataWithDialog()`（内部走 `VACUUM INTO` + `runtime.SaveFileDialog`），不再需要读取 JSON 字符串。

### Requirement: 导入按钮交互

前端导入按钮改为调用后端 `ImportDatabaseWithDialog()`（内部走 `runtime.OpenFileDialog` + 文件覆盖 + 重连），不再需要前端读取文件内容。

### Requirement: 前端刷新

导入成功后，前端 `loadNotes()` / `loadDataStats()` / `loadTags()` 全量刷新以反映还原后的数据。
