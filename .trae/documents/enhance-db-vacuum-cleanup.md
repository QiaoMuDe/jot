# 存储优化功能增强计划

## 现有逻辑回顾

当前 `VacuumDatabase()` 的执行顺序：
1. 清理空 AI 会话（`DeleteEmptyAISessions()`）
2. 执行 SQLite VACUUM 命令

> **说明**：新增的设置项 `trash_cleanup_retention_days` 完全遵循现有模式——字段加入 `SettingsConfig` 结构体，读取在 `GetAllSettings()`、写入在 `SaveAllSettings()`、DOM 加载在 `loadSettings()`、DOM 保存在 `saveSettings()`，不新增任何独立的方法函数。

## 变更一览

| # | 变更内容 | 涉及文件 |
|---|---------|---------|
| 1 | 按钮重命名："数据库瘦身" → "存储优化"，描述更新 | `index.html` |
| 2 | 新增设置项：回收站清理保留天数（默认30天） | `db.go`, `types.go`, `index.html`, `main.js` |
| 3 | 清理过期回收站笔记（>N天） | `note_service.go` / `app.go` |
| 4 | 清理过期回收站笔记本（>N天） | `notebook_service.go` / `app.go` |
| 5 | 清理孤儿 AI 消息（防御性） | `ai_service.go` / `app.go` |
| 6 | 迁移指向不存在笔记本的笔记到默认笔记本 | `note_service.go` / `app.go` |

## 详细变更

### 变更 1：按钮重命名

**文件**：`frontend/index.html` 第 728-736 行

- `dar-label`：`数据库瘦身` → `存储优化`
- `dar-desc`：`回收已删除数据占用的空间` → `回收存储空间，清理无效数据`

### 变更 2：新增设置项 `trash_cleanup_retention_days`

#### 2a. 默认值
**文件**：`internal/database/db.go` 第 497-521 行的 `defaults` 切片中追加：
```go
{Key: "trash_cleanup_retention_days", Value: "30"},
```

#### 2b. SettingsConfig 结构体
**文件**：`internal/services/types.go` 第 52-77 行

在 `SettingsConfig` 中追加字段：
```go
TrashCleanupRetentionDays int `json:"trash_cleanup_retention_days"`
```

在 `GetAllSettings()` 中追加：
```go
TrashCleanupRetentionDays: parseIntSetting(s.Get("trash_cleanup_retention_days"), 30),
```

在 `SaveAllSettings()` 中追加校验和映射（范围 1-365）：
```go
TrashCleanupRetentionDays: clampInt(cfg.TrashCleanupRetentionDays, 1, 365),
```
并在 `sets` map 中加入：
```go
"trash_cleanup_retention_days": strconv.Itoa(cfg.TrashCleanupRetentionDays),
```

#### 2c. 前端 HTML
**文件**：`frontend/index.html`，在数据管理页面的"存储优化"按钮之前插入设置项

在 `d:\峡谷\Dev\本地项目\jot\frontend\index.html` 的 `data-action-list` 上方（或内部）新增：
```html
<div class="data-action-list" data-group="自动清理">
    <div class="ai-setting-item" style="padding:8px 0;">
        <span class="ai-setting-label">回收站自动清理</span>
        <div class="ai-setting-control">
            <input type="number" id="trashCleanupRetentionDays" class="settings-input" value="30" min="1" max="365" style="width:80px;" />
            <span class="ai-setting-hint" style="white-space:nowrap;">天前的笔记自动清理</span>
        </div>
    </div>
</div>
```

#### 2d. 前端 JS：loadSettings()
**文件**：`frontend/src/main.js` 第 7128-7336 行

在 `loadSettings()` 中添加：
```js
const retentionDays = document.getElementById('trashCleanupRetentionDays');
if (retentionDays) retentionDays.value = cfg.trash_cleanup_retention_days || 30;
```

#### 2e. 前端 JS：saveSettings()
**文件**：`frontend/src/main.js` 第 7341-7381 行

在 `saveSettings()` 的 cfg 对象中添加：
```js
trash_cleanup_retention_days: parseInt(document.getElementById('trashCleanupRetentionDays')?.value) || 30,
```

### 变更 3：清理过期回收站笔记（>N天）

**文件**：`internal/services/note_service.go`，新增方法

```go
// CleanExpiredTrash 永久删除回收站中超过指定天数的笔记
func (s *NoteService) CleanExpiredTrash(days int) int64 {
    // 硬删除 deleted_at 在 N 天之前的笔记
}
```

### 变更 4：清理过期回收站笔记本（>N天）

**文件**：`internal/services/notebook_service.go`，新增方法

```go
// CleanExpiredTrash 永久删除回收站中超过指定天数的笔记本
// 删除前将笔记本下的笔记迁移到默认笔记本
func (s *NotebookService) CleanExpiredTrash(days int) int64 {
    
}
```

### 变更 5：清理孤儿 AI 消息

**文件**：`internal/services/ai_service.go`，新增方法

```go
// DeleteOrphanMessages 删除指向不存在会话的孤儿 AI 消息
func (a *AIService) DeleteOrphanMessages() int64 {
    
}
```

### 变更 6：迁移指向不存在笔记本的笔记

**文件**：`internal/services/note_service.go`，扩展或新增方法

```go
// MigrateOrphanNotes 将所有指向不存在笔记本的笔记迁移到默认笔记本
func (s *NoteService) MigrateOrphanNotes() int64 {
    
}
```

### 变更 7：整合到 VacuumDatabase()

**文件**：`app.go` 第 333-374 行

```go
func (a *App) VacuumDatabase() (string, error) {
    // 1. 读取保留天数设置
    daysStr := a.settingService.Get("trash_cleanup_retention_days")
    days, _ := strconv.Atoi(daysStr)
    if days <= 0 { days = 30 }
    
    // 2. 清理空 AI 会话
    deletedSessions := a.aiService.DeleteEmptyAISessions()
    
    // 3. 清理孤儿 AI 消息
    deletedOrphanMsgs := a.aiService.DeleteOrphanMessages()
    
    // 4. 清理过期回收站笔记
    deletedNotes := a.noteService.CleanExpiredTrash(days)
    
    // 5. 清理过期回收站笔记本
    deletedNotebooks := a.notebookService.CleanExpiredTrash(days)
    
    // 6. 迁移孤儿笔记
    migratedNotes := a.noteService.MigrateOrphanNotes()
    
    // 7. 执行 VACUUM
    // ... 现有逻辑 ...
    
    // 8. 组装消息
    var parts []string
    // ...
}
```

### 变更 8：更新 AGENTS.md

新增记忆点记录"存储优化"功能的变更。

## 已排除的清理项

| 项 | 排除原因 |
|---|---------|
| 清理孤儿 `note_tags` 关联记录 | GORM many2many 自动维护，不会产生孤儿 |
| 清理废弃设置键 | 用户要求排除 |
| 清理未使用的 API 配置 | 用户要求排除 |
| 清理旧备份文件 | 用户要求排除 |

## 验证方式

1. `wails build -s` 编译通过
2. `npx vite build` 前端构建通过
3. 检查设置页出现"回收站自动清理"输入框，默认值为 30
4. 输入不同天数（如 7、90），确认前端能正确保存和回显
5. 制造过期回收站数据（修改 `deleted_at` 到 31 天前），运行存储优化，确认被清理
6. 制造孤儿 AI 消息（手动从 `ai_sessions` 中删除一条记录），运行存储优化，确认被清理
7. 制造指向不存在笔记本的笔记，运行存储优化，确认被迁移到默认笔记本
8. 确认按钮名称为"存储优化"，描述正确
