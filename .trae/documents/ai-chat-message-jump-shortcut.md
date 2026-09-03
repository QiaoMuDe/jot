# AI 助手消息列表 — 用户消息快捷跳转

## 概述

为 AI 助手消息列表添加快捷键 `Alt+↑` / `Alt+↓`，让用户在当前会话的消息列表中快速跳转到上一条/下一条**用户消息**（即 role='user' 的消息气泡）。

## 当前状态分析

### 消息 DOM 结构
- 消息列表容器：`#aiChatMessages`（`.ai-chat-messages`），溢出滚动
- 消息渲染在 `.ai-chat-messages-inner` 内，每条消息为 `<div class="ai-msg ai-msg-user">` 或 `ai-msg-assistant`
- 用户消息可通过 `.ai-msg-user` 选择器直接查询
- 每条消息元素有 `data-msgId` 属性存储消息 ID

### 分页加载机制（需适配）
- 切换会话时，只加载最近 6 条消息（`LoadAISessionMessagesPaginated(id, 6, 0)`）
- 滚动到列表顶部时，自动加载更早的消息（一次 6 条），并 **prepend** 到 DOM 顶部
- `_oldestMsgId` 记录当前已加载的最旧消息 ID，`_oldestMsgId === 0` 表示全部加载完毕
- `_loadingMore` 标志防止重复加载
- 因此 DOM 中任何时候只包含**部分已加载的消息**，跳转操作需要在已加载的范围内进行

### 现有基础设施
- 视图切换：`window.state.currentView` 追踪当前视图（`'ai-chat'` = AI 助手页）
- 流式状态：`isStreaming` 表示正在输出
- 通知 API：`window.showNotification(message, type)`
- 全局键盘监听：已有 `document.addEventListener('keydown', ...)` 模式

## 修改计划

### 涉及文件

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `frontend/src/js/ai-chat.js` | 新增 ~50 行 | 快捷键监听 + 跳转逻辑（含分页适配） |
| `frontend/src/css/components/ai-chat.css` | 新增 ~15 行 | 跳转目标消息的闪烁高亮动画 |

---

### 1. 快捷键

`Alt+↑` → 跳转到上一条用户消息
`Alt+↓` → 跳转到下一条用户消息

- Alt 键不会干扰输入框打字（Enter/Ctrl+Enter 已在输入框内处理）
- 不与现有快捷键冲突（现有 Escape 关闭菜单、Enter 发送等）
- 方向键直观：↑ 为向上/前一条，↓ 为向下/后一条

---

### 2. 前端逻辑（ai-chat.js）

#### 2.1 新增模块级变量

```javascript
let _jumpDebounceTimer = null;    // 跳转防抖定时器
let _jumpHighlightTimer = null;   // 高亮清除定时器
```

#### 2.2 新增 `jumpToUserMessage(direction)` 函数

```javascript
/**
 * 跳转到上一条/下一条用户消息
 * @param {number} direction - 1 向下（Alt+↓），-1 向上（Alt+↑）
 */
function jumpToUserMessage(direction) {
    // 防抖：300ms 内禁止重复触发
    if (_jumpDebounceTimer) return;
    _jumpDebounceTimer = setTimeout(() => { _jumpDebounceTimer = null; }, 300);

    // 流式输出中禁止跳转
    if (isStreaming) {
        window.showNotification?.('回复进行中，无法跳转', 'warning');
        return;
    }

    const userMessages = messagesInnerEl.querySelectorAll('.ai-msg-user');
    if (userMessages.length === 0) {
        window.showNotification?.('没有用户消息可供跳转', 'info');
        return;
    }

    // 找到当前滚动位置最近的用户消息索引
    const container = messagesEl;
    const containerRect = container.getBoundingClientRect();
    const containerCenter = containerRect.top + containerRect.height / 2;

    let currentIndex = -1;
    let minDist = Infinity;
    userMessages.forEach((el, i) => {
        const rect = el.getBoundingClientRect();
        const elCenter = rect.top + rect.height / 2;
        const dist = Math.abs(elCenter - containerCenter);
        if (dist < minDist) {
            minDist = dist;
            currentIndex = i;
        }
    });

    // 计算目标索引（循环跳转：首尾相连）
    const total = userMessages.length;
    let targetIndex = (currentIndex + direction + total) % total;

    // 滚动到目标消息
    const targetEl = userMessages[targetIndex];
    targetEl.scrollIntoView({ block: 'center', behavior: 'smooth' });

    // 清除旧高亮
    document.querySelectorAll('.ai-msg-jump-target').forEach(el => el.classList.remove('ai-msg-jump-target'));
    if (_jumpHighlightTimer) { clearTimeout(_jumpHighlightTimer); _jumpHighlightTimer = null; }

    // 添加高亮闪烁动画
    targetEl.classList.add('ai-msg-jump-target');
    _jumpHighlightTimer = setTimeout(() => {
        targetEl.classList.remove('ai-msg-jump-target');
        _jumpHighlightTimer = null;
    }, 1500);
}
```

#### 2.3 在 `initAIChat()` 中注册全局键盘监听

在 [ai-chat.js#L1240-L1248](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1240-L1248) 的 Escape 监听之后，新增 Alt+↑/↓ 监听：

```javascript
// Alt+↑/↓ 跳转用户消息（仅 AI 助手视图有效）
document.addEventListener('keydown', (e) => {
    if ((e.key === 'ArrowUp' || e.key === 'ArrowDown') && e.altKey && !e.ctrlKey && !e.metaKey) {
        // 仅在 AI 助手视图激活时生效
        if (window.state?.currentView !== 'ai-chat') return;
        // 输入框/搜索框聚焦时禁用（避免干扰打字）
        if (e.target?.tagName === 'TEXTAREA' || e.target?.tagName === 'INPUT') return;
        e.preventDefault();
        jumpToUserMessage(e.key === 'ArrowDown' ? 1 : -1);
    }
});
```

---

### 3. CSS 样式（ai-chat.css）

新增跳转高亮动画：

```css
/* ── 用户消息跳转高亮 ── */
.ai-msg-jump-target {
    animation: ai-msg-jump-flash 1.5s ease-out;
}

@keyframes ai-msg-jump-flash {
    0% {
        box-shadow: 0 0 0 3px var(--accent);
    }
    50% {
        box-shadow: 0 0 0 3px var(--accent), 0 0 12px color-mix(in srgb, var(--accent) 50%, transparent);
    }
    100% {
        box-shadow: 0 0 0 0 transparent;
    }
}
```

---

### 4. 分页加载适配说明

跳转函数基于 DOM 查询（`.ai-msg-user`），自动适配分页加载：

| 场景 | 行为 |
|------|------|
| 目标消息已加载（在 DOM 中） | 正常跳转 |
| 目标消息未加载（不在 DOM 中） | 循环跳转仅在当前已加载的消息范围内进行 |
| 滚到顶部触发加载后 | 新消息 prepend 到 DOM，后续 `Alt+↑` 可跳转到新加载的用户消息 |

**分页边界处理：**
- 跳转只在当前 DOM 已渲染的消息范围内循环
- 如果用户消息全部已加载（`_oldestMsgId === 0`），循环范围即是全部消息
- 如果还有未加载消息（`_oldestMsgId > 0`），用户可通过滚动到顶部触发加载后，再用 Alt+↑ 跳转到更早的消息
- 无需在跳转函数中主动触发加载，避免打断用户操作流

---

### 5. 完整边界处理

| 场景 | 处理方式 |
|------|----------|
| 无用户消息 | `window.showNotification?.('没有用户消息可供跳转', 'info')` |
| 仅有一条用户消息 | 高亮该消息一小段时间作为视觉反馈 |
| 流式输出中 | `window.showNotification?.('回复进行中，无法跳转', 'warning')` |
| 非 AI 助手视图 | 静默忽略（`window.state.currentView !== 'ai-chat'`） |
| 输入框/搜索框聚焦 | 静默忽略（避免 Alt+↑/↓ 干扰打字） |
| 快速连按 | 300ms 防抖，连按只触发一次 |
| 第一条←上一条 | 循环到最后一条（`(currentIndex - 1 + total) % total`） |
| 最后一条→下一条 | 循环到第一条（`(currentIndex + 1) % total`） |

---

## 验证步骤

1. 打开 AI 助手，进入一个有多个用户消息的会话
2. 按 `Alt+↓` → 跳转到下一条用户消息，消息气泡闪烁高亮
3. 按 `Alt+↑` → 跳转到上一条用户消息
4. 在最后一条按 `Alt+↓` → 循环到第一条
5. 在第一条按 `Alt+↑` → 循环到最后一条
6. 在无用户消息的会话中按 `Alt+↓` → 弹出通知提示
7. 在流式输出中按 `Alt+↓` → 弹出"回复进行中"提示
8. 切换到其他视图（如笔记网格）按 `Alt+↓` → 无反应
9. 快速连按 `Alt+↓` → 300ms 内只触发一次
10. 滚动到顶部触发加载更多消息后，`Alt+↑` 可跳转到新加载的早期用户消息