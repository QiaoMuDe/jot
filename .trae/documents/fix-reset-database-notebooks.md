# 修复：恢复出厂设置未清空笔记本

## 摘要

`ResetDatabase()` 只清空了笔记、标签和设置表，**漏掉了 notebooks 表**。重置后重启应用，`EnsureDefaultNotebook()` 发现 notebooks 表非空（旧笔记本还在），不会创建默认笔记本，导致侧栏仍显示旧的笔记本名。

## 当前状态分析

### 后端 `ResetDatabase()`（[app.go:769-783](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L769-L783)）

```go
func (a *App) ResetDatabase() error {
    if err := a.noteService.ResetAll(); err != nil { return err }     // notes + tags
    if err := a.settingService.DeleteAll(); err != nil { return err }  // settings
    if err := services.InitDefaultTags(a.db); err != nil { return err }
    return nil
}
```

### 启动时 `EnsureDefaultNotebook()`（[notebook_service.go:176-203](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/notebook_service.go#L176-L203)）

只在 notebooks 表**完全为空**时创建默认笔记本。如果旧笔记本还在，它什么都不做。

### 前端 `resetDatabase()`（[main.js:2021-2051](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/main.js#L2021-L2051)）

调用后端 `ResetDatabase()` 后执行 `loadNotebooks()` → 从数据库加载旧笔记本列表并渲染到侧栏。

## 问题链路

```
ResetDatabase()  →  清空 notes/tags/settings
                  →  notebooks 表原封不动
                  →  loadNotebooks() 从后端读到旧笔记本
                  →  侧栏显示旧笔记本名

重启应用:
startup()
  → EnsureDefaultNotebook() 检查 notebooks 表
  → 表非空（旧笔记本还在），什么都不做
  → loadNotebooks() 再次加载旧笔记本
  → 用户看到旧笔记本名
```

## 修改方案

### 修改 1：后端 `ResetDatabase()` 追加 notebookService 清空

在 [app.go:769-783](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L769-L783)，`ResetDatabase()` 末尾追加 notebook 表清理，然后由 `EnsureDefaultNotebook()` 在下次启动时自动重建。

或者更直接：在 ResetDatabase 中清空 notebooks + 立即创建默认笔记本（id=1）。

更干净的方案：调用 `notebookService.ResetAll()` 清空 notebooks 表（硬删除），然后创建一个名称为"默认笔记本"的新笔记本。这样重置后立即生效，不需要重启。

但还有一个微妙问题：`notebook_service.go` 中默认笔记本被硬编码为 ID=1，且笔记迁移、删除保护都基于这个假设。如果清空后重建，新的默认笔记本又是 ID=1，所以自增 ID 需要处理。

实际上 GORM 的 SQLite 在删除所有记录后自增 ID 不会重置，所以新插入的默认笔记本 ID 会是 2（或更大）。这会导致问题。

所以更好的方案是：
1. 硬删除所有笔记本（`Unscoped().Delete()`）
2. 重置 SQLite 的自增序列（`DELETE FROM sqlite_sequence WHERE name='notebooks'`）
3. 创建新的默认笔记本（ID 会重新从 1 开始）

或者另一个简单的方案：在 `ResetDatabase()` 中硬删除所有笔记本后，直接创建默认笔记本。

让我简化这个方案：

**方案：** 在 `NotebookService` 中新增 `ResetAll()` 方法，硬删除所有笔记本并重置自增序列，然后创建默认笔记本（id=1）。在 `ResetDatabase()` 中调用。

### 修改 1：`notebook_service.go` 新增 `ResetAll()`

```go
// ResetAll 清空所有笔记本（硬删除），然后创建默认笔记本（id=1）
func (s *NotebookService) ResetAll() error {
    // 硬删除所有笔记本
    if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Notebook{}).Error; err != nil {
        return err
    }
    // 重置 SQLite 自增序列，确保下一个 ID 为 1
    s.db.Exec("DELETE FROM sqlite_sequence WHERE name='notebooks'")
    // 创建默认笔记本
    return s.db.Create(&models.Notebook{Name: "默认笔记本"}).Error
}
```

### 修改 2：`app.go` 中 `ResetDatabase()` 调用 notebookService.ResetAll()

```go
func (a *App) ResetDatabase() error {
    if err := a.noteService.ResetAll(); err != nil { return err }
    if err := a.settingService.DeleteAll(); err != nil { return err }
    if err := a.notebookService.ResetAll(); err != nil { return err }   // 新增
    if err := services.InitDefaultTags(a.db); err != nil { return err }
    return nil
}
```

### 修改 3：前端 `resetDatabase()` 中在 loadNotebooks 后设 activeNotebookId = 1

前端在 `loadNotebooks()` 后，`activeNotebookId` 可能还是旧的 ID。添加确保逻辑：

```javascript
// 重置后 activeNotebookId 设为新默认笔记本
state.activeNotebookId = 1;
```

## 验证步骤

1. 正常启动应用，创建一些笔记本和笔记
2. 进入设置 → 数据管理 → 恢复出厂设置
3. 确认：侧栏不再显示旧笔记本，只显示"默认笔记本"
4. 重启应用
5. 确认：侧栏仍然只显示"默认笔记本"
6. `EnsureDefaultNotebook()` 不会重复创建默认笔记本
