# 批量管理模式添加"全选/取消全选"功能

## 摘要

在批量管理模式的批量操作栏中，新增一个"全选/取消全选"按钮，点击可一键选中当前可见的所有笔记，或取消全选。

## 当前状态分析

批量模式已具备以下能力：
- 进入/退出批量模式（`toggleBatchMode()`）
- 单张笔记选中/取消选中（`toggleNoteSelection(id)`）
- 批量删除选中笔记（`batchDeleteSelected()`）
- 清空选中（`clearSelection()`）
- 计数更新（`updateBatchBar()`）
- 卡片渲染时显示复选框

当前批量栏结构（`index.html#L40-L47`）：
```
[已选 0 条]                    [批量删除] [退出批量]
```

缺少：一键全选/取消全选的能力。

## 改动计划

### 1. HTML — 批量栏新增"全选"按钮

**文件：** `frontend/index.html`

在 `batch-bar-info` 后面、`batch-bar-actions` 前面插入一个"全选"按钮。

```html
<div id="batchBar" class="batch-bar" style="display:none;">
    <div class="batch-bar-left">
        <span class="batch-bar-info">已选 <span id="batchCount">0</span> 条</span>
        <button class="batch-select-all-btn" id="batchSelectAllBtn">全选</button>
    </div>
    <div class="batch-bar-actions">
        <button class="btn btn-sm batch-btn btn-danger" id="batchDeleteBtn">批量删除</button>
        <button class="btn btn-sm batch-btn batch-cancel" id="batchCancelBtn">退出批量</button>
    </div>
</div>
```

说明：
- 用 `.batch-bar-left` 包裹左侧区域（信息文字 + 全选按钮），与右侧操作按钮形成 flex 布局。
- `#batchSelectAllBtn` 按钮文字在"全选"和"取消全选"之间动态切换。

### 2. JS — 引用 + 逻辑函数

**文件：** `frontend/src/main.js`

#### 2a. 添加 DOM 引用

在 `els` 对象中 `batchCancelBtn` 下方新增：
```js
batchSelectAllBtn: $('batchSelectAllBtn'),
```

#### 2b. 新增全选/取消全选函数

```js
/**
 * 全选/取消全选当前可见笔记
 */
function toggleSelectAll() {
    const allIds = state.notes.map(n => n.id);
    const allSelected = allIds.every(id => state.selectedNoteIds.has(id));

    if (allSelected) {
        // 取消全选
        state.selectedNoteIds.clear();
    } else {
        // 全选：将当前所有可见笔记 ID 加入选中集合
        allIds.forEach(id => state.selectedNoteIds.add(id));
    }
    updateBatchBar();
    renderCardGrid();
}
```

#### 2c. 更新 `updateBatchBar()` — 同步按钮文字

```js
function updateBatchBar() {
    const count = state.selectedNoteIds.size;
    els.batchCount.textContent = count;
    // 同步全选按钮文字
    const total = state.notes.length;
    if (total > 0 && count === total) {
        els.batchSelectAllBtn.textContent = '取消全选';
    } else {
        els.batchSelectAllBtn.textContent = '全选';
    }
}
```

#### 2d. 添加事件绑定

在 `initEventListeners()` 的批量模式事件绑定区新增：
```js
els.batchSelectAllBtn.addEventListener('click', toggleSelectAll);
```

### 3. CSS — 全选按钮样式

**文件：** `frontend/src/style.css`

在 `.batch-bar-actions` 样式之后新增：

```css
.batch-bar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.batch-select-all-btn {
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 3px 10px;
  font-size: 0.75rem;
  color: var(--accent);
  cursor: pointer;
  font-family: inherit;
  transition: var(--transition);
  white-space: nowrap;
}

.batch-select-all-btn:hover {
  background: var(--accent-lighter);
  border-color: var(--accent);
}
```

## 影响范围

| 文件 | 改动类型 |
|------|---------|
| `frontend/index.html` | 批量栏 HTML 结构调整 + 新增按钮 |
| `frontend/src/main.js` | els 引用 + 新增 `toggleSelectAll()` + 更新 `updateBatchBar()` + 事件绑定 |
| `frontend/src/style.css` | 新增 `.batch-bar-left` 和 `.batch-select-all-btn` 样式 |

## 设计决策

1. **全选范围**：仅选中当前 `state.notes` 中的笔记（即当前页可见笔记），而非后端全部笔记。这是常见的"全选"行为，与分页场景下"当前页全选"一致。
2. **按钮文字联动**：通过 `selectedNoteIds.size === state.notes.length` 自动判断。手动逐个选中所有笔记后，按钮自动变为"取消全选"；取消选中任意一个后恢复为"全选"。
3. **无额外状态变量**：不需要新增 `allSelected` 状态，通过现有的 `selectedNoteIds` 和 `state.notes` 推导即可。

## 验证

1. `npm run build` 构建通过
2. 进入批量模式 → 点击"全选" → 所有卡片复选框勾选 → 按钮文字变为"取消全选" → 计数显示"已选 N 条"
3. 点击"取消全选" → 所有复选框取消 → 计数归零 → 按钮文字恢复为"全选"
4. 手动逐个选中所有笔记 → 按钮自动变为"取消全选"
5. 退出批量模式后重新进入 → 选中状态清零 → 按钮显示"全选"
