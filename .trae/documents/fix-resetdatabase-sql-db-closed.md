# 修复: ResetDatabase 后 "sql: database is closed"

## Summary

`ResetDatabase()` 删除所有表重建后，底层 SQLite 连接进入无效状态，后续所有数据库操作都报 `sql: database is closed`。

## Current State

`app.go` 的 `ResetDatabase()`（\~1530-1570 行）：

```go
func (a *App) ResetDatabase() error {
    // 1. Drop all 7 tables in a loop
    // 2. AutoMigrate all models
    // 3. InitDefaultTags
    // 4. InitDefaultSettings  
    // 5. EnsureDefaultNotebook
    return nil
}
```

所有操作使用同一个 `a.db`（`*gorm.DB`），`SetMaxOpenConns(1)`。`glebarez/sqlite` 驱动在全表 DropTable 后底层连接状态失效，但 Go 层面 `sql.DB` 对象仍存活，后续操作报 `sql: database is closed`。

已有的 `reconnectDB()`（\~1572 行）正是解决此类问题的标准模式——关闭旧连接，用 `database.InitDB()` 重建新连接，并重新注入所有 service。

## Proposed Changes

### 1. `app.go` — `ResetDatabase()` 末尾追加 `reconnectDB`

在 `ResetDatabase()` 的最后，调用 `reconnectDB` 获取新连接：

```go
// 5. 确保默认笔记本存在
if err := a.notebookService.EnsureDefaultNotebook(); err != nil {
    return err
}

// 6. 重建数据库连接（DropTable 后 glebarez/sqlite 驱动连接可能失效）
dbPath, err := database.DefaultDBPath()
if err != nil {
    return fmt.Errorf("获取数据库路径失败: %w", err)
}
if err := a.reconnectDB(dbPath); err != nil {
    return fmt.Errorf("重置后重连失败: %w", err)
}

return nil
```

注意：`reconnectDB` 内部会关闭旧连接、重建新连接、重新创建所有 service（noteService, tagService, settingService, aiService, profileService, notebookService）。这样就保证了 reset 后数据库连接是全新的、有效的。

## Verification

1. `go vet ./...` 通过
2. 在应用中执行"恢复出厂设置"操作后：

   * 不再弹出 `重置数据库失败: sql: database is closed`

   * 设置页能正常加载

   * 笔记本列表能正常加载

   * 笔记列表能正常加载

   * 页面不会出现其他 "sql: database is closed" 错误

