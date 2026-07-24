# 待办 FAB 位置调整：移至列表右侧边缘

## 摘要

将 FAB 按钮从 `.todo-container` 的右下角（距右 16px）移到 `.todo-list-wrap`（列表区域）的右侧边缘，使按钮紧贴列表条目右侧，而非悬浮在容器右侧空白区域。

---

## 当前状态

### 当前布局

```
┌────── .todo-container (max-width: 720px, padding: 0 20px) ──────┐
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 待办 3 │ 全部 5 │ 已完成 2   清空                           │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ ▓ 条目 A                                                   │ │
│  │ ▓ 条目 B                                                   │ │
│  │                                                            │ │
│  │                                        [输入框...]    [＋] │ │
│  └────────────────────────────────────────────────────────────┘ │
│                               ↑ 距容器右 16px，在 padding 区域内   │
└──────────────────────────────────────────────────────────────────┘
```

### 当前 FAB 定位方式

```css
.todo-container {
    position: relative; /* FAB 的定位锚点 */
    padding: 0 var(--space-5); /* 0 20px */
}

.todo-fab {
    position: absolute;
    bottom: 16px;
    right: 16px;  /* 距容器 padding 右边缘 16px */
}
```

### 位置计算

- `.todo-container` 内容区域右边缘 = `container_width - 20px`
- `.todo-list-wrap` 内容区域右边缘（考虑 `margin: 0 -4px; padding: 0 4px`）= 同样是 `container_width - 20px`
- 当前 FAB 右边缘 = `container_width - 16px`（距容器 padding 右边缘 16px）
- FAB 比列表条目右边缘**偏右 4px**，悬浮在容器的 padding 空白区

---

## 目标状态

### 新布局

```
┌────── .todo-container ──────────────────────────────────────────┐
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ 待办 3 │ 全部 5 │ 已完成 2   清空                           │ │
│  └────────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ ▓ 条目 A                                                  │ │
│  │ ▓ 条目 B                                                  │ │
│  │                                                          ＋│ │
│  │                                              [输入框...]  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                               ↑ FAB 右边缘对齐列表条目右边缘      │
└──────────────────────────────────────────────────────────────────┘
```

FAB 移至 `.todo-list-wrap` 内部（列表区域内），其右边缘与列表条目右边缘对齐，不再悬浮在右侧空白区。

### 实现策略

FAB 不能直接放入 `.todo-list-wrap`（`overflow-y: auto` 滚动容器），否则会随列表滚动。

**方案：** 在 `.todo-list-wrap` 内部加一个 `position: relative` 的包裹层 `.todo-list-inner`，FAB 和面板 absolute 定位在此层内。因 `.todo-list-inner` 高度撑满 `.todo-list-wrap`（`min-height: 100%`），FAB 位于底部且不超出滚动容器的 padding 框，不会被 `overflow` 裁剪。

---

## 修改内容

### 1. HTML — `frontend/index.html`

**修改位置：** `#viewTodo > .todo-container` 内部

**当前结构：**
```html
<div class="todo-container">
    <div class="todo-filter-bar">...</div>
    <div class="todo-list-wrap">
        <div id="todoList" class="todo-list"><!-- 动态 --></div>
    </div>
    <div id="todoEmpty">...</div>
    <!-- FAB + 面板 -->
    <button class="todo-fab" id="todoFab">...</button>
    <div class="todo-fab-panel" id="todoFabPanel">...</div>
</div>
```

**改为：**
```html
<div class="todo-container">
    <div class="todo-filter-bar">...</div>
    <div class="todo-list-wrap">
        <div class="todo-list-inner">
            <div id="todoList" class="todo-list"><!-- 动态 --></div>
            <button class="todo-fab" id="todoFab">...</button>
            <div class="todo-fab-panel" id="todoFabPanel">...</div>
        </div>
    </div>
    <div id="todoEmpty">...</div>
</div>
```

即：
- 删除 `.todo-container` 末尾的 FAB + 面板
- 在 `.todo-list-wrap` 内添加 `.todo-list-inner` 包裹层
- 将 FAB + 面板移到 `.todo-list-inner` 内部（`.todo-list` 之后）

### 2. CSS — `frontend/src/css/components/todo.css`

| 修改 | 操作 |
|------|------|
| `.todo-container` | 删除 `position: relative` |
| `.todo-list-wrap` | 保持 `overflow-y: auto` 不变（不添加 `position: relative`，避免影响剪裁行为） |
| **新增** `.todo-list-inner` | `position: relative; min-height: 100%;` |
| `.todo-fab` | 保持 `position: absolute; bottom: 16px; right: 16px;` 不变（锚点变为 `.todo-list-inner`，位置不变但容器变了） |
| `.todo-fab-panel` | 保持 `position: absolute; bottom: 20px; right: 68px;` 不变 |
| `.todo-list` | 删除 `padding-bottom: 72px`，改回 `padding-bottom: var(--space-3)`（FAB 不再占用列表底部空间） |

### 3. JS — `frontend/src/main.js`

**无需修改。** `els.todoFab`、`els.todoFabPanel`、`els.todoInput` 的 ID 选择器不变，DOM 树变化不影响选择器结果。

### 4. 不需要改的

- `openTodoInputPanel()` / `closeTodoInputPanel()` — 逻辑不变
- `addTodo()` — 不关心 DOM 位置
- 全局 Escape 处理 — 不变

---

## 验证

1. 构建通过（0 errors）
2. FAB 按钮显示在列表区域的右下角（紧贴条目右侧边缘，不在 padding 空白区）
3. 点击 FAB：面板从左侧滑入，输入框聚焦
4. Enter 提交：条目添加，面板保持打开
5. 滚动列表：FAB 不随内容滚动（因是 `.todo-list-inner` absolute 定位，但需验证 `overflow:auto` 未裁剪）
6. 筛选切换/清空/编辑/删除：不受影响
