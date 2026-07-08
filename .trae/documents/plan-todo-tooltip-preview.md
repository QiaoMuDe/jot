# 待办条目悬浮预览 Tooltip 计划

## 概述

为待办清单条目添加悬浮预览 tooltip：鼠标悬停在条目上时，弹出一个精致的卡片显示文本全文，移出时优雅消失。

---

## 当前状态

- 每个 `.todo-item` 内的 `.todo-text` 使用 `white-space: nowrap; overflow: hidden; text-overflow: ellipsis` 截断文本
- 项目中已有 `action-popup`（ai-chat.js）、`context-menu` 等动态悬浮层实现，均使用 `position: fixed` + `getBoundingClientRect()` + `document.body.appendChild` 模式
- 有完整的 CSS 变量系统（`--card-bg`, `--shadow-lg`, `--radius-lg`, `--accent` 等），tooltip 可直接复用
- todo-item 上**没有**绑定过 mouseenter/mouseleave 事件

---

## 设计方案

### UI 风格

- 与项目现有暖色调一致：白色卡片（`--card-bg`）+ 柔和阴影（`--shadow-lg` + `--shadow-xl`）
- 左上角加一条 3px 的 `--accent` 色竖条装饰，呼应已完成条目的 `inset 3px 0 0 var(--accent)` 样式
- 文本使用 `--text-primary`，行高 `1.6`，`word-break: break-word` 允许自动换行
- 圆角 `var(--radius-lg)`（10px）

### 动画

- **显示**：`scale(0.95) + opacity(0)` → `scale(1) + opacity(1)`，时长 150ms，ease-out
- **隐藏**：`opacity(1)` → `opacity(0)`，时长 120ms，ease-in
- 使用 `transform + opacity` 保证性能
- 添加 `transform-origin: var(--origin-x) var(--origin-y)` 使缩放从触发点展开，更自然

---

## 具体改动

### 1. `frontend/src/css/components/todo.css` — 新增 tooltip 样式

在文件末尾添加：

```css
/* ==================== 悬浮预览 ==================== */
.todo-tooltip {
    position: fixed;
    z-index: 1200;
    max-width: 360px;
    padding: 12px 16px;
    border-radius: var(--radius-lg);
    background: var(--card-bg);
    border: 1px solid var(--border);
    box-shadow: var(--shadow-lg), var(--shadow-xl);
    font-size: 0.875rem;
    line-height: 1.6;
    color: var(--text-primary);
    word-break: break-word;
    white-space: pre-wrap;
    pointer-events: none;
    opacity: 0;
    transform: scale(0.95);
    transition: opacity 0.12s ease-in, transform 0.15s ease-out;
    border-left: 3px solid var(--accent);
}

.todo-tooltip.visible {
    opacity: 1;
    transform: scale(1);
}
```

说明：
- `pointer-events: none` — 防止 tooltip 自身触发鼠标事件，避免闪烁
- `border-left: 3px solid var(--accent)` — 左侧强调色装饰
- `white-space: pre-wrap` — 保留换行符，完整显示编辑时输入的多行文本

### 2. `frontend/src/main.js` — 新增事件绑定

在 `editTodo()` 之后（或单独的函数中），添加鼠标悬浮事件。

#### 2a. 创建 `showTodoTooltip` / `hideTodoTooltip` 函数

```javascript
let todoTooltipEl = null;

function showTodoTooltip(item, text) {
    hideTodoTooltip(); // 清除之前的

    const el = document.createElement('div');
    el.className = 'todo-tooltip';
    el.textContent = text;
    document.body.appendChild(el);

    // 触发回流后添加 visible 类以启动动画
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            el.classList.add('visible');
        });
    });

    // 定位
    positionTodoTooltip(item, el);
    todoTooltipEl = el;
}

function positionTodoTooltip(item, el) {
    const rect = item.getBoundingClientRect();
    const tooltipW = el.offsetWidth;
    const tooltipH = el.offsetHeight;
    const gap = 8;

    // 优先显示在条目下方
    let top = rect.bottom + gap;
    let left = rect.left;

    // 如果下方空间不够，显示在上方
    if (top + tooltipH > window.innerHeight - 8) {
        top = rect.top - tooltipH - gap;
    }

    // 水平：左对齐，但保证不超出右边界
    if (left + tooltipW > window.innerWidth - 12) {
        left = window.innerWidth - tooltipW - 12;
    }
    if (left < 12) left = 12;

    el.style.left = left + 'px';
    el.style.top = top + 'px';

    // 设置 transform-origin，使缩放从条目方向展开
    const originY = top > rect.top ? '0%' : '100%';
    el.style.setProperty('--origin-x', '0%');
    el.style.setProperty('--origin-y', originY);
    el.style.transformOrigin = 'var(--origin-x) var(--origin-y)';
}

function hideTodoTooltip() {
    if (todoTooltipEl) {
        todoTooltipEl.classList.remove('visible');
        // 动画结束后移除 DOM
        const el = todoTooltipEl;
        el.addEventListener('transitionend', () => el.remove(), { once: true });
        // 安全兜底：150ms 后强制移除
        setTimeout(() => { if (el.parentNode) el.remove(); }, 200);
        todoTooltipEl = null;
    }
}
```

#### 2b. 在 `renderTodos()` 中绑定事件

在现有 `renderTodos()` 函数的 HTML 模板生成后（第 7530 行 `.join('')` 之后），为每个 `.todo-item` 绑定事件：

```javascript
// 在 renderTodos() 末尾，listEl.innerHTML 赋值之后
listEl.querySelectorAll('.todo-item').forEach(item => {
    const text = item.querySelector('.todo-text')?.textContent || '';
    // 仅当文本被截断（有省略号）或有多行时才显示 tooltip
    // 但简单起见，有内容就显示 tooltip
    if (text) {
        item.addEventListener('mouseenter', () => showTodoTooltip(item, text));
        item.addEventListener('mouseleave', hideTodoTooltip);
    }
});
```

**注意**：直接在 `listEl.innerHTML = ...` 之后绑定事件，因为渲染和事件绑定是同步的，无需等待。

#### 2c. 在 `insertNewTodoItem()` 中为新条目绑定事件

在 `insertNewTodoItem()` 函数末尾（动画监听之后），为新创建的条目绑定事件：

```javascript
// 在 insertNewTodoItem() 末尾
const text = itemEl.querySelector('.todo-text')?.textContent || '';
if (text) {
    itemEl.addEventListener('mouseenter', () => showTodoTooltip(itemEl, text));
    itemEl.addEventListener('mouseleave', hideTodoTooltip);
}
```

### 3. 窗口滚动/尺寸变化时隐藏 tooltip

```javascript
// 在页面初始化区域添加
window.addEventListener('scroll', hideTodoTooltip, true); // 捕获阶段
window.addEventListener('resize', hideTodoTooltip);
```

---

## 影响范围

| 文件 | 改动类型 | 说明 |
|---|---|---|
| `frontend/src/css/components/todo.css` | 新增 | 追加 `.todo-tooltip` 样式 |
| `frontend/src/main.js` | 新增 | 添加 tooltip 函数 + 在 `renderTodos()` 和 `insertNewTodoItem()` 中绑定事件 |

- 不涉及 HTML 模板
- 不涉及后端
- tooltip 使用 `position: fixed` 脱离文档流，不影响现有布局
- `pointer-events: none` 保证不干扰条目本身的 hover 和 click 事件

---

## 验证

1. 鼠标悬停在文本被截断的 todo 条目上 → 弹出预览卡片，显示全文
2. 卡片出现在条目下方（或上方当空间不足时）→ 定位合理，不超出视口
3. 移出条目 → 卡片平滑消失
4. 滚动页面 → tooltip 自动隐藏
5. 双击编辑后保存 → 新文本在 tooltip 中正确显示
6. 新增条目 → 同样有 tooltip 行为
