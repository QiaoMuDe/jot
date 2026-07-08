# 待办清单标题居中 + 筛选合并到输入框右侧方案

## 当前状态

1. **标题** — 在 `.view-header` 中，`justify-content: space-between` 布局，返回按钮在左、标题在中间偏左（无右侧元素平衡），不是真正的居中
2. **筛选栏** — 独立一行 `.todo-filter-bar`，在输入框下方，包含"全部/待办/已完成"三个按钮
3. **默认筛选** — `_todoFilter = 'all'`，页面加载时"全部"按钮为 `.active`

## 修改方案

### 1. 标题居中 (`frontend/index.html` + `frontend/src/css/components/todo.css`)

- 给 `#viewTodo .view-header` 增加专用的 CSS，使用 `display: grid; grid-template-columns: 1fr auto 1fr;` 三列布局，使标题严格居中，返回按钮在左、右侧空白列平衡
- HTML 结构不变

### 2. 筛选栏合并到输入框右侧 (`frontend/index.html`)

- 将 `.todo-filter-bar` 从独立行移到 `.todo-input-wrap` 内部，放在 `<input>` 后面
- 移除"全部"按钮，只保留"待办"和"已完成"
- "待办"按钮默认为 `.active`

**改动前：**
```html
<div class="todo-input-wrap">
    <input type="text" id="todoInput" ...>
</div>
<div class="todo-filter-bar">
    <button class="todo-filter-btn active" data-filter="all">全部</button>
    <button class="todo-filter-btn" data-filter="active">待办</button>
    <button class="todo-filter-btn" data-filter="done">已完成</button>
</div>
```

**改动后：**
```html
<div class="todo-input-wrap">
    <input type="text" id="todoInput" ...>
    <div class="todo-filter-bar">
        <button class="todo-filter-btn active" data-filter="active">待办</button>
        <button class="todo-filter-btn" data-filter="done">已完成</button>
    </div>
</div>
```

### 3. CSS 调整 (`frontend/src/css/components/todo.css`)

| 选择器 | 改动 |
|--------|------|
| `#viewTodo .view-header` | 新增：`display: grid; grid-template-columns: 1fr auto 1fr; align-items: center;` 覆盖默认的 flex space-between |
| `.todo-input-wrap` | 改为 `display: flex; align-items: center; gap: 8px;`（之前是垂直 block） |
| `.todo-input` | 添加 `flex: 1;` 撑满剩余空间 |
| `.todo-filter-bar` | 移除 `margin-bottom: 16px;`，改为 `flex-shrink: 0;` |
| `.todo-filter-btn` | 调整为更紧凑的 `padding: 4px 10px; font-size: 0.75rem;` 以适应在输入框旁显示 |

### 4. JS 调整 (`frontend/src/main.js`)

| 行号 | 改动 |
|------|------|
| ~7466 | 将 `let _todoFilter = 'all';` 改为 `let _todoFilter = 'active';` |
| (已有逻辑) | 筛选按钮点击事件 `data-filter` 仍正常工作，"待办"按钮默认 active |

## 涉及的文件

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `frontend/index.html` | 修改 | 筛选栏移入输入框包裹层，删除"全部"按钮 |
| `frontend/src/css/components/todo.css` | 修改 | 标题居中样式、输入行水平布局、筛选按钮紧凑样式 |
| `frontend/src/main.js` | 修改 | 默认筛选从 `'all'` 改为 `'active'` |

## 验证方式

1. 打开待办清单，标题"待办清单"应严格居中，返回按钮在左侧
2. 输入框和筛选按钮在同一行，筛选按钮在输入框右侧
3. 默认只显示"待办"和"已完成"两个分类按钮，"待办"为选中态
4. 切换筛选正常工作，只显示对应状态的待办项
