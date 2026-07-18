# Checklist

- [x] `db.go` 中 `InitDB` 已添加 WAL 模式 PRAGMA（`journal_mode=WAL`）
- [x] `db.go` 中 `InitDB` 已添加 `busy_timeout=5000` PRAGMA
- [x] `db.go` 中 `InitDB` 已添加 `synchronous=NORMAL` PRAGMA
- [x] `db.go` 中 `InitDB` 已添加 `cache_size=-8000` PRAGMA
- [x] PRAGMA 执行失败时打印警告日志，不中断初始化
- [x] `app.go` 中 `replaceDatabase` 在 Step 2 后、Step 3 前已清理 `-wal`/`-shm` 文件
- [x] 构建通过，无编译错误