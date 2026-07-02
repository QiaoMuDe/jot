# AI 对话 - 用户消息重新发送功能

## 摘要

为 AI 对话中用户消息添加「重新发送」功能，用户可一键重发自己的消息，触发 AI 重新响应。该功能同时出现在消息底部的操作按钮和右键菜单中。

## 现状分析

现有消息操作能力对比：

| 操作 | AI 消息 (assistant) | 用户消息 (user) |
|------|:---:|:---:|
| 复制 | ✅ | ✅ |
| 编辑 | ❌ | ✅（操作栏） |
| 保存为笔记 | ✅ | ❌ |
| **重新生成** | ✅（操作栏 + 右键菜单） | ❌ |
| 追问 | ✅ | ❌ |
| 删除 | ✅（操作栏 + 右键菜单） | ✅（操作栏 + 右键菜单） |

**用户消息没有「重新发送」功能。** 如果用户想重发消息，只能手动复制文本 → 清空输入框 → 粘贴 → 发送，流程繁琐。

## 设计思路

参考 `handleRegenerate`（AI 消息重新生成）的实现模式，为用户消息实现对称的 `handleResend`：

```
用户点击"重新发送"
  → 从该用户消息位置截断 chatHistory 和 DOM
  → 清空 DB 并重新保存截断后的历史（保持 DB 一致性）
  → 重新添加用户消息到 DOM + chatHistory（fresh 状态）
  → startStreaming(false) 发起新请求
```

与 `handleRegenerate` 的关键区别：`resend` **要移除**该用户消息自身（它在 chatHistory 和 DOM 都消失），然后重新添加并发送；而 `regen` **保留**用户消息，只移除 AI 回复。

## 修改文件

只修改一个文件：
- `d:\峡谷\Dev\本地项目\jot\frontend\src\js\ai-chat.js`

## 具体改动

### 改动 1：新增 `handleResend(msgEl)` 函数

放在 `handleRegenerate` 函数附近（第 2952 行之后），实现逻辑：

```javascript
async function handleResend(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    // 从 DOM 和 chatHistory 中找到该用户消息的索引
    const children = Array.from(messagesEl.children);
    const idx = children.indexOf(msgEl);
    if (idx === -1) return;

    // 获取用户消息内容
    const contentDiv = msgEl.querySelector('.msg-content');
    if (!contentDiv) return;
    const content = contentDiv.textContent || '';

    // 截断：移除该用户消息及之后的所有内容
    chatHistory.splice(idx);
    updateContextSize();
    children.slice(idx).forEach(el => el.remove());

    // 同步 DB
    try {
        await window.go.main.App.ClearAISessionMessages(activeSessionId);
        if (chatHistory.length > 0) {
            await window.go.main.App.SaveAIMessages(activeSessionId, chatHistory);
        }
    } catch (_) {}

    // 重新添加用户消息（fresh DOM）
    addMessage(content, 'user');
    const newUserMsgEl = messagesEl.lastElementChild;
    if (newUserMsgEl) {
        newUserMsgEl.appendChild(createMsgActions(content, 'user'));
        bindMsgContextMenu(newUserMsgEl, content, 'user');
    }
    chatHistory.push({ role: 'user', content });
    updateContextSize();

    // 重新发送（复用现有的系统上下文: referencedNotes / followUpRef 等仍在内存中）
    startStreaming(false);
}
```

### 改动 2：右键菜单添加「重新发送」

在 `showAiMsgContextMenu` 函数（第 2443-2446 行）的用户消息菜单段中，在「编辑」之后添加「重新发送」：

```javascript
if (role === 'user') {
    items.push({ type: 'divider' });
    items.push({ action: 'edit', label: '编辑' });
    items.push({ action: 'resend', label: '重新发送' });  // ← 新增
}
```

### 改动 3：右键菜单点击处理

在 `aiMsgContextMenu` 的 click 事件处理（第 1238 行附近）中，在 `action === 'delete'` 之前添加：

```javascript
} else if (action === 'resend') {
    if (isStreaming) return;
    if (msgEl) handleResend(msgEl);
}
```

### 改动 4：消息操作按钮添加「重新发送」

在 `createMsgActions` 函数（第 2620-2634 行）的用户消息操作区域中，在「编辑」按钮之后添加「重新发送」按钮：

```javascript
if (role === 'user') {
    // 编辑按钮（已有）
    // ...
    
    // 重新发送按钮（新增）
    const resendBtn = document.createElement('button');
    resendBtn.innerHTML = RESEND_ICON;  // 使用旋转箭头 SVG
    resendBtn.title = '重新发送';
    resendBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        if (isStreaming) return;
        handleResend(container.parentElement);
    });
    btnWrap.appendChild(resendBtn);
}
```

### 改动 5：新增 SVG 图标

在 `REGEN_ICON` 定义（第 2589 行）之后，新增 `RESEND_ICON` — 使用一个指向右侧的发送箭头图标：

```javascript
const RESEND_ICON = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>';
```

## 不变的行为

- `isStreaming` 检查：流式传输中禁用重新发送（与编辑、删除、重新生成一致）
- 重新发送时系统上下文（`referencedNotes`、`followUpRef`）保持不变，因为它们在内存中未清除
- 重新发送后侧栏会话预览会正确更新（通过 `startStreaming` 内部 done/error 回调自动处理）
- DB 同步逻辑与 `handleRegenerate` 一致

## 验证方式

1. 发送一条用户消息 → 看到 AI 回复
2. 右键该用户消息 → 出现「重新发送」选项 → 点击 → 消息正常重新发送，AI 重新回复
3. 鼠标悬浮在用户消息上 → 操作栏出现「重新发送」按钮 → 点击 → 同上
4. 多条消息历史中重新发送某条用户消息 → 该消息之后的所有内容被正确截断
5. 流式传输中右键点击用户消息 → 「重新发送」按钮应无响应（与删除行为一致）
