# 一键清空数据库 / 恢复出厂设置 计划

## Summary

在数据管理页面新增「恢复出厂设置」功能，一键清空所有笔记（含回收站）、标签和设置，重置数据库到初始状态（保留默认标签）。

## Current State Analysis

* 数据管理页面 (`index.html` 视图 5) 已有：统计卡片（笔记总数/标签总数/回收站数）、导出数据、导入数据按钮

* 后端 `note_service.go` 已有：`EmptyTrash()`（清空回收站）、`PermanentDelete()`（单条永久删除）

* 后端 `tag_service.go` 已有：`Delete()`（单条删除，自动解除多对多关联）

* 后端 `setting_service.go` 已有：无批量删除方法

* 前端 `main.js` 已有：`showConfirmDialog()` 自定义确认弹窗、`showImportResult()` 结果提示、`loadDataStats()` 统计刷新

* 确认弹窗 (`confirmDialog`) 已在 DOM 中，可直接复用

## Proposed Changes

### 1. 后端: `note_service.go` — 新增 `ResetAll()`

```go
// ResetAll 清空所有数据（笔记/标签/设置），恢复出厂状态
func (s *NoteService) ResetAll() error
```

逻辑：

* `Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Note{})` — 永久删除所有笔记（含软删除和 join 表记录）

* `Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Tag{})` — 永久删除所有标签（GORM 的 Select 自动清理 note\_tags 关联表）

* 跳过 Settings 的删除（交给 service 层处理，因为 app.go 中更清晰）

### 2. 后端: `setting_service.go` — 新增 `DeleteAll()`

```go
// DeleteAll 删除所有配置项
func (s *SettingService) DeleteAll() error
```

逻辑：

* `Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Setting{})` — 清空 settings 表

### 3. 后端: `app.go` — 新增 `ResetDatabase()` 绑定方法

```go
// ResetDatabase 清空所有数据（笔记/标签/设置），重新初始化默认标签，恢复出厂状态
func (a *App) ResetDatabase() error
```

逻辑：

1. 调用 `a.noteService.ResetAll()` — 清空笔记和标签
2. 调用 `a.settingService.DeleteAll()` — 清空设置
3. 调用 `services.InitDefaultTags(a.db)` — 重新初始化默认标签（因 `InitDefaultTags` 检查标签是否为空，需要访问 db）
4. 返回 error

注意点：

* `App` 结构体中需要持有 `*gorm.DB` 或让 service 都共用同一个 db 连接。当前 `NewApp()` 中的 `db` 是局部变量，需要先提取为 `App` 字段。

* 当前 `App` 结构体没有 `db` 字段，仅有三个 service 实例 → 需新增 `db *gorm.DB` 字段

* 或者将 `ResetAll` 改为接收 `db` 参数，或在 `note_service.go` 中新增方法同时操作三个表（因为三个 service 共享同一个 db 连接实例）

更简洁的方案：在 app.go 中获取 db 实例进行清理。

**修改** **`App`** **结构体**：

```go
type App struct {
    ctx            context.Context
    db             *gorm.DB          // 新增
    noteService    *services.NoteService
    tagService     *services.TagService
    settingService *services.SettingService
}
```

**修改** **`NewApp()`**：

```go
func NewApp() *App {
    dbPath, _ := database.DefaultDBPath()
    db, err := database.InitDB(dbPath)
    if err != nil {
        panic(err)
    }
    return &App{
        db:             db,
        noteService:    services.NewNoteService(db),
        tagService:     services.NewTagService(db),
        settingService: services.NewSettingService(db),
    }
}
```

**在** **`note_service.go`** **或直接 app.go 中使用 db**：

方案A（推荐）：在 `note_service.go` 中暴露一个 ResetAll 方法操作 note+tag，settings 在 app.go 中处理。

```go
// note_service.go
func (s *NoteService) ResetAll() error {
    // 清空所有笔记（包括软删除，自动清理 note_tags 关联）
    if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Note{}).Error; err != nil {
        return err
    }
    // 清空所有标签（自动清理 note_tags 中残留关联）
    if err := s.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Tag{}).Error; err != nil {
        return err
    }
    return nil
}
```

### 4. 前端: `index.html` — 数据管理页面新增按钮

在导出/导入按钮之后，增加分隔线和「恢复出厂设置」按钮：

```html
<!-- 数据管理页面 / viewData -->
<!-- 在现有导出/导入按钮下方 -->
<div class="data-divider"></div>
<button class="data-action-btn data-action-btn-danger" id="resetAllBtn">
    <span class="dab-icon">⚠️</span>
    <span class="dab-text">恢复出厂设置</span>
    <span class="dab-desc">清空所有笔记、标签和设置，此操作不可撤销</span>
</button>
```

### 5. 前端: `main.js` — 新增 reset 处理函数 + 事件绑定

```javascript
// 新增 DOM 引用
els.resetAllBtn: $('resetAllBtn'),

// 新增函数
async function resetDatabase() {
    const confirmed = await showConfirmDialog(
        '确定要恢复出厂设置吗？这将永久删除所有笔记、标签和设置，此操作不可撤销。'
    );
    if (!confirmed) return;

    // 二次确认
    const confirmed2 = await showConfirmDialog(
        '再次确认：所有数据将被清空，且无法恢复。确定要继续吗？'
    );
    if (!confirmed2) return;

    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.ResetDatabase) {
            await window.go.main.App.ResetDatabase();
            showImportResult({ success_count: 0, fail_count: 0, skipped_count: 0, message: '已恢复出厂设置，所有数据已清空' });
        } else {
            console.warn('ResetDatabase 未绑定');
            showImportResult({ success_count: 0, fail_count: 0, skipped_count: 0, message: '功能不可用：后端未绑定' });
        }
    } catch (err) {
        console.error('重置数据库失败:', err);
        showImportResult({ success_count: 0, fail_count: 0, skipped_count: 0, message: '重置失败：' + err.message });
    }
    await loadDataStats();
}

// 事件绑定（在 init 的事件绑定区域）
els.resetAllBtn.addEventListener('click', resetDatabase);
```

### 6. 前端: `style.css` — 危险按钮样式

```css
/* 数据管理页面的危险操作按钮 */
.data-action-btn-danger {
    border-color: var(--danger-border, var(--danger, #ef4444)) !important;
    color: var(--danger, #ef4444) !important;
}
.data-action-btn-danger:hover {
    background: var(--danger-bg, #fef2f2) !important;
    border-color: var(--danger, #ef4444) !important;
}
```

## Files to Modify

| 文件                                     | 改动类型 | 说明                                  |
| -------------------------------------- | ---- | ----------------------------------- |
| `internal/services/note_service.go`    | 修改   | 新增 `ResetAll()` 方法                  |
| `internal/services/setting_service.go` | 修改   | 新增 `DeleteAll()` 方法                 |
| `app.go`                               | 修改   | 新增 `db` 字段 + `ResetDatabase()` 绑定方法 |
| `frontend/index.html`                  | 修改   | 数据管理页面新增「恢复出厂设置」按钮                  |
| `frontend/src/main.js`                 | 修改   | 新增 reset 函数、DOM 引用、事件绑定             |
| `frontend/src/style.css`               | 修改   | 危险按钮样式                              |

## Assumptions & Decisions

1. **二次确认**：使用现有的自定义确认弹框 (`showConfirmDialog`)，调用两次（第一次提示 + 第二次确认）防止误触
2. **重新初始化默认标签**：清理后调用 `InitDefaultTags()` 注入 6 个默认标签
3. **设置重置**：清空 settings 表后，前端下次读取配置时自动使用默认值（各 getter 已处理空值回退）
4. **数据页面刷新**：重置后执行 `loadDataStats()` 刷新统计卡片显示 0
5. **无需重新加载笔记网格**：因为用户在数据管理页面，切回首页时 `loadNotes()` 自动加载空列表
6. **`App`** **结构体新增** **`db`** **字段**：因为 `ResetDatabase()` 需要访问数据库做清理和重新初始化默认标签，三个 service 共享同一个 db 连接

## Verification

1. `cd frontend && npx vite build` — 前端构建无错误
2. 数据管理页面能看到「恢复出厂设置」按钮，带危险样式
3. 点击按钮弹出二次确认
4. 确认后后端清空所有数据并重新初始化默认标签
5. 统计卡片刷新显示 0
6. 切回首页笔记列表为空，再切到回收站为空，切到设置页标签为 6 个默认标签

