# 增强通知系统 — 支持动作按钮 + 保存笔记通知带"查看"跳转

## 现状分析

- `NotificationManager`（[notification.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/notification.js)）已有两种通知：
  - `show(msg, type, duration)` — 纯文本 + 关闭按钮，不支持动作按钮
  - `showUndo(msg, onUndo, duration)` — 专门用于"撤销"，硬编码了"撤销"按钮
- `showNotification`（全局函数，L104）只透传 `show`，不支持按钮
- `openEditor`（[main.js#L2160](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L2160)）未暴露到 `window`，不可从外部调用
- "保存为笔记"（[ai-chat.js#L909](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L909)）只调了 `showNotification('笔记已创建', 'success')`，无跳转能力

## 改动计划

### 文件 1: `frontend/src/js/notification.js`

**新增 `showAction()` 方法**（位于 `showUndo` 之后）：

```js
/**
 * 显示带动作按钮的通知
 * @param {string} message - 通知内容
 * @param {string} type - 类型：'success' | 'error' | 'warning' | 'info'
 * @param {Array<{text:string, callback:Function}>} actions - 动作按钮数组
 * @param {number} duration - 自动消失毫秒数（默认 5000）
 */
showAction(message, type, actions, duration = 5000) {
    const el = document.createElement('div');
    el.className = `notification ${type}`;
    const SVGS = window.SVGS;
    const iconSvg = { success: SVGS.checkmark, error: SVGS.windowClose, warning: SVGS.warning, info: SVGS.info };
    el.innerHTML = `
        <span class="notification-icon">${iconSvg[type] || SVGS.info}</span>
        <span class="notification-msg">${this._esc(message)}</span>
    `;

    // 追加动作按钮
    if (Array.isArray(actions)) {
        const actionsContainer = document.createElement('span');
        actionsContainer.className = 'notification-actions';
        actions.forEach(action => {
            const btn = document.createElement('button');
            btn.className = 'notification-action-btn';
            btn.textContent = action.text;
            btn.addEventListener('click', (e) => {
                e.stopPropagation();
                if (typeof action.callback === 'function') action.callback();
                this._dismiss(el);
            });
            actionsContainer.appendChild(btn);
        });
        el.appendChild(actionsContainer);
    }

    // 关闭按钮
    const closeBtn = document.createElement('button');
    closeBtn.className = 'notification-close';
    closeBtn.innerHTML = SVGS.windowClose;
    closeBtn.setAttribute('aria-label', '关闭');
    closeBtn.addEventListener('click', () => this._dismiss(el));
    el.appendChild(closeBtn);

    this.container.appendChild(el);

    const timer = setTimeout(() => this._dismiss(el), duration);
    el._timer = timer;
    return el;
}
```

**新增全局函数**（位于 `window.showNotification` 之后）：

```js
window.showActionNotification = (msg, type, actions, duration) => {
    if (!window.__nm) window.__nm = new NotificationManager();
    window.__nm.showAction(msg, type, actions, duration);
};
```

### 文件 2: `frontend/src/css/components/modals.css`

在 `.notification-undo-btn` 样式块附近新增动作按钮样式（`notification-actions` 容器 + `notification-action-btn` 按钮）：

- **`.notification-actions`** — `display:flex; gap:6px; flex-shrink:0; align-items:center`
- **`.notification-action-btn`** — 复用 `.notification-undo-btn` 的大部分样式（padding/border/radius/font/transition），但颜色使用 `type` 对应的 accent 色（例如 success → `#2ea043`）
  - 不同通知类型下通过父级 `.success`/`.error`/`.warning`/`.info` 选择器设置不同边框色

```css
.notification-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
    align-items: center;
    margin-left: 8px;
}

.notification-action-btn {
    flex-shrink: 0;
    padding: 4px 12px;
    border-radius: var(--radius-md, 8px);
    border: 1px solid var(--accent, #6366f1);
    background: transparent;
    color: var(--accent, #6366f1);
    cursor: pointer;
    font-size: 0.8125rem;
    font-weight: 500;
    transition: background 0.15s, transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1);
    user-select: none;
}

.notification-action-btn:hover {
    background: rgba(var(--accent-rgb, 99,102,241), 0.1);
}

.notification-action-btn:active {
    transform: scale(0.95);
}

/* 按类型适配按钮色 */
.notification.success .notification-action-btn { border-color: #2ea043; color: #2ea043; }
.notification.success .notification-action-btn:hover { background: rgba(46,160,67,0.1); }
.notification.error .notification-action-btn { border-color: #ef4444; color: #ef4444; }
.notification.error .notification-action-btn:hover { background: rgba(239,68,68,0.1); }
.notification.warning .notification-action-btn { border-color: #f59e0b; color: #f59e0b; }
.notification.warning .notification-action-btn:hover { background: rgba(245,158,11,0.1); }
.notification.info .notification-action-btn { border-color: #3b82f6; color: #3b82f6; }
.notification.info .notification-action-btn:hover { background: rgba(59,130,246,0.1); }
```

### 文件 3: `frontend/src/main.js`

在 `window.switchView = switchView;` 附近补充暴露 `openEditor`：

```js
window.openEditor = openEditor;
```

### 文件 4: `frontend/src/js/ai-chat.js`

修改"保存为笔记"的 click handler（L904-L913），从：

```js
const note = await window.go.main.App.SaveAIMessageAsNote(content);
if (note && note.id) {
    window.showNotification?.('笔记已创建', 'success');
}
```

改为：

```js
const note = await window.go.main.App.SaveAIMessageAsNote(content);
if (note && note.id) {
    window.showActionNotification?.('笔记已创建', 'success', [
        { text: '查看', callback: () => { window.switchView('grid'); window.openEditor(note.id, true); } }
    ]);
}
```

## 验证步骤

1. 编辑 `notification.js` — 语法无错误
2. 编辑 `modals.css` — 样式语法正确
3. 编辑 `main.js` — 确认 `window.openEditor` 已添加
4. 编辑 `ai-chat.js` — 确认 action 回调中使用了 `window.switchView` + `window.openEditor`
5. `go build ./...` 后端编译通过
6. `npx vite build` 前端构建通过

## 决策说明

- `showAction` 复用现有通知容器和出场动画，不新增 HTML 结构
- 使用 `.notification-action-btn` 类而非复用 `.notification-undo-btn`，避免与现有点击逻辑耦合
- 按钮颜色随通知类型变化（success 绿的边框，error 红色等）
- "查看"按钮点击后先自动关闭通知（已在 `showAction` 中处理），再切换视图加打开编辑器
