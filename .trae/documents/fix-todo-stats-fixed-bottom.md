# 待办清单底部状态栏固定方案

## 当前状态

待办清单视图 (`#viewTodo`) 的 `.todo-stats` 状态栏当前嵌套在 `.todo-container` 内部，作为普通流式元素排在待办列表后面。当待办项很多时，状态栏会随着页面一起滚动，而不是固定在视图底部。

## 原因分析

`#viewTodo` 所在的 `.view` 类在 `.active` 时已经是 `display: flex; flex-direction: column;` 布局。但由于 `.todo-stats` 被包裹在 `.todo-container` 内部，无法利用 flex 布局的底部固定能力。

## 修改方案

只需两处改动：

### 1. HTML 结构调整 (`frontend/index.html`)

将 `.todo-stats` 从 `.todo-container` 内部移出，变成 `#viewTodo` 的直接子元素，放在 `.todo-container` 后面。这样可以利用 `#viewTodo` 的 flex 列布局特性。

**改动前：**
```html
<div id="viewTodo" class="view">
    <div class="view-header">...</div>
    <div class="todo-container">
        <div class="todo-input-wrap">...</div>
        <div class="todo-filter-bar">...</div>
        <div id="todoList" class="todo-list">...</div>
        <div id="todoEmpty" class="todo-empty">...</div>
        <!-- 统计栏在容器内，会随列表滚动 -->
        <div class="todo-stats" id="todoStats">...</div>
    </div>
</div>
```

**改动后：**
```html
<div id="viewTodo" class="view">
    <div class="view-header">...</div>
    <div class="todo-container">
        <div class="todo-input-wrap">...</div>
        <div class="todo-filter-bar">...</div>
        <div id="todoList" class="todo-list">...</div>
        <div id="todoEmpty" class="todo-empty">...</div>
    </div>
    <!-- 统计栏移出容器，作为 flex 子项固定在底部 -->
    <div class="todo-stats" id="todoStats">...</div>
</div>
```

### 2. CSS 调整 (`frontend/src/css/components/todo.css`)

- `.todo-container` — 添加 `flex: 1; overflow-y: auto;` 使其撑满剩余空间并支持滚动
- `.todo-stats` — 添加 `flex-shrink: 0;` 确保不收缩；移除 `margin-top`（由 flex 布局控制间距）；可考虑添加顶部内边距或轻微分隔

## 涉及的文件

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `frontend/index.html` | 修改 | 将 `.todo-stats` 移出 `.todo-container`，放到 `#viewTodo` 的直接子级 |
| `frontend/src/css/components/todo.css` | 修改 | `.todo-container` 增加 `flex: 1; overflow-y: auto;`；`.todo-stats` 底部固定相关样式 |

## 验证方式

1. 添加大量待办项，滚动列表时状态栏应始终固定在视图底部不动
2. 无待办项时，状态栏也固定在底部（而非跟随空状态位置）
3. 筛选切换不影响底部固定行为
