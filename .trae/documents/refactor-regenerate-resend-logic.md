# 重构重新生成/重新发送逻辑

## 概述

去掉用户消息的"重新发送"按钮，只保留 AI 消息的"重新生成"。重构 `handleRegenerate`：不再截断后续对话，而是根据 AI 消息是否为最后一条，走两种不同路径。

***

## 当前状态分析

### 现有函数

- **`handleRegenerate`** — [ai-chat.js#L5205](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5205)

  - 找到前一条用户消息的 `msgId`

  - 调用 `TruncateAISessionAfterMessage(prevMsgId)` 截断该消息之后的所有内容

  - 更新前一条用户消息的 Meta

  - `startStreaming('', 0)`（后端找末条用户消息重新生成）

- **`handleResend`** — [ai-chat.js#L5264](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5264)

  - 读取用户消息文本

  - 调用 `TruncateAISessionAtMessage(msgId)` 截断该消息及之后的所有内容

  - 保存新用户消息到 DB

  - 添加新用户气泡

  - `startStreaming(content, newUserMsgId)`

### 存在问题的行为

```
用户: "1+1=?"
AI:  "2"    ← 点重新生成
用户: "1+2=?"
AI:  "3"    ← 后续对话被截断丢失！
```

***

## 修改方案

### 1. 删除 `handleResend` 函数

整段删除 [ai-chat.js#L5264-L5337](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L5264-L5337)。

### 2. 删除用户消息菜单中的"重新发送"按钮

在 [ai-chat.js#L3990](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3990) 删除 `resend` 菜单项和对应的图标映射。

### 3. 删除 `resend` action 处理分支

在 [ai-chat.js#L1269-L1271](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1269-L1271) 删除 `resend` action 的 `handleResend` 调用。

### 4. 重构 `handleRegenerate` 函数

**新逻辑：**

```js
async function handleRegenerate(msgEl) {
    if (!msgEl || !msgEl.parentNode || isStreaming) return;

    const msgId = parseInt(msgEl.dataset.msgId);
    if (!msgId) return;

    // 找到前一条消息（用户消息）
    const prevEl = msgEl.previousElementSibling;
    if (!prevEl || !prevEl.classList.contains('ai-msg')) return;
    const prevMsgId = +prevEl.dataset.msgId || 0;
    if (!prevMsgId) return;

    // 获取前一条用户消息的文本
    const prevContentDiv = prevEl.querySelector('.msg-content');
    if (!prevContentDiv) return;
    const prevTextEl = prevContentDiv.querySelector('.ai-msg-text');
    const prevContent = prevTextEl ? prevTextEl.textContent : (prevContentDiv.textContent || '');

    // 判断后续是否还有消息
    const hasNext = msgEl.nextElementSibling !== null &&
        msgEl.nextElementSibling.classList.contains('ai-msg');

    if (!hasNext) {
        // ── 情况 1：AI 消息是最后一条，原地重新生成 ──
        // 删除该 AI 消息 DOM
        msgEl.remove();

        // 后端仅删除本条 AI 消息（DeleteAIMessage 已存在）
        try {
            await window.go.main.App.DeleteAIMessage(msgId);
        } catch (_) { /* 静默 */ }

        // 从 chatHistory 移除
        const idx = chatHistory.findIndex(m => m.id === msgId);
        if (idx >= 0) chatHistory.splice(idx, 1);

        // 同步前一条用户消息的 Meta
        try {
            const newMeta = buildUserMessageMeta();
            await window.go.main.App.UpdateAIMessageMeta(prevMsgId, newMeta);
            rerenderUserMessageChips(prevMsgId, newMeta);
            const prevIdx = chatHistory.findIndex(m => m.id === prevMsgId);
            if (prevIdx >= 0) chatHistory[prevIdx].meta = newMeta;
        } catch (e) {
            console.warn('UpdateAIMessageMeta 失败', e);
        }

        // Phase 1: 更新上下文使用率
        updateContextUsage();

        // 重新生成
        if (!(await ensureAIReady('重新生成'))) return;
        await startStreaming('', 0);
    } else {
        // ── 情况 2：AI 消息后面还有内容，直接作为新消息发送 ──
        // 从前一条用户消息读取文本和 Meta
        const regenerateMeta = buildUserMessageMeta();

        // 保存为新用户消息
        let newUserMsgId = 0;
        let regenerateTokens = 0;
        if (activeSessionId !== null) {
            try {
                const result = await window.go.main.App.SaveAIMessage(
                    activeSessionId, prevContent, 'user', regenerateMeta
                );
                newUserMsgId = result?.msgID || 0;
                regenerateTokens = result?.tokens || 0;
            } catch (_) { /* 静默 */ }
        }

        // 添加新用户消息气泡
        addMessage(prevContent, 'user', undefined, undefined, undefined, regenerateTokens, newUserMsgId || undefined, undefined, undefined, false, false, regenerateMeta);
        const newUserMsgEl = messagesInnerEl.lastElementChild;
        if (newUserMsgEl) {
            newUserMsgEl.appendChild(createMsgActions(prevContent, 'user', undefined, regenerateTokens));
            bindMsgContextMenu(newUserMsgEl, prevContent, 'user');
        }
        // 同步 push 到 chatHistory
        if (newUserMsgId) {
            chatHistory.push({
                id: newUserMsgId, role: 'user', content: prevContent,
                tokens: regenerateTokens, meta: regenerateMeta || ''
            });
        }

        // Phase 1: 更新上下文使用率
        updateContextUsage();

        // 重新发送（不截断后续内容）
        if (!(await ensureAIReady('重新发送'))) return;
        await startStreaming(prevContent, newUserMsgId);

        // 清空上传文件列表
        uploadedFiles = [];
        renderFileChips();
    }

    scrollToBottom();
}
```

### 5. 删除 `RESEND_ICON` 引用

如果 `RESEND_ICON` 不再被其他地方引用，可以删除。但建议保留（暂不删除，避免产生死代码线索）。

### 6. 不需要后端改动

- `DeleteAIMessage` 已存在 — [app.go#L3010](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L3010)

- `SaveAIMessage` 已存在

- `UpdateAIMessageMeta` 已存在

- 不再需要 `TruncateAISessionAfterMessage` 和 `TruncateAISessionAtMessage` 调用

***

## 验证步骤

1. `go build` 编译通过
2. 前端 `npm run build` 无报错
3. 手动测试：

   - **情况 1**：单条 AI 消息 → 点重新生成 → 旧 AI 消息消失 → 新 AI 回复出现

   - **情况 2**：AI 消息后有后续对话 → 点重新生成 → 新用户消息出现 → 后续对话保留 → 新 AI 回复出现

   - 确认用户消息菜单不再显示"重新发送"按钮

   - 确认 AI 消息菜单仍显示"重新生成"按钮

