# AI 助手用户消息截断显示功能计划

## 概述

在 AI 助手模块中，当用户消息内容超过指定字符数时，默认折叠显示摘要，用户点击按钮后展开显示全文。切换会话时，所有用户消息恢复为折叠状态。

---

## 现状分析

- **消息渲染入口**：`frontend/src/js/ai-chat.js`，`renderUserMessageWithChips()`（第 2902 行）
- **用户消息文本 DOM**：`.ai-msg-text` div，通过 `document.createTextNode` 写入完整内容（第 2908-2911 行）
- **会话切换**：`switchSession()`（第 1368 行）清空 `messagesInnerEl.innerHTML` 后重新渲染所有消息
- **当前无截断展开功能**：所有用户消息内容完整显示，无折叠/展开交互
- **CSS 位置**：`frontend/src/css/components/ai-chat.css`，用户消息样式在第 99-130 行
- **技术栈**：Vanilla JS + Wails (Go Desktop)，无前端框架

---

## 变更内容

### 1. 新增常量 — `ai-chat.js`

在文件顶部常量区域（第 135 行附近）新增：

```js
const MAX_COLLAPSE_CHARS = 200;   // 用户消息超过此字符数时折叠显示
```

### 2. 修改 `renderUserMessageWithChips()` — `ai-chat.js`

**位置**：第 2902-2929 行

**改动**：
- 在 `textEl`（`.ai-msg-text`）创建后，判断 `content.length > MAX_COLLAPSE_CHARS`
- 若超过阈值：
  - 在 `textEl` 上添加 class `collapsed`，表示截断状态
  - 创建一个展开按钮 `.ai-msg-expand-btn`，内容为"展开全文"
  - 按钮点击时：切换 `textEl` 的 `collapsed` class，并切换按钮文本（"展开全文" ↔ "收起"）

**设计决策**：使用 CSS class 切换控制截断，而非 DOM 内容替换。这样避免重复创建 textNode，展开/收起操作仅切换样式，性能更好。

### 3. 新增 CSS 样式 — `ai-chat.css`

在 `.ai-msg-user` 相关样式区域（第 108 行附近）新增：

```css
/* 用户消息截断折叠 */
.ai-msg-user .msg-content .ai-msg-text.collapsed {
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
    max-height: none;  /* 由 line-clamp 控制行数 */
}

.ai-msg-user .msg-content .ai-msg-expand-btn {
    flex: 0 0 auto;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    font-size: 0.85em;
    color: var(--accent);
    cursor: pointer;
    border: none;
    background: transparent;
    user-select: none;
    white-space: nowrap;
}
.ai-msg-user .msg-content .ai-msg-expand-btn:hover {
    text-decoration: underline;
}
```

### 4. 切换会话时重置折叠状态 — 无需额外修改

**原因**：`switchSession()` 每次都会清空 `messagesInnerEl.innerHTML` 并通过 `addMessage()` → `renderUserMessageWithChips()` 重新创建所有消息 DOM。因此每次切换会话，所有用户消息都会重新渲染，`content.length > MAX_COLLAPSE_CHARS` 的判断会重新执行，自动使所有大消息处于折叠状态。无需额外代码。

---

## 文件修改清单

| 文件 | 修改类型 | 说明 |
|------|---------|------|
| `frontend/src/js/ai-chat.js` | 编辑 | 新增常量 `MAX_COLLAPSE_CHARS`；修改 `renderUserMessageWithChips()` 增加截断逻辑 |
| `frontend/src/css/components/ai-chat.css` | 编辑 | 新增截断折叠样式 + 展开按钮样式 |

---

## 验证步骤

1. 发送一条超过 200 字符的用户消息，验证：
   - 默认只显示 3 行，其余内容隐藏
   - 出现"展开全文"按钮
   - 点击后显示全部内容，按钮变为"收起"
   - 点击"收起"恢复折叠
2. 发送一条少于 200 字符的消息，验证不显示截断
3. 切换会话后，回到原会话，验证所有大消息恢复为折叠状态
4. 分页加载更多历史消息（滚动到顶部加载），验证旧消息也正确应用截断