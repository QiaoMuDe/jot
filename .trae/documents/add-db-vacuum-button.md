# 数据库瘦身按钮（VACUUM）计划

## 概述
在数据管理页面增加一个"数据库瘦身"按钮，点击后执行 SQLite `VACUUM` 命令，重建数据库文件以回收已删除记录占用的磁盘空间，完成后刷新数据统计卡片显示新的数据库大小。

## 当前状态分析
- 数据管理页面位于 `frontend/index.html` 第 377-469 行，包含：数据统计卡片（笔记数/标签数/回收站/笔记本数/数据库大小）、数据操作（导出/导入/恢复出厂设置）、快速备份、数据目录
- 后端 `app.go` 已暴露 `GetDataStats()` 方法，返回包含 `DBSize` 和 `DBSizeStr` 的统计信息
- `note_service.go` 已有 `ExportBackup()` 方法使用 `VACUUM INTO` 导出压缩副本，但无直接的 `VACUUM` 方法
- 前端 `loadDataStats()` 函数（`main.js` 第 1898-1947 行）已用于加载和刷新统计卡片

## 改动清单

### 1. 后端：在 `note_service.go` 添加 `Vacuum()` 方法
- **文件**: `d:\峡谷\Dev\本地项目\jot\internal\services\note_service.go`
- **改动**: 新增方法，执行 `s.db.Exec("VACUUM")`，返回 error
- **为什么**: GORM 的 `Exec` 可直接执行原生 SQL；`VACUUM` 是 SQLite 原生命令，无需额外依赖
- **如何验证**: 调用后数据库文件大小应减小，且不破坏数据完整性

### 2. 后端：在 `app.go` 添加 `VacuumDatabase()` 绑定方法
- **文件**: `d:\峡谷\Dev\本地项目\jot\app.go`
- **改动**: 新增 `VacuumDatabase()` 方法，调用 `noteService.Vacuum()`，执行前后分别获取数据库文件大小以确认瘦身效果
- **为什么**: Wails 通过 `app.go` 中的方法暴露给前端调用

### 3. 前端 HTML：在数据管理页面添加"数据库瘦身"按钮
- **文件**: `d:\峡谷\Dev\本地项目\jot\frontend\index.html`
- **改动**: 在"数据操作"区域（现有导出/导入/重置按钮之后），新增一个 `data-action-btn` 按钮，使用压缩/清理类图标
- **位置**: 放在"数据操作" section 的 `data-actions-row` 之外，作为独立的一行按钮（类似重置按钮的样式）

### 4. 前端 JS：添加事件绑定和瘦身函数
- **文件**: `d:\峡谷\Dev\本地项目\jot\frontend\src\main.js`
- **改动**:
  - 在 `els` 对象中添加 `vacuumDbBtn` 引用（指向新按钮的 id）
  - 新增 `vacuumDatabase()` 异步函数（调用 `window.go.main.App.VacuumDatabase()` 并显示通知）
  - 在按钮事件绑定区域（第 3940-3950 行附近）添加 `els.vacuumDbBtn.addEventListener('click', vacuumDatabase)`
  - 瘦身成功后调用 `loadDataStats()` 刷新统计卡片

### 5. 更新 `GetDataStats()` 在瘦身后自动获取新大小
- 无需额外改动：`GetDataStats()` 每次调用都实时从 `os.Stat` 读取文件大小，瘦身后自动反映新值

## 实现细节

### `note_service.go` 新增方法
```go
// Vacuum 执行 SQLite VACUUM 命令，回收已删除数据占用的磁盘空间
func (s *NoteService) Vacuum() error {
	return s.db.Exec("VACUUM").Error
}
```

### `app.go` 新增方法
```go
// VacuumDatabase 对当前数据库执行 VACUUM 瘦身操作
func (a *App) VacuumDatabase() (string, error) {
	// 获取瘦身前大小
	dbPath, _ := database.DefaultDBPath()
	var beforeSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		beforeSize = fi.Size()
	}

	if err := a.noteService.Vacuum(); err != nil {
		return "", fmt.Errorf("数据库瘦身失败: %w", err)
	}

	// 获取瘦身后大小
	var afterSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		afterSize = fi.Size()
	}

	saved := beforeSize - afterSize
	var savedStr string
	switch {
	case saved < 0:
		savedStr = "0"
	case saved < 1024:
		savedStr = fmt.Sprintf("%d B", saved)
	case saved < 1024*1024:
		savedStr = fmt.Sprintf("%.1f KB", float64(saved)/1024)
	default:
		savedStr = fmt.Sprintf("%.1f MB", float64(saved)/(1024*1024))
	}

	return fmt.Sprintf("数据库瘦身完成，释放了 %s 空间", savedStr), nil
}
```

### HTML 新增按钮
```html
<!-- 在"数据操作"区域内，恢复出厂设置按钮之后 -->
<button class="data-action-btn" id="vacuumDbBtn">
    <span class="dab-icon"><!-- 压缩/清理 SVG 图标 --></span>
    <span class="dab-text">数据库瘦身</span>
    <span class="dab-desc">VACUUM 回收已删除数据占用的磁盘空间</span>
</button>
```

### JS 新增函数
```javascript
async function vacuumDatabase() {
    try {
        if (window.go && window.go.main && window.go.main.App && window.go.main.App.VacuumDatabase) {
            const msg = await window.go.main.App.VacuumDatabase();
            nm.show(msg, 'success');
            await loadDataStats();
        } else {
            nm.show('数据库瘦身功能不可用', 'error');
        }
    } catch (err) {
        nm.show('数据库瘦身失败：' + err.message, 'error');
    }
}
```

## 验证步骤
1. 构建应用：`cd frontend && npx wails build`（或开发模式 `wails dev`）
2. 打开数据管理页面，确认看到"数据库瘦身"按钮
3. 先创建若干笔记，再删除部分笔记（产生已删除但未回收的空间）
4. 点击"数据库瘦身"按钮
5. 确认通知提示成功消息，数据库大小统计卡片更新为新的（更小的）数值