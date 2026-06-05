# 一键备份还原 Spec

## Why

数据管理页面的导出/导入需要用户手动选择文件路径，操作步骤多。新增一键备份 → 固定目录、一键还原 → 从固定目录恢复，降低数据保护门槛，同时提供备份信息展示让用户知晓备份状态。

## What Changes

- **后端新增** 2 个绑定方法：`BackupToDir()` 和 `RestoreFromDir()`
- **后端新增** 1 个辅助方法：`GetBackupInfo()` 返回最新备份的文件名/日期/大小
- **前端数据管理页面**：在「数据操作」区新增「备份还原」分区
  - 「一键备份」按钮 → 调用 `BackupToDir()`
  - 「一键还原」按钮 → 调用 `RestoreFromDir()`
  - 备份信息标签 → 显示最新备份的文件名、日期、大小
- 备份目录固定为 `~/.jot/backup/`

## Impact

- Affected specs: 数据管理功能
- Affected code:
  - `app.go` — 新增 `BackupToDir()`、`RestoreFromDir()`、`GetBackupInfo()`
  - `internal/services/note_service.go` — 新增 `ExportBackupTo(path)` 导出到指定路径
  - `frontend/src/main.js` — 新增备份还原按钮交互 + 备份信息标签更新
  - `frontend/index.html` — 新增备份还原分区 UI
  - `frontend/src/style.css` — 新增备份还原区域的样式

## ADDED Requirements

### Requirement: 一键备份

The system SHALL support one-click backup to a fixed directory.

- **WHEN** user clicks "一键备份" button
- **THEN** system creates `~/.jot/backup/` directory if not exists
- **AND** system uses VACUUM INTO to export database to `~/.jot/backup/jot-backup-YYYY-MM-DD_HHmmss.db`
- **AND** system updates the backup info label with the latest backup info
- **AND** system shows a success Toast "备份成功：文件名"

- **WHEN** backup fails (disk full, permission error)
- **THEN** system shows an error Toast with failure reason

### Requirement: 一键还原

The system SHALL support one-click restore from the backup directory.

- **WHEN** user clicks "一键还原" button
- **THEN** system lists `.db` files in `~/.jot/backup/` sorted by modification time (newest first)
- **AND** system picks the latest backup file
- **AND** system imports this file using the same logic as `ImportDatabaseWithDialog` (backup → close → overwrite → reopen → rebuild)
- **AND** system shows a success Toast "已从备份文件恢复：文件名"
- **AND** frontend automatically refreshes all data

- **WHEN** `~/.jot/backup/` directory is empty or doesn't exist
- **THEN** system shows a warning Toast "暂无可用备份"

- **WHEN** restore fails at any step
- **THEN** system restores from the `.bak` file and shows error Toast

### Requirement: 备份信息标签

The system SHALL display backup information in the data management page.

- **WHEN** user enters the data management page
- **THEN** system calls `GetBackupInfo()` to retrieve the latest backup info
- **AND** displays a small label: `最新备份：文件名 | 日期时间 | 文件大小`
- **AND** if no backup exists, displays `暂无备份`

- **WHEN** backup or restore succeeds
- **THEN** the label refreshes automatically

### Requirement: 后端 GetBackupInfo

The system SHALL provide a binding method to get backup info.

- `GetBackupInfo()` returns a struct containing:
  - `file_name` — 最新备份文件名
  - `file_time` — 文件修改时间，格式 `YYYY-MM-DD HH:mm`
  - `file_size` — 文件大小，格式 `X.XX MB`

- **WHEN** backup directory doesn't exist or no `.db` files found
- **THEN** returns empty strings / "暂无备份"

## MODIFIED Requirements

### Requirement: 数据管理页面布局

在「数据操作」区新增「备份还原」分区，位于导出/导入按钮之后、恢复出厂设置之前。
