# 数据库瘦身按钮 Spec

## Why

用户长时间使用 jot 后，频繁增删改笔记会导致 SQLite 数据库文件只增不减（SQLite 不自动回收已删除数据页），文件体积膨胀。需要一个手动触发 `VACUUM` 的入口，让用户可以主动回收已删除数据占用的磁盘空间。

## What Changes

- 后端 `NoteService` 新增 `Vacuum()` 方法，执行 `VACUUM` SQL 命令
- 后端 `App` 新增 `VacuumDatabase()` 绑定方法，供前端 Wails 调用，执行前后记录文件大小
- 前端 `index.html` 数据操作区域新增"数据库瘦身"按钮
- 前端 `main.js` 新增 `vacuumDatabase()` 函数及事件绑定

## Impact

- Affected specs: 数据管理页面
- Affected code:
  - `internal/services/note_service.go` — 新增 `Vacuum()`
  - `app.go` — 新增 `VacuumDatabase()`
  - `frontend/index.html` — 新增按钮
  - `frontend/src/main.js` — 新增函数和事件绑定

## ADDED Requirements

### Requirement: Database Vacuum

The system SHALL provide a "数据库瘦身" button in the data management page that performs SQLite VACUUM to reclaim disk space.

#### Scenario: Success case
- **WHEN** user clicks "数据库瘦身" button
- **THEN** system executes SQLite VACUUM
- **AND** system shows success notification with freed space amount (e.g., "数据库瘦身完成，释放了 1.2 MB 空间")
- **AND** system refreshes database size statistic card

#### Scenario: Error case
- **WHEN** VACUUM execution fails
- **THEN** system shows error notification with failure reason