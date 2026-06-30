# 为 AI 对话区标题"AI 助手"添加双击新建会话功能

## 摘要

为 AI 对话页面顶部视图标题"AI 助手"添加双击（dblclick）事件，行为与侧栏"会话"标题、`+` 按钮一致，均调用 `createSession()` 新建会话。

## 当前状态分析

### 相关文件
- `frontend/index.html` — AI 对话页面的视图头部定义
- `frontend/src/js/ai-chat.js` — AI 对话全部 JS 逻辑

### 现有实现

侧栏"会话"标题已实现双击新建会话（之前添加的功能）：
- `index.html` 中 `<span class="ai-session-sidebar-title" id="aiSessionTitle">会话</span>`
- `ai-chat.js` 中：
  - `sessionTitleEl = document.getElementById('aiSessionTitle')`
  - `sessionTitleEl.addEventListener('dblclick', createSession)`

### 目标元素

`index.html` 第 653 行：
```html
<h2 class="view-title">AI 助手</h2>
```

位于 `#viewAiChat > .view-header` 中，即对话区顶部的标题文字。

## 修改方案

### 修改 1: `frontend/index.html`

给 `AI 助手` 标题添加 `id` 属性：

```html
<h2 class="view-title" id="aiChatTitle">AI 助手</h2>
```

### 修改 2: `frontend/src/js/ai-chat.js`

在 `initAIChat()` 函数的事件绑定区域，获取该元素并绑定 dblclick 事件：

```js
const aiChatTitleEl = document.getElementById('aiChatTitle');
if (aiChatTitleEl) {
    aiChatTitleEl.addEventListener('dblclick', createSession);
}
```

与侧栏标题的双击处理完全一致，不涉及额外逻辑。

## 验证方式

1. 进入 AI 助手页面
2. 双击顶部"AI 助手"标题
3. 确认新建了一个会话，侧栏列表出现新条目
