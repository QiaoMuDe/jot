# 批量置顶/取消置顶功能计划

## 需求概述
在批量模式下，对已选中的笔记支持批量置顶和取消置顶操作。

## 变更清单

### 1. 后端: `internal/services/note_service.go`
**新增函数** `BatchPinNotes(ids []uint, pin bool) error`
- 使用 `db.Model(&models.Note{}).Where("id IN ?", ids).UpdateColumn("pinned", pin)` 
- 沿用 `TogglePin` 的 `UpdateColumn` 方式，避免更新 `UpdatedAt`
- 返回 `error`

### 2. 后端: `app.go`
**新增绑定函数** `BatchPinNotes(noteIDs []uint, pin bool) error`
- 调用 `noteService.BatchPinNotes(noteIDs, pin)`
- 格式与 `BatchDeleteNotes` 一致（第 160-163 行）

### 3. 前端: `index.html` (第 44-49 行)
在批量操作栏 `<div class="batch-bar-actions">` 中新增一个按钮：
```html
<button class="btn btn-sm batch-btn" id="batchPinBtn">置顶</button>
```
- 放在 `+标签` 按钮前面（或后面，和 `+标签`/`-标签` 放一起）
- 使用已有 `.batch-btn` 样式，无需新增 CSS

### 4. 前端: `frontend/src/main.js`
**a. els 引用**（第 276-284 行区域）
- 新增 `batchPinBtn: $('batchPinBtn')`

**b. 新增函数** `batchPinSelected()`
- 读取 `state.selectedNoteIds` 获取选中笔记 ID 数组
- 判断选中笔记的 pin 状态：
  - 如果至少有一条未置顶 → `pin = true`（全部置顶）
  - 如果全部已置顶 → `pin = false`（全部取消置顶）
- 调用 `window.go.main.App.BatchPinNotes(ids, pin)`
- 成功后批量更新 `state.notes` 中对应笔记的 `pinned` 字段
- 调用 `clearSelection()` + `renderCardGrid('none')`
- 显示 `nm.showUndo()` 撤销提示

**c. `updateBatchBar()` 函数**（第 2736-2748 行）
- 新增逻辑：更新 `batchPinBtn` 按钮文字
  - 选中笔记全部已置顶 → 文字「取消置顶」
  - 否则 → 文字「置顶」
  - 无选中时保留「置顶」

**d. 事件绑定**（第 3148-3154 行区域）
- 新增 `els.batchPinBtn.addEventListener('click', batchPinSelected)`

## 交互流程
```
用户进入批量模式 → 选中笔记 → 点击「置顶」/「取消置顶」
  → batchPinSelected()
    → 判断选中笔记状态（有无未置顶）
    → 调用后端 BatchPinNotes(ids, pin)
    → 本地更新 state.notes
    → clearSelection() + renderCardGrid('none')
    → 显示撤销通知
```

## 验证
1. 选中未置顶笔记 → 按钮显示「置顶」→ 点击后全部变为 ★
2. 选中已置顶笔记 → 按钮显示「取消置顶」→ 点击后全部变为 ☆
3. 混合选中 → 按钮显示「置顶」→ 点击后全部置顶
4. 无选中时 → 按钮不可操作（CSS 已有 `.batch-btn` 样式，无需额外处理）
