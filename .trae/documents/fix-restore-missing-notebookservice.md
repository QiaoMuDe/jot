# 修复：还原备份后 notebookService 未重建导致下次重置报错

## Summary

`RestoreFromDir()` 和 `ImportDatabaseWithDialog()` 在成功重建连接后，未重新创建 `notebookService`。当用户在此之后再次执行 "恢复出厂设置"，`ResetDatabase()` 中的 `a.notebookService.EnsureDefaultNotebook()` 使用了已 close 的旧连接，报 `sql: database is closed`。

## Current State

### `app.go` — `RestoreFromDir()` Step 5（~1392-1398 行）

```go
a.db = newDB
a.noteService = services.NewNoteService(newDB, a.settingService)
a.tagService = services.NewTagService(newDB)
a.settingService = services.NewSettingService(newDB)
a.aiService = services.NewAIService(newDB)
a.profileService = services.NewProfileService(newDB)
// 缺少 a.notebookService = services.NewNotebookService(newDB)
```

### `app.go` — `ImportDatabaseWithDialog()` Step 5（~461-466 行）

```go
a.db = newDB
a.noteService = services.NewNoteService(newDB, a.settingService)
a.tagService = services.NewTagService(newDB)
a.settingService = services.NewSettingService(newDB)
a.aiService = services.NewAIService(newDB)
a.profileService = services.NewProfileService(newDB)
// 同样缺少 a.notebookService = services.NewNotebookService(newDB)
```

两处都只有 5 个 service 重建，漏了第 6 个 `notebookService`。

与此对比，我们上次修复的 `reconnectDB()` 已经包含了 `notebookService`，所以错误恢复路径（走 `reconnectDB` 的情况）没有问题，但成功路径（Step 5 直接赋值）漏了。

## Proposed Changes

### 1. `app.go` — `RestoreFromDir()` Step 5

在 `a.profileService` 之后添加：
```go
a.notebookService = services.NewNotebookService(newDB)
```

### 2. `app.go` — `ImportDatabaseWithDialog()` Step 5

同样在 `a.profileService` 之后添加：
```go
a.notebookService = services.NewNotebookService(newDB)
```

## Verification

1. `go vet ./...` 通过
2. 完整复现链路：重置 → 一键还原备份 → 再次重置 → 不再报错
