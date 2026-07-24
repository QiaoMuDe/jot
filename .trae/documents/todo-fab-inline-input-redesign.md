# 待办页面 FAB + 内联弹出输入框 重构计划

## 摘要

将待办页面顶部固定输入栏替换为右下角 FAB 按钮 + 点击后向左弹出内联输入框，保持所有现有业务逻辑不变，仅改造输入交互方式。

---

## 当前状态分析

### 现有布局（`frontend/index.html` #viewTodo 区块）

```
┌─ #viewTodo ──────────────────────────┐
│  ← 返回     待办清单                  │
│  ┌──────────────────────────────────┐ │
│  │ [ 输入框............. ] [添加]   │ │  ← todo-input-wrap（将被移除）
│  ├──────────────────────────────────┤ │
│  │ 待办 3 │ 全部 5 │ 已完成 2  清空 │ │  ← todo-filter-bar
│  ├──────────────────────────────────┤ │
│  │  ▓ 条目 A                        │ │
│  │  ▓ 条目 B                        │ │
│  │  ▓ 条目 C                        │ │
│  │  ───────                        │ │  ← todo-list + todo-list-wrap
│  └──────────────────────────────────┘ │
└──────────────────────────────────────┘
```

### 现有 HTML 结构

```
#viewTodo
  ├── .view-header (← 返回 + 标题)
  └── .todo-container
        ├── .todo-input-wrap ← **将移除**
        ├── .todo-filter-bar  ← 保持不变
        ├── .todo-list-wrap
        │     └── #todoList
        └── #todoEmpty
```

### 相关文件

| 文件 | 行范围 | 说明 |
|------|--------|------|
| `frontend/index.html` | 1643-1698 | #viewTodo HTML 结构 |
| `frontend/src/main.js` | ~360 | `els.todoInput` 等引用 |
| `frontend/src/main.js` | 5569-5592 | todoInput 键盘/事件绑定 |
| `frontend/src/main.js` | 641 | switchView 中的 focus() |
| `frontend/src/main.js` | 8569-8655 | `addTodo()` 函数 |
| `frontend/src/css/components/todo.css` | 28-110 | `.todo-input-wrap` 全套样式 |

### 核心设计约束

1. **保持 `els.todoInput` 引用不变** — 所有现有 JS（`addTodo`, 键盘事件, `autoResizeTodoInput`, `switchView` 聚焦）通过 `els.xxx` 或 `$('todoInput')` 引用此 textarea
2. **keep `todo-input` class** — CSS 涉及 `.todo-input`, `::-webkit-scrollbar` 等样式，复用
3. **空状态文本更新** — 当前 "在上方输入框中添加新待办吧" 需要改为 "点击右下角 + 添加新待办"

---

## 目标设计

### 新布局示意

```
┌─ #viewTodo ───────────────────────────┐
│  ← 返回     待办清单                   │
│  ┌───────────────────────────────────┐ │
│  │ 待办 3 │ 全部 5 │ 已完成 2   清空  │ │
│  ├───────────────────────────────────┤ │
│  │  ▓ 条目 A                         │ │
│  │  ▓ 条目 B                         │ │
│  │                                   │ │
│  │                    [ 输入框....] ＋│ │  ← 右下角：FAB + 左侧弹出输入
│  └───────────────────────────────────┘ │
└───────────────────────────────────────┘
```

### 交互行为

| 行为 | 说明 |
|------|------|
| **默认态** | 右下角仅显示 FAB（"+" 图标），页面干净无输入栏 |
| **点击 FAB** | 输入面板从 FAB 左侧弹性滑入，FAB 图标变为 "✕" |
| **输入添加** | Enter 提交添加（保持打开，不清空焦点），Ctrl+Enter 换行 |
| **关闭方式** | ① 再次点击 FAB（"✕"） ② 按 Escape 键 ③ 点击面板外部区域 |
| **切换视图** | 离开 todo 视图时自动收起面板 |
| **自动聚焦** | 打开面板时自动聚焦到输入框（复用现有 `setTimeout` 逻辑） |

### 不支持的功能（保持不变）

- 现有 `addTodo()` 的连续添加逻辑不变（添加后不清空输入，保留焦点）
- 现有 `autoResizeTodoInput()` 多行扩展不变
- 现有筛选/清空/编辑/删除/完成切换等功能不变

---

## 具体修改方案

### 1. HTML — `frontend/index.html`

#### 1a. 移除 `.todo-input-wrap`
删除第 1649-1659 行的全部内容（`<div class="todo-input-wrap">... </div>`）。

#### 1b. 添加 FAB + 输入面板
在 `.todo-container` 末尾（`#todoEmpty` 后）、`</div>` 前，插入：

```html
<!-- 浮动添加按钮 + 内联输入面板 -->
<button class="todo-fab" id="todoFab" title="添加待办">
    <svg class="todo-fab-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
    </svg>
    <svg class="todo-fab-close" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none;">
        <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
    </svg>
</button>
<div class="todo-fab-panel" id="todoFabPanel">
    <textarea id="todoInput" class="todo-input" placeholder="添加待办事项，Enter 提交" rows="1" autocomplete="off"></textarea>
</div>
```

关键点：
- `textarea#todoInput` ID 与原输入框一致 → `els.todoInput` 自动生效
- `class="todo-input"` 复用现有输入框样式
- 两个 SVG 图标交替显示（`+` ↔ `✕`）

#### 1c. 更新空状态文字
第 1695 行：`"在上方输入框中添加新待办吧"` → `"点击右下角 + 添加新待办"`

### 2. CSS — `frontend/src/css/components/todo.css`

#### 2a. 移除旧输入区样式
删除或注释以下 CSS 区块：
- `.todo-input-wrap`（第 28-45 行）
- `.todo-add-btn`（第 86-110 行）

`.todo-input` 核心样式（第 47-83 行）**保留**（复用为面板内输入框）。

#### 2b. 为 `.todo-container` 添加相对定位
```css
.todo-container {
    /* 现有... */
    position: relative;  /* 新增：作为 FAB 的定位锚点 */
}
```

#### 2c. 添加 FAB 样式

```css
/* ==================== 浮动添加按钮 (FAB) ==================== */
.todo-fab {
    position: absolute;
    bottom: 16px;
    right: 16px;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 50%;
    background: var(--accent);
    color: #fff;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow-md);
    transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
    z-index: 10;
}

.todo-fab:hover {
    transform: scale(1.08) translateY(-1px);
    box-shadow: var(--shadow-elevated);
}

.todo-fab:active {
    transform: scale(0.95);
}

.todo-fab svg {
    transition: transform 0.25s var(--anim-easing-spring);
}

.todo-fab.open svg {
    transform: rotate(45deg);  /* "+" 旋转 45° 呈现 ✕ 效果 */
}
```

设计思路：
- 使用单 SVG（"+" 图标），`.open` 态下旋转 45° 呈现 ✕ 效果，无需交替两个图标
- 44px 大小与首页 FAB 一致，保持视觉统一
- `z-index: 10` 确保在列表项之上

#### 2d. 添加输入面板样式

```css
/* 内联输入面板 */
.todo-fab-panel {
    position: absolute;
    bottom: 20px;
    right: 68px;  /* FAB 右侧 + 间距 */
    width: 320px;
    background: var(--card-bg);
    border: 1.5px solid var(--border);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-elevated);
    padding: var(--space-2);
    opacity: 0;
    transform: translateX(16px);
    pointer-events: none;
    transition: opacity 0.2s ease, transform 0.25s var(--anim-easing-out);
    z-index: 9;
}

.todo-fab-panel.open {
    opacity: 1;
    transform: translateX(0);
    pointer-events: auto;
}

.todo-fab-panel .todo-input {
    min-height: 36px;
    font-size: 0.9rem;
}
```

输入面板从右侧滑入（`translateX(16px) → translateX(0)`），配合 `opacity` 淡入。
`pointer-events: none` → `auto` 确保关闭时不可交互。

#### 2e. 调整列表容器底部间距

```css
.todo-list {
    /* ...现有 */
    padding-bottom: 72px;  /* 增加底部空间，避免列表被 FAB 遮挡 */
}
```

### 3. JS — `frontend/src/main.js`

#### 3a. els 注册 — 新增 FAB 和面板引用

在 `els` 对象中（第 359-363 行附近）新增：
```js
todoFab: $('todoFab'),
todoFabPanel: $('todoFabPanel'),
```

保持 `todoInput: $('todoInput')` 不变（新 textarea ID 相同）。

#### 3b. 移除旧输入区事件绑定

移除第 5569-5592 行的事件绑定：
```js
// 删除以下整个块
if (els.todoInput) {
    els.todoInput.addEventListener('keydown', (e) => { ... });
    els.todoInput.addEventListener('input', autoResizeTodoInput);
}
els.todoAddBtn?.addEventListener('click', addTodo);
```

#### 3c. 替换为 FAB + 面板事件绑定

```js
// 待办 FAB + 内联输入面板
if (els.todoFab && els.todoFabPanel) {
    // FAB 点击切换面板
    els.todoFab.addEventListener('click', toggleTodoInputPanel);
    
    // 面板内输入框键盘事件
    const panelInput = els.todoFabPanel.querySelector('.todo-input');
    if (panelInput) {
        panelInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.ctrlKey && !e.shiftKey) {
                e.preventDefault();
                addTodo();
            } else if (e.key === 'Enter' && e.ctrlKey) {
                e.preventDefault();
                // 插入换行 + auto-resize（复用原逻辑）
                const start = panelInput.selectionStart;
                const end = panelInput.selectionEnd;
                const val = panelInput.value;
                panelInput.value = val.substring(0, start) + '\n' + val.substring(end);
                panelInput.selectionStart = panelInput.selectionEnd = start + 1;
                autoResizeTodoInput();
            } else if (e.key === 'Escape') {
                closeTodoInputPanel();
            }
        });
        
        panelInput.addEventListener('input', autoResizeTodoInput);
    }
    
    // 点击面板外部关闭
    document.addEventListener('click', (e) => {
        if (!els.todoFabPanel.classList.contains('open')) return;
        const target = e.target;
        if (!els.todoFabPanel.contains(target) && !els.todoFab.contains(target)) {
            closeTodoInputPanel();
        }
    });
}

// 辅助函数：打开面板
function openTodoInputPanel() {
    els.todoFabPanel.classList.add('open');
    els.todoFab.classList.add('open');
    setTimeout(() => els.todoInput?.focus(), 100);
}

// 辅助函数：关闭面板
function closeTodoInputPanel() {
    els.todoFabPanel.classList.remove('open');
    els.todoFab.classList.remove('open');
}
```

#### 3d. 更新 `switchView` 中的聚焦行为

第 641-642 行：
```js
// 原：setTimeout(() => els.todoInput?.focus(), 100);
// 改为：
setTimeout(() => openTodoInputPanel(), 100);
```

#### 3e. 调整 `addTodo` 中的输入清空逻辑

第 8576 行 `input.value = '';` 后面添加面板保持打开状态的逻辑（不需要修改，面板默认保持打开）。

### 4. 不需要改的代码

| 模块 | 原因 |
|------|------|
| `addTodo()` 主体逻辑 | 只读 `els.todoInput.value`，不关心输入框位置 |
| `renderTodos()` | 与输入无关 |
| `updateTodoStats()` / `updateTodoStatsAfterAdd()` | 与输入无关 |
| `buildTodoItemHTML()` / `insertNewTodoItem()` | 与输入无关 |
| `.todo-filter-bar` / `.todo-clear-btn` | 筛选和清空逻辑不变 |
| 动画库（`todoStaggerEnter` 等）| 未改动 |

---

## 实施步骤

| 步骤 | 文件 | 操作 |
|------|------|------|
| 1 | `index.html` | 删除 `.todo-input-wrap` |
| 2 | `index.html` | 在 `.todo-container` 末尾添加 FAB + 面板 HTML |
| 3 | `index.html` | 更新空状态提示文字 |
| 4 | `todo.css` | 删除 `.todo-input-wrap` 和 `.todo-add-btn` 样式 |
| 5 | `todo.css` | 为 `.todo-container` 添加 `position: relative` |
| 6 | `todo.css` | 新增 `.todo-fab` / `.todo-fab-panel` 样式块 |
| 7 | `todo.css` | 调整 `.todo-list` 的 `padding-bottom` |
| 8 | `main.js` | `els` 注册 `todoFab` / `todoFabPanel` |
| 9 | `main.js` | 移除旧的 `els.todoInput` 键盘/点击事件绑定 |
| 10 | `main.js` | 新增 FAB 点击 + 面板键盘 + 外部点击关闭事件绑定 |
| 11 | `main.js` | 新增 `openTodoInputPanel()` / `closeTodoInputPanel()` 辅助函数 |
| 12 | `main.js` | 更新 `switchView` case 'todo' 中的聚焦逻辑 |
| 13 | 全项目 | `npm run build` 验证构建 |

---

## 验证

1. 构建通过（0 errors）
2. 进入待办页面：自动弹出输入面板，FAB 呈 ✕ 态
3. 输入文本按 Enter：条目添加，输入框保留打开，不清空焦点
4. 输入文本按 Ctrl+Enter：正常换行
5. 点击 FAB（✕）：面板关闭，FAB 恢复为 +
6. 按 Escape：面板关闭
7. 点击面板外部区域：面板关闭
8. 连续添加多条：每次 Enter 后焦点保持，可连续输入
9. 筛选/清空/编辑/删除/标记完成：功能不受影响
10. 空状态文字：显示 "点击右下角 + 添加新待办"
