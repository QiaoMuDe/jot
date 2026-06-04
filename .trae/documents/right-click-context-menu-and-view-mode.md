# 右键菜单 + 左击查看 + 精简卡片操作按钮

## 概述
1. 右键笔记卡片弹出菜单（查看/编辑/置顶/删除）
2. 左击笔记默认为**只读查看**模式
3. 卡片右上角只保留置顶按钮，去掉编辑和删除按钮

## 当前状态分析

### 卡片渲染（`renderCardGrid`）
- 左击 `onclick="window.openNote(id)"` → 直接打开编辑器（编辑模式）
- `.card-actions` 区域含 3 个按钮：编辑(✎)、置顶(★/☆)、删除(✕)
- 无右键菜单
- 无只读查看模式

### 编辑器（`openEditor`）
- 仅有编辑/新建两种模式，无只读查看模式
- 编辑器始终显示标题输入框、内容文本域、标签选择器、保存/取消按钮

## 修改方案

### 涉及文件
| 文件 | 操作 |
|------|------|
| `frontend/index.html` | 新增：右键菜单 DOM 结构 |
| `frontend/src/main.js` | 编辑：5 处改动 |
| `frontend/src/style.css` | 新增：右键菜单样式 + 查看器样式 |

---

### 1. `frontend/index.html` — 新增右键菜单 DOM

在 `<div id="app">` 末尾、`<script>` 之前添加：

```html
<!-- 右键菜单 -->
<div id="contextMenu" class="context-menu">
    <div class="context-menu-item" data-action="view">👁 查看</div>
    <div class="context-menu-item" data-action="edit">✎ 编辑</div>
    <div class="context-menu-divider"></div>
    <div class="context-menu-item" data-action="pin">📌 置顶</div>
    <div class="context-menu-divider"></div>
    <div class="context-menu-item danger" data-action="delete">✕ 删除</div>
</div>
```

### 2. `frontend/src/main.js` — 卡片渲染修改（`renderCardGrid`）

**a)** 修改左击事件：`onclick="window.openNote(...)"` → `onclick="window.viewNote(...)"`

**b)** 添加右击事件：在 `note-card` div 上添加 `oncontextmenu="event.preventDefault(); window.showContextMenu(event, ${note.id})"`

**c)** 精简 `.card-actions`：只保留置顶按钮，移除编辑和删除按钮

```
旧：3 个按钮（编辑 + 置顶 + 删除）
新：1 个按钮（置顶）
```

### 3. `frontend/src/main.js` — 新增全局函数

**a) `window.viewNote(id)`** — 只读查看笔记
- 调用改造后的 `openEditor(id, true)`

**b) `window.showContextMenu(event, noteId)`** — 显示右键菜单
- 阻止默认右键菜单
- 将 `#contextMenu` 定位到鼠标位置
- 使用 `data-note-id` 属性存储当前笔记 ID
- 显示菜单

**c) `window.handleContextAction(action)`** — 处理菜单点击
- `view`: 调用 `window.viewNote(noteId)`
- `edit`: 调用 `window.openNote(noteId)` (编辑模式)
- `pin`: 调用 `window.togglePin(noteId)`
- `delete`: 调用 `window.deleteNote(noteId)`

**d) `hideContextMenu()`** — 隐藏右键菜单

### 4. `frontend/src/main.js` — 修改 `openEditor` 支持只读模式

将 `openEditor(noteId)` 改为 `openEditor(noteId, readOnly = false)`：

当 `readOnly = true` 时：
- 标题改为 "查看笔记"
- 标题输入框 `readonly`（添加 `.editor-input-readonly` 类）
- 内容文本域 `readonly`（添加 `.editor-textarea-readonly` 类）
- 隐藏标签选择器（标签不可编辑）
- 隐藏保存按钮、取消按钮
- 关闭按钮从 "✕" 改为 "关闭"（或保持 ✕ 但不可保存）

当 `readOnly = false` 时：保持现有编辑/新建行为不变

### 5. `frontend/src/main.js` — 事件绑定新增

在 `initEventListeners()` 中添加：
- 点击页面任意位置关闭右键菜单：`document.addEventListener('click', hideContextMenu)`
- 右键菜单项点击事件：使用事件委托处理 `.context-menu-item` 的点击

### 6. `frontend/src/style.css` — 新增样式

**右键菜单样式：**
```css
.context-menu {
    position: fixed;
    background: #fff;
    border-radius: 10px;
    box-shadow: 0 4px 20px rgba(0,0,0,0.15);
    padding: 6px;
    min-width: 160px;
    z-index: 2000;
    display: none;
}
.context-menu.active { display: block; }
.context-menu-item {
    padding: 8px 12px;
    border-radius: 6px;
    font-size: 13px;
    cursor: pointer;
    color: #334155;
    transition: background 0.1s;
}
.context-menu-item:hover { background: #f1f5f9; }
.context-menu-item.danger { color: #ef4444; }
.context-menu-item.danger:hover { background: #fef2f2; }
.context-menu-divider {
    height: 1px;
    background: #e2e8f0;
    margin: 4px 6px;
}
```

**查看器只读样式：**
```css
.editor-input-readonly,
.editor-textarea-readonly {
    background: #f8fafc;
    cursor: default;
    border-color: transparent;
}
.editor-textarea-readonly {
    resize: none;
}
```

### 7. `frontend/src/style.css` — 移除多余样式

移除 `.card-action-btn.delete:hover` 相关的旧样式（删除按钮已不存在）

## 验证步骤
1. 左击笔记卡片 → 打开只读查看器（输入框不可编辑，无保存按钮）
2. 右键笔记卡片 → 弹出菜单（查看/编辑/置顶/删除）
3. 点击菜单中的"编辑" → 打开编辑器（可编辑）
4. 点击菜单中的"置顶" → 切换置顶状态
5. 点击菜单中的"删除" → 确认后删除
6. 点击页面其他区域 → 右键菜单关闭
7. 卡片右上角 hover 只显示置顶按钮
