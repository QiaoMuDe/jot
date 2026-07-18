# Tasks

- [x] Task 1: `InitDB` 中添加 WAL 模式及优化 PRAGMA
  - 在 `internal/database/db.go` 的 `InitDB` 函数中，`sqlDB.SetMaxOpenConns(1)` 之后（第 49 行后）添加 PRAGMA 执行代码
  - 依次执行：`journal_mode=WAL`、`busy_timeout=5000`、`synchronous=NORMAL`、`cache_size=-8000`
  - 每个 PRAGMA 执行失败时打印警告日志（`fmt.Printf`），不中断初始化
  - 添加函数级注释说明 PRAGMA 作用

- [x] Task 2: `replaceDatabase` 中添加 WAL 残留文件清理
  - 在 `app.go` 的 `replaceDatabase` 函数中，Step 2（关闭旧连接）之后、Step 3（复制 db 文件）之前添加清理逻辑
  - 删除 `{dbPath}-wal` 和 `{dbPath}-shm` 文件，忽略 `os.Remove` 的错误

# Task Dependencies

- 无依赖关系，两个任务可独立执行