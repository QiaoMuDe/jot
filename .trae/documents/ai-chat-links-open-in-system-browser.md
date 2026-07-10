# AI 聊天链接全局事件委托 — 修复方案

## Summary

通过全局事件委托，拦截 AI 聊天区域 `#aiChatMessages` 内所有 `<a>` 标签的点击，统一调用 `window.runtime.BrowserOpenURL()` 在系统浏览器中打开，避免链接在 Wails WebView2 窗口内导航导致用户离开应用。

## Current State Analysis

**问题**: AI 消息正文中 `marked.parse()` 渲染的 Markdown 链接（`<a href="...">`）没有 `target="_blank"` 也没有任何点击拦截处理。点击后 Wails WebView2 直接在应用窗口内加载目标页面，用户会"离开" jot 界面。

**已有正确实现**: 搜索来源面板的链接（[ai-chat.js:2129](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2129-L2134) 和 [ai-chat.js:2501](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2501-L2506)）通过 `e.preventDefault()` + `window.runtime.BrowserOpenURL(url)` 正确处理了外部链接。

## Proposed Changes

### 文件: `frontend/src/js/ai-chat.js`

#### 变更 1: 添加全局事件委托（新增，约 10 行）

在 `initAIChat()` 函数底部（`return` 之前，约第 4221 行附近），添加一个点击事件委托到 `messagesEl`（即 `#aiChatMessages`）：

```javascript
// 统一处理 AI 消息中所有链接→系统浏览器打开
messagesEl.addEventListener('click', function (e) {
    const link = e.target.closest('a');
    if (link && link.href && !link.href.startsWith('#')) {
        e.preventDefault();
        window.runtime.BrowserOpenURL(link.href);
    }
});
```

**Why**:
- `e.target.closest('a')` 能精确捕获点击的 `<a>` 标签，即使内部有嵌套元素
- `!link.href.startsWith('#')` 排除锚点链接
- 放置在 `initAIChat()` 末尾，此时 `messagesEl` 已通过 `document.getElementById('aiChatMessages')` 获取

#### 变更 2: 清理搜索来源面板的冗余点击处理（可选清理，2 处）

搜索来源面板的链接**也会被全局委托捕获**，可以删掉原有的重复代码：

**位置 1** — [ai-chat.js:2129-2134](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2129-L2134)：
移除 `link.addEventListener('click', ...)` 块（4 行），保留其他代码。

**位置 2** — [ai-chat.js:2501-2506](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2501-L2506)：
同上，移除 `link.addEventListener('click', ...)` 块。

**Why**: 全局委托会捕获 `#aiChatMessages` 内所有 `<a>` 的点击，包括搜索来源面板中的链接。删除重复代码简化维护。

## Assumptions & Decisions

1. 所有 AI 聊天中的外部链接都应该在系统浏览器中打开，无例外。
2. `window.runtime.BrowserOpenURL` 在 Wails v2 中可用（已在搜索来源面板中验证）。
3. 搜索来源面板中的 `<a>` 标签位于 `#aiChatMessages` 容器内，会被全局委托捕获。
4. 不处理 `#` 锚点链接（保留默认行为，虽然 AI 消息中几乎不会出现）。

## Verification

1. 在 AI 聊天中发送一条包含 Markdown 链接的消息（如 `[百度](https://www.baidu.com)`）
2. 点击该链接，确认系统浏览器打开目标页面，jot 应用界面保持不变
3. 点击搜索来源面板中的链接，确认行为一致（系统浏览器打开）
4. 点击非链接文本区域，确认无异常触发
