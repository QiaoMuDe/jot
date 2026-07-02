# AI 对话 - 消息操作按钮折叠（更多菜单）

## 摘要

当消息气泡宽度不足以容纳所有操作按钮时，将按钮折叠到 ⋮（更多）按钮中，点击弹出菜单显示完整操作列表，避免按钮在窄气泡中互相挤压溢出。

## 现状分析

当前操作按钮始终全部显示在消息气泡下方：

**DOM 结构：**
```
div.ai-msg-actions (position: absolute, left:0; right:0; 宽度 = 气泡宽度)
  ├── span.ai-msg-time              (左侧耗时)
  └── div.action-buttons            (flex 行, margin-left: auto 右对齐)
        ├── button[复制]            26×26px
        ├── button[编辑/重发/保存]   26×26px
        ├── button[再生/追问/重发]   26×26px
        └── button[删除]            26×26px
```

- 用户消息气泡：`max-width: 75%`，至少 3 个按钮（复制、编辑、重发、删除）= ~110px
- AI 消息气泡：`max-width: 82%`，至少 5 个按钮（复制、保存、再生、追问、删除）= ~138px
- 当消息只有几个字（如"好的"「收到」「OK」），气泡宽度可能小于按钮总宽度 → 按钮溢出气泡

## 设计

### 折叠策略

- 始终在 `.action-buttons` 末尾添加一个 `.more-btn`（⋮ 图标）
- 挂载后检测：`.action-buttons.scrollWidth > .ai-msg.clientWidth`
- 如果溢出 → 给 `.action-buttons` 加 `.collapsed` 类
- `.collapsed` 状态下：所有常规按钮隐藏，仅 `.more-btn` 可见
- 点击 `.more-btn` → 弹出下拉菜单显示所有操作
- 点击菜单项或外部 → 关闭菜单

### 效果

```
短消息 "好的"（气泡 ~60px）:
┌──────┐
│ 好的  │
└──────┘
  [⋮]              ← 只有更多按钮

点击 ⋮:
┌──────┐
│ 好的  │
└──────┘
  ┌──────────┐
  │ 📋 复制   │  ← 弹出菜单
  │ ✏️ 编辑   │
  │ 📤 重发   │
  │ 🗑 删除   │
  └──────────┘
```

正常宽度消息 → 所有按钮直接显示，行为不变。

## 修改文件

| 文件 | 改动 |
|------|------|
| `frontend/src/js/ai-chat.js` | 新增 `MORE_ICON`、修改 `createMsgActions`、新增折叠检测函数、弹出菜单处理 |
| `frontend/src/css/components/ai-chat.css` | 新增 `.more-btn`、`.collapsed`、`.action-popup` 样式 |

## 具体改动

### JS 改动

#### 1. 新增 `MORE_ICON` 常量

放在 `RESEND_ICON` 之后（约第 2595 行）：

```javascript
const MORE_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="5" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="12" cy="19" r="1.5"/></svg>';
```

三个竖点（⋮）SVG，与现有图标风格一致。

#### 2. 在 `createMsgActions()` 末尾添加 `.more-btn`

在所有按钮之后（删除按钮之后），追加更多按钮：

```javascript
const moreBtn = document.createElement('button');
moreBtn.className = 'more-btn';
moreBtn.innerHTML = MORE_ICON;
moreBtn.title = '更多操作';
moreBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    if (isStreaming) return;
    const msgEl = container.parentElement;
    toggleActionPopup(msgEl, moreBtn);
});
btnWrap.appendChild(moreBtn);
```

#### 3. 新增 `collapseActionsIfNeeded(msgEl)` 函数

在 `createMsgActions` 函数附近新增，负责检测并折叠：

```javascript
function collapseActionsIfNeeded(msgEl) {
    const actions = msgEl.querySelector('.ai-msg-actions');
    const btnWrap = actions?.querySelector('.action-buttons');
    if (!actions || !btnWrap) return;
    
    // 移除之前的折叠状态重新测量
    btnWrap.classList.remove('collapsed');
    
    // 使用 RAF 确保布局已计算
    requestAnimationFrame(() => {
        const availableWidth = actions.clientWidth;
        const buttonsWidth = btnWrap.scrollWidth;
        if (buttonsWidth > availableWidth && buttonsWidth > 60) {
            btnWrap.classList.add('collapsed');
        }
    });
}
```

阈值 `60px` 是为了避免空状态或仅 1 个按钮时也折叠（约 2 个按钮宽度）。

#### 4. 新增 `toggleActionPopup(msgEl, moreBtn)` 函数

创建/切换弹出菜单：

```javascript
function toggleActionPopup(msgEl, moreBtn) {
    // 关闭已存在的弹出菜单
    const existing = msgEl.querySelector('.action-popup');
    if (existing) {
        existing.remove();
        return;
    }
    
    // 收集隐藏按钮信息
    const btnWrap = msgEl.querySelector('.action-buttons');
    if (!btnWrap) return;
    
    const hiddenButtons = [];
    btnWrap.querySelectorAll('button:not(.more-btn)').forEach(btn => {
        hiddenButtons.push({
            html: btn.innerHTML,
            title: btn.title,
        });
    });
    
    if (hiddenButtons.length === 0) return;
    
    // 创建弹出菜单
    const popup = document.createElement('div');
    popup.className = 'action-popup';
    popup.addEventListener('click', (e) => e.stopPropagation());
    
    hiddenButtons.forEach((btnInfo, index) => {
        const item = document.createElement('button');
        item.innerHTML = btnInfo.html;
        item.title = btnInfo.title;
        
        // 重新绑定点击事件 — 从原始按钮列表获取对应索引的按钮并触发 click
        item.addEventListener('click', (e) => {
            e.stopPropagation();
            const allBtns = btnWrap.querySelectorAll('button:not(.more-btn)');
            if (allBtns[index]) {
                allBtns[index].click();
            }
            popup.remove();
        });
        
        popup.appendChild(item);
    });
    
    msgEl.appendChild(popup);
    
    // 关闭：点击外部
    const closeHandler = (ev) => {
        if (!popup.contains(ev.target) && ev.target !== moreBtn) {
            popup.remove();
            document.removeEventListener('click', closeHandler);
        }
    };
    setTimeout(() => document.addEventListener('click', closeHandler), 0);
}
```

**核心思路：** 弹出菜单中的按钮通过 **触发 DOM 中原始按钮的 click 事件** 来执行操作，无需重新绑定所有 handler，也无需维护 action dispatch 表。

#### 5. 在 `createMsgActions()` 的所有调用点后添加折叠检测

共 5 处调用点，在 `appendChild` 后调用 `collapseActionsIfNeeded(msgEl)`：

**调用点 1：** 发送新消息（约第 1725 行）— 用户消息
**调用点 2：** 加载历史消息（约第 1470 行）— 用户消息
**调用点 3：** 加载历史消息（约第 1475 行）— AI 消息
**调用点 4：** 流式完成后（约第 1936 行）— AI 消息
**调用点 5：** 重新发送后（约第 2998 行）— 用户消息

每个调用点类似：
```javascript
msgEl.appendChild(createMsgActions(content, 'user'));
collapseActionsIfNeeded(msgEl);
```

### CSS 改动

在 `ai-chat.css` 中 `.action-buttons` 相关样式区域（约第 1284-1324 行）后追加：

```css
/* 更多按钮（始终隐藏，collapsed 时显示） */
.action-buttons .more-btn {
    display: none;
}
.action-buttons.collapsed .more-btn {
    display: flex;
}
.action-buttons.collapsed > button:not(.more-btn) {
    display: none;
}

/* 弹出菜单 */
.action-popup {
    position: absolute;
    bottom: 100%;
    right: 0;
    margin-bottom: 4px;
    background: var(--card-bg);
    border: 1px solid var(--border);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    padding: 4px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 36px;
}

.action-popup button {
    width: 32px;
    height: 32px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s, color 0.12s;
}

.action-popup button:hover {
    background: var(--hover-bg);
    color: var(--text-primary);
}

.action-popup button:last-child:hover {
    color: #e74c3c;
    background: color-mix(in srgb, #e74c3c 15%, transparent);
}
```

## 不变的行为

- 正常宽度的消息：所有按钮直接显示，和现在完全一致（`.collapsed` 类不添加）
- 按钮的操作逻辑不变：复制、编辑、重发、保存、再生、追问、删除都与原来相同
- 流式传输中：折叠检测照常运行（但 `more-btn` 的 `isStreaming` 检查会阻止操作）
- 暗色主题：继承 `var(--card-bg)` 等 CSS 变量，无需额外适配

## 验证方式

1. 发送一条短消息（如"好"）→ 气泡窄 → 只显示 ⋮ 按钮
2. 点击 ⋮ → 弹出菜单显示所有操作按钮 → 点击任一按钮 → 操作正常执行，菜单关闭
3. 点击 ⋮ 外部 → 菜单关闭
4. 发送正常长度消息 → 所有按钮直接显示，行为不变
5. 加载历史消息（短 + 正常）→ 分别正确折叠/展开
6. 重新发送短消息后 → 新气泡的折叠也正常
