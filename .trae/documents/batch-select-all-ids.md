# 批量选择：全选时加载所有笔记 ID

## 摘要

当前「全选」仅选中当前页（默认 20 条）的笔记，批量删除一次只能删一页。改为全选时从后端获取所有未删除笔记的 ID，支持一次选择全部、一次删除全部。

## 改动

### 1. 后端：NoteService 新增 GetAllIDs

**文件：** `internal/services/note_service.go`

在 `GetAll` 方法之后新增：

```go
// GetAllIDs 获取所有未删除笔记的 ID 数组
func (s *NoteService) GetAllIDs() ([]uint, error) {
	var ids []uint
	if err := s.db.Model(&models.Note{}).
		Where("deleted_at IS NULL").
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
```

使用 `Pluck("id")` 只查询单列，性能最优。

### 2. 后端：App 层新增绑定方法

**文件：** `app.go`

在 `GetNotes` 方法之后新增：

```go
// GetAllNoteIDs 获取所有未删除笔记的 ID 数组
func (a *App) GetAllNoteIDs() ([]uint, error) {
	return a.noteService.GetAllIDs()
}
```

### 3. 前端：更新 toggleSelectAll

**文件：** `frontend/src/main.js`

```js
function toggleSelectAll() {
    const allSelected = state.selectedNoteIds.size === state.totalAllNotes;

    if (allSelected) {
        // 取消全选
        state.selectedNoteIds.clear();
    } else {
        // 全选：先从后端拉取所有 ID（如可用），再塞入选中的 Set
        selectAllIds();
    }
    updateBatchBar();
    renderCardGrid();
}

/**
 * 全选：获取所有未删除笔记的 ID
 */
async function selectAllIds() {
    let ids = [];
    if (window.go && window.go.main && window.go.main.App && window.go.main.App.GetAllNoteIDs) {
        try {
            ids = await window.go.main.App.GetAllNoteIDs();
        } catch (err) {
            console.error('获取全部 ID 失败，降级为当前页:', err);
            ids = state.notes.map(n => n.id);
        }
    } else {
        ids = state.notes.map(n => n.id);
    }
    ids.forEach(id => state.selectedNoteIds.add(id));
}
```

同时新增一个 `state.totalAllNotes` 状态变量，在 `loadNotes()` 中同步更新：

```js
// 在 loadNotes() 中，result 分支里设置：
state.totalAllNotes = result.total || 0;
```

以及 Mock 降级分支：

```js
state.totalAllNotes = state.notes.length;
```

并在 `resetPagination()` 中重置：

```js
state.totalAllNotes = 0;
```

`updateBatchBar()` 也更新为使用 `state.totalAllNotes`：

```js
function updateBatchBar() {
    const count = state.selectedNoteIds.size;
    els.batchCount.textContent = count;
    // 同步全选按钮文字
    const total = state.totalAllNotes || state.notes.length;
    if (els.batchSelectAllBtn) {
        if (total > 0 && count === total) {
            els.batchSelectAllBtn.textContent = '取消全选';
        } else {
            els.batchSelectAllBtn.textContent = '全选';
        }
    }
}
```

### 4. 初始化 state 对象

**文件：** `frontend/src/main.js`

在 `state` 定义中添加：

```js
totalAllNotes: 0,
```

## 影响范围

| 文件                                  | 改动                                                                   |
| ----------------------------------- | -------------------------------------------------------------------- |
| `internal/services/note_service.go` | 新增 `GetAllIDs()` 方法                                                  |
| `app.go`                            | 新增 `GetAllNoteIDs()` 绑定方法                                            |
| `frontend/src/main.js`              | 更新 `toggleSelectAll()` + 新增 `selectAllIds()` + `state.totalAllNotes` |

## 设计决策

1. **后端只查 ID，不查全部笔记**：`Pluck("id")` 传输极小，笔记量大（<10 万条）也无压力
2. **降级策略**：后端 API 不可用时，回退到当前页 `state.notes` 的 ID（与现状一致）
3. **全选判断依据**：`selectedNoteIds.size === state.totalAllNotes` 而不是 `state.notes.length`，全选按钮文字能正确反映是否已选所有笔记
4. **Wails 绑定**：无需手动运行 `wails generate module`，前端直接通过 `window.go.main.App.GetAllNoteIDs()` 调用

## 验证

1. `go build` 编译通过
2. `npm run build` 前端构建通过
3. 进入批量模式 → 全选 → 所有笔记 ID 从后端拉取 → 按钮变为「取消全选」
4. 批量删除所有笔记 → 正常删除所有

