# 重构：提取统一的服务重建方法

## 问题

现在有 4 处需要重建所有 service 的代码，但分散在不同地方且不一致：

| 位置                                  | 行号        | 包含的 service | 缺漏                          |
| ----------------------------------- | --------- | ----------- | --------------------------- |
| `NewApp()` 初始化                      | 50-58     | 6 个         | 无（初始创建，用 struct literal）    |
| `ImportDatabaseWithDialog()` Step 5 | 462-466   | 5 个         | **缺** **`notebookService`** |
| `RestoreFromDir()` Step 5           | 1394-1398 | 5 个         | **缺** **`notebookService`** |
| `reconnectDB()`                     | 1593-1598 | 6 个         | 无（上次修复补齐）                   |

每次新增/删除一个 service 类都需要同步修改这 4 处，且只要漏一处就会引起 `sql: database is closed` 类错误。

## 方案

在 `App` 上新增一个 `rebuildServices(db *gorm.DB)` 方法，统一管理所有 service 的重新创建。后续 3 处重连点都调用此方法。

### 改动清单

### 1. `app.go` — 新增 `rebuildServices(db *gorm.DB)`

```go
// rebuildServices 使用新的数据库连接重新创建所有服务实例
func (a *App) rebuildServices(db *gorm.DB) {
	a.noteService = services.NewNoteService(db, a.settingService)
	a.tagService = services.NewTagService(db)
	a.notebookService = services.NewNotebookService(db)
	a.settingService = services.NewSettingService(db)
	a.aiService = services.NewAIService(db)
	a.profileService = services.NewProfileService(db)
}
```

注意：`NewNoteService` 第二个参数是 `settingService`。在替换 `db` 前，`a.settingService` 还是旧连接。执行顺序上需要**先替换** **`a.settingService`，再调用** **`NewNoteService`**。所以调用方应该这样做：

```go
a.settingService = services.NewSettingService(newDB)
a.rebuildServices(newDB)
```

但这样又显得很奇怪... 其实观察现有的代码，`NewNoteService` 的第二个参数只用于读取设置（如分页大小、排序），这些在重连后会重新从新的 DB 读取。所以 `NewNoteService(newDB, a.settingService)` 中的 `a.settingService` 实际上应该是已经指向新 DB 的实例。

更好的做法：

```go
func (a *App) rebuildServices(db *gorm.DB) {
	a.settingService = services.NewSettingService(db)
	a.noteService = services.NewNoteService(db, a.settingService)
	a.tagService = services.NewTagService(db)
	a.notebookService = services.NewNotebookService(db)
	a.aiService = services.NewAIService(db)
	a.profileService = services.NewProfileService(db)
}
```

这样内部按依赖顺序排列，调用方只需 `a.rebuildServices(newDB)` 一行。

### 2. `app.go` — `ImportDatabaseWithDialog()` Step 5

原（461-466 行）：

```go
a.db = newDB
a.noteService = services.NewNoteService(newDB, a.settingService)
a.tagService = services.NewTagService(newDB)
a.settingService = services.NewSettingService(newDB)
a.aiService = services.NewAIService(newDB)
a.profileService = services.NewProfileService(newDB)
```

改为：

```go
a.db = newDB
a.rebuildServices(newDB)
```

### 3. `app.go` — `RestoreFromDir()` Step 5

原（1393-1398 行）：

```go
a.db = newDB
a.noteService = services.NewNoteService(newDB, a.settingService)
a.tagService = services.NewTagService(newDB)
a.settingService = services.NewSettingService(newDB)
a.aiService = services.NewAIService(newDB)
a.profileService = services.NewProfileService(newDB)
```

改为：

```go
a.db = newDB
a.rebuildServices(newDB)
```

### 4. `app.go` — `reconnectDB()` Step 2

原（1592-1598 行）：

```go
a.db = db
a.noteService = services.NewNoteService(db, a.settingService)
a.tagService = services.NewTagService(db)
a.settingService = services.NewSettingService(db)
a.aiService = services.NewAIService(db)
a.profileService = services.NewProfileService(db)
a.notebookService = services.NewNotebookService(db)
```

改为：

```go
a.db = db
a.rebuildServices(db)
```

## 验证

1. `go vet ./...` 通过
2. 确认 3 处底层重建代码完全一致
3. 首次进入应用（`NewApp`）正常
4. 重置出厂设置正常
5. 一键还原备份后再次重置正常

