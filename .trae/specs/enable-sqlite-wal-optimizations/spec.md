# 启用 SQLite WAL 模式与性能优化 PRAGMA Spec

## Why

当前项目使用 SQLite 默认的 **delete 日志模式**，写入性能较差，且多线程场景下读取会阻塞写入。同时缺少常用的性能优化 PRAGMA 配置，数据库在高并发场景下容易出现 "database is locked" 错误。启用 WAL 模式并添加常用优化 PRAGMA 可显著提升数据库读写性能和应用响应速度。

## What Changes

- **`internal/database/db.go`** — `InitDB` 函数中，在连接池配置之后添加 WAL 模式及常用优化 PRAGMA
- **`app.go`** — `replaceDatabase` 函数中，在关闭旧连接后、复制新数据库文件前，清理残留的 `-wal`/`-shm` 文件，防止导入/还原时数据损坏

## Impact

- Affected specs: 数据库初始化、数据备份与还原
- Affected code:
  - `internal/database/db.go` — `InitDB` 函数
  - `app.go` — `replaceDatabase` 函数

## ADDED Requirements

### Requirement: SQLite 优化 PRAGMA

The system SHALL configure SQLite with WAL mode and performance optimization PRAGMAs after opening the database connection.

#### Scenario: 数据库初始化时应用优化配置

- **WHEN** `InitDB` 打开 SQLite 连接并配置连接池后
- **THEN** 系统按顺序执行以下 PRAGMA：
  - `PRAGMA journal_mode=WAL` — 启用 WAL 模式，提升并发读写性能
  - `PRAGMA busy_timeout=5000` — 忙等待超时 5 秒，避免 "database is locked"
  - `PRAGMA synchronous=NORMAL` — WAL 模式下安全且性能更好
  - `PRAGMA cache_size=-8000` — 8MB 页面缓存
- **AND** 如果任一 PRAGMA 执行失败，打印警告日志但不中断初始化流程

### Requirement: 导入/还原时清理 WAL 残留文件

The system SHALL clean up stale `-wal` and `-shm` files before replacing the database file during import/restore operations.

#### Scenario: 导入或还原数据库时清理 WAL 文件

- **WHEN** `replaceDatabase` 关闭旧数据库连接后（Step 2）
- **AND** 复制新数据库文件前（Step 3）
- **THEN** 系统删除 `{dbPath}-wal` 和 `{dbPath}-shm` 文件（如果存在），忽略删除失败