# 待办数据加入数据管理页面 — 实现计划

## 概述

在数据管理页面的信笺统计中新增待办事项统计摘要，并在操作列表中新增「清空已完成待办」的批量维护操作。

## 当前状态分析

### 后端
- [DataStats 结构体](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go#L13-L28) — 有笔记/AI/DB 统计字段，没有待办字段
- [GetDataStats() 聚合函数](file:///d:/峡谷/Dev/本地项目/jot/app.go#L289-L333) — 调用 noteService.GetStats() + 手动追加标签数、DB大小、AI统计
- [todoService](file:///d:/峡谷/Dev/本地项目/jot/internal/services/todo_service.go) — 已有 Create/List/Toggle/Delete/Update，没有统计方法
- [ResetAll()](file:///d:/峡谷/Dev/本地项目/jot/internal/services/note_service.go#L578-L589) — 只清空笔记和标签，不处理待办

### 前端
- [data-management.js loadDataStats()](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js#L36-L136) — 读取 stats 对象并拼接信笺 HTML
- [data-management.js clearAISessions()](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js#L141-L159) — 清空 AI 会话的示例模式（确认对话框 → 调用后端 → reloadStats）
- [index.html #viewData](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L706-L844) — 数据管理页面的 HTML 结构（信笺 + 5 个操作列表分组）

## 改动方案

### 1. 后端：DataStats 新增待办统计字段

**文件**: [internal/services/types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go#L13-L28)

- `DataStats` 结构体新增两个字段：
  ```go
  TotalTodos     int64 `json:"total_todos"`
  CompletedTodos int64 `json:"completed_todos"`
  ```

### 2. 后端：GetDataStats() 添加待办统计查询

**文件**: [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L289-L333)

在 AI 统计之后（第 330 行附近）追加：
```go
// 待办统计
totalTodos, _ := a.todoService.Count()
completedTodos, _ := a.todoService.CountCompleted()
stats.TotalTodos = totalTodos
stats.CompletedTodos = completedTodos
```

### 3. 后端：TodoService 新增统计方法

**文件**: [internal/services/todo_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/todo_service.go)

新增两个方法：
- `Count() (int64, error)` — 统计 todo 表总行数
- `CountCompleted() (int64, error)` — 统计 done=true 的 todo 数量

### 4. 后端：新增清空已完成待办方法

**文件**: [internal/services/todo_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/todo_service.go)

新增方法：
- `DeleteCompleted() (int64, error)` — 删除所有 done=true 的待办，返回删除条数

**文件**: [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go)

新增 Wails 绑定方法：
- `ClearCompletedTodos() (string, error)` — 调用 todoService.DeleteCompleted()，返回成功消息

### 5. 前端：loadDataStats() 读取待办统计并渲染

**文件**: [frontend/src/js/data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js#L36-L136)

改动点：
- 变量声明区增加 `totalTodos = 0, completedTodos = 0`
- stats 读取区增加 `totalTodos = stats.total_todos || 0; completedTodos = stats.completed_todos || 0`
- `hasData` 判断增加 `|| totalTodos > 0`
- 信笺 HTML 在「📝 笔记与存储」段落和 `<hr>` 之间插入待办段落：
  ```html
  <hr class="letter-divider">
  <p class="letter-section-title">✓ 待办事项</p>
  <p>
      你共创建了 <strong>${totalTodos}</strong> 个待办事项，
      已完成 <strong>${completedTodos}</strong> 项，
      完成率 <strong>${totalTodos > 0 ? Math.round(completedTodos / totalTodos * 100) : 0}%</strong>。
  </p>
  ```

### 6. 前端：操作列表新增「待办数据」分组

**文件**: [frontend/index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html)

在「AI 数据」分组之后（第 785 行附近）新增：
```html
<!-- 待办数据 -->
<div class="data-action-list" data-group="待办数据">
    <button class="data-action-row" id="clearCompletedTodosBtn">
        <span class="dar-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg></span>
        <span class="dar-body">
            <span class="dar-label">清空已完成待办</span>
            <span class="dar-desc">删除所有已完成的待办事项</span>
        </span>
        <span class="dar-chevron">&rsaquo;</span>
    </button>
</div>
```

### 7. 前端：绑定清空已完成待办事件

**文件**: [frontend/src/js/data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js)

新增 `clearCompletedTodos()` 函数（参考 `clearAISessions()` 模式）：
- 确认对话框确认
- 调用 `window.go.main.App.ClearCompletedTodos()`
- 成功提示
- 刷新 `loadDataStats()`
- 如果当前在待办页面则刷新 `loadTodos()`

在 `index.html` 或 `main.js` 中绑定 `#clearCompletedTodosBtn` 的 click 事件。

### 8. 后端：ResetAll 清空待办数据

**文件**: [internal/services/note_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/note_service.go#L578-L589)

`ResetAll()` 追加清空 todos 表（恢复出厂设置应清空所有数据，包含待办）。

## 任务列表

| # | 任务 | 文件 | 预估复杂度 |
|---|------|------|-----------|
| 1 | DataStats 结构体新增待办字段 | `internal/services/types.go` | ★ |
| 2 | TodoService 新增 Count/CountCompleted/DeleteCompleted | `internal/services/todo_service.go` | ★★ |
| 3 | GetDataStats() 追加待办统计查询 | `app.go` | ★ |
| 4 | 新增 ClearCompletedTodos() Wails 绑定 | `app.go` | ★ |
| 5 | loadDataStats() 读取并渲染待办统计 + 空状态适配 | `frontend/src/js/data-management.js` | ★★ |
| 6 | HTML 新增「待办数据」操作分组 | `frontend/index.html` | ★ |
| 7 | 绑定 clearCompletedTodos 事件 | `frontend/src/js/data-management.js` | ★ |
| 8 | ResetAll() 清空待办数据 | `internal/services/note_service.go` | ★ |

## 依赖关系

- 任务 1、2 无依赖，可并行
- 任务 3 依赖 1、2
- 任务 4 依赖 2
- 任务 5、6、7 无前端依赖，可并行
- 任务 8 独立

## 验证步骤

1. 编译运行 `wails dev`
2. 进入数据管理页 → 信笺中应显示「✓ 待办事项」段落（待办数为 0 时显示"0 个待办事项，已完成 0 项，完成率 0%"）
3. 创建几条待办并完成部分 → 回到数据管理页 → 信笺统计数字正确
4. 点击「清空已完成待办」→ 确认对话框 → 已完成待办被删除，未完成保留
5. 刷新待办清单页 → 完成的已消失，未完成的还在
6. 恢复出厂设置 → 待办数据也被清空
7. 快捷键说明弹窗 Ctrl+6/7/8 不受影响
