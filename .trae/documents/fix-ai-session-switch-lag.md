# 修复 AI 会话切换卡顿

## 总结

切换 AI 会话时，如果消息较多会出现「顿一下才滚动到底部」。根因是 `switchSession()` 中的 `forEach` 循环逐条调用 `addMessage()`，每条消息都触发 `scrollToBottom()`（强制回流 N 次），加上 `loadSessionList()` 多余的全量重渲染，叠加产生明显卡顿。

## 当前状态分析

### 问题根因

`switchSession()`（`ai-chat.js:1405`）的当前执行流：

```
switchSession(id)
  → GetAIConfig() IPC 调用（约 1-5ms）
  → messagesEl.innerHTML = ''（清空旧消息，触发 GC）
  → msgs.forEach(msg => {
      addMessage(msg.content, msg.role, ...)
        → renderMarkdown() ← marked.parse + hljs.highlightElement（同步，每条 AI 消息 10-200ms）
        → messagesEl.appendChild(el) 
        → scrollToBottom() ← scrollTop = scrollHeight 强制回流 ← ▲ 每次 addMessage 都触发
    })
  → loadSessionList() ← 全量重新加载并重绘会话列表（切换会话时完全没必要）
  → scrollToBottom() ← 多余的第 N+1 次强制回流
```

**关键数据**：每条 AI 消息的 `renderMarkdown` 耗时 10-200ms（取决于代码块数量和复杂度）。N=30 条消息时，`scrollToBottom` 被调用 30 次，`renderMarkdown` 累积耗时 0.3-6 秒。

### 发现的全部瓶颈

| 优先级 | 瓶颈 | 文件:行 | 影响 |
|--------|------|--------|------|
| **P0** | `forEach` 逐条 `addMessage` + N 次 `scrollToBottom` | `ai-chat.js:1436-1451,2396` | N 次强制回流，N=30 时额外增加 30 次 layout 计算 |
| **P0** | `renderMarkdown` 内 `marked.parse` + `hljs.highlightElement` 主线程同步执行 | `ai-chat.js:2169-2252` | 每条 AI 消息 10-200ms，累积占据主线程阻塞渲染 |
| **P1** | `switchSession` 末尾多余调用 `loadSessionList()` | `ai-chat.js:1453` | 额外一次 IPC + 全量重建侧栏 DOM（会话列表未变化） |
| **P1** | 后端 `LoadAISessionMessages` 无分页 | `ai_service.go:428` | 大量消息一次性加载到内存 + IPC 传输 |
| **P2** | 全部消息同时播放 CSS 入场动画 `0.2s` | `ai-chat.css:39` | N 条消息同时启动合成器动画 |
| **P3** | 每个 code block / table 独立绑定 click 事件 | `ai-chat.js:2193,2235` | 大量事件监听器占用内存 |

## 变更方案

### 改动文件

#### 1. `frontend/src/js/ai-chat.js` — 核心性能优化（3 处改动）

**改动 A：`switchSession()` 中批量构建 DOM + 单次滚动**

将 `msgs.forEach` 循环改为使用 `DocumentFragment`：

```javascript
// 改为使用 DocumentFragment
const fragment = document.createDocumentFragment();
const msgElements = [];

msgs.forEach(msg => {
    const el = document.createElement('div');
    el.className = 'ai-msg ' + (msg.role === 'user' ? 'ai-msg-user' : 'ai-msg-assistant');
    // ... 构建 el 内容（复现 addMessage 逻辑，但不 append 到 DOM）...
    fragment.appendChild(el);
    msgElements.push(el);
});

// 一次性写入 DOM
messagesEl.innerHTML = '';
messagesEl.appendChild(fragment);

// 处理动作按钮和右键菜单（需要 DOM 已存在）
msgElements.forEach((el, i) => {
    const msg = msgs[i];
    // 创建并追加 actions 等...（需在 DOM 中操作）
});

// 单次滚动到底部
scrollToBottom();
```

**简化方案**（改动更小，风险更低）：

保持 `addMessage` 结构不变，但改为**先收集到 fragment，再一次性 append**：

```javascript
const fragment = document.createDocumentFragment();
const tempContainer = document.createElement('div');

msgs.forEach(msg => {
    const el = addMessageToFragment(msg, tempContainer);
    fragment.appendChild(el);
});

messagesEl.innerHTML = '';
messagesEl.appendChild(fragment);
scrollToBottom(); // 只滚动一次
```

**改动 B：移除 `switchSession()` 末尾多余的 `loadSessionList()` 调用**

```javascript
// 删除 switchSession 末尾的：
// renderSessionList();     // ← 删除
// loadSessionList();       // ← 删除
// 改为只更新当前会话高亮状态：
highlightCurrentSession(); // 轻量操作，只改 className
updateChatTitle();
updateContextSize();
```

**改动 C：`addMessage` 中移除 `scrollToBottom()` 调用**

新增一个可选参数 `skipScroll = false`：

```javascript
function addMessage(content, role, reasoningContent, thinkingElapsed, totalElapsed,
                     tokens, msgId, searchSources, recallCards, skipScroll = false) {
    // ... 原有构建逻辑不变 ...
    messagesEl.appendChild(el);
    if (!skipScroll) scrollToBottom();  // 仅在流式追加时滚动
    return el;
}
```

#### 2. `frontend/src/js/ai-chat.js` — 优化 `renderMarkdown`（可选改进）

对于极高延迟场景（50+ 条消息，每条都有大量代码块）：

```javascript
function renderMarkdown(el, content, deferHighlight = false) {
    el.innerHTML = marked.parse(content);
    if (deferHighlight) {
        // 延迟高亮：使用 requestIdleCallback 分批处理
        const blocks = [...el.querySelectorAll('pre code[class*="language-"]')];
        let i = 0;
        const schedule = () => {
            const chunk = blocks.slice(i, i + 3);
            chunk.forEach(block => { try { hljs.highlightElement(block); } catch(_) {} });
            i += 3;
            if (i < blocks.length) requestIdleCallback(schedule, { timeout: 300 });
        };
        requestIdleCallback(schedule, { timeout: 300 });
    } else {
        // 立即高亮（流式回复等需要即时展示的场景）
        el.querySelectorAll('pre code[class*="language-"]').forEach(block => {
            try { hljs.highlightElement(block); } catch(_) {}
        });
    }
    // ... 后续复制按钮等操作 ...
}
```

> **决策**：这个优化对大多数会话（10-20 条消息）收益有限，且增加了复杂度。**暂不纳入本次改动**，仅作为文档记录。

### 不涉及的改动

- Go 后端：不修改。后端分页需要配合前端虚拟滚动，改动范围过大。
- CSS：不修改入场动画。0.2s 动画对感受延迟影响很小，且去除动画会降低体验。
- `index.html`：不修改。

## 执行计划

### Task 1：优化 `switchSession` 批量构建 DOM

**步骤**：
1. 修改 `switchSession()`，将 `msgs.forEach(msg => { ... })` 改为使用 `DocumentFragment` 一次性构建
2. 内部调用 `addMessage` 时传入新参数 `skipScroll = true`（或抽离纯 DOM 构建函数）
3. 将 fragment 一次性 append 到 `messagesEl`
4. 末尾只调用一次 `scrollToBottom()`

### Task 2：移除多余 `loadSessionList()` 调用

**步骤**：
1. 删除 `switchSession()` 末尾的 `loadSessionList()` 和 `renderSessionList()` 调用
2. 替换为轻量级的高亮更新（仅更新当前选中项的 `.active` 类名）
3. 保留 `updateChatTitle()` 和 `updateContextSize()` 等必要调用

### Task 3：编译验证

## 验证步骤

1. 创建一个有 30+ 条消息的 AI 对话会话（包含代码块等复杂内容）
2. 在侧栏切换到此会话
3. **预期**：消息瞬间加载完毕，无明显顿挫，直接滚动到底部
4. 在侧栏切换回会话列表较少会话
5. **预期**：切换流畅无卡顿
6. 新发送一条消息（非切换场景）
7. **预期**：流式逐条追加正常，`scrollToBottom()` 正常工作，不受改动影响
