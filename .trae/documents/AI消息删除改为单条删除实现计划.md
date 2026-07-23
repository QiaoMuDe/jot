# AI 消息删除改为单条删除 — 实现计划

## 概要

将 AI 对话中右键删除消息的行为从"截断删除（删除本条及后续所有消息）"改为"仅删除本条消息"。

**改动范围：仅需修改前端 1 个函数（`handleDeleteMsg`）中的 3 处代码，后端无需任何修改。**

---

## 当前状态分析

### 前端：`handleDeleteMsg`（[ai-chat.js#L3513-L3554](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3513-L3554)）

```javascript
// 当前行为：
// 1. DOM：移除 msgEl 及之后所有 .ai-msg 兄弟节点
// 2. 后端：调用 TruncateAISessionAtMessage（删除 created_at >= 本消息的所有记录）
// 3. chatHistory：slice(0, idx) 截断
```

### 后端：已有现成 API

- [`DeleteAIMessage(id)`](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2623-L2632) — Wails 绑定层，已暴露给前端
- [`DeleteAIMessage(id)`](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L920-L927) — 业务层，直接 `db.Delete(&AIMessage{}, id)`

**后端无需任何改动。**

---

## 具体变更

### 变更 1：DOM 移除 — 只删本条消息

**文件：** `frontend/src/js/ai-chat.js` 第 3525-3533 行

**旧代码：**
```javascript
// 移除该消息及之后的所有后续消息（DOM）
let nextEl = msgEl;
while (nextEl) {
    const toRemove = nextEl;
    nextEl = nextEl.nextElementSibling;
    if (toRemove.classList.contains('ai-msg')) {
        toRemove.remove();
    }
}
```

**新代码：**
```javascript
// 仅移除本条消息（DOM）
msgEl.remove();
```

### 变更 2：后端 API 调用 — 从截断改为单条删除

**文件：** `frontend/src/js/ai-chat.js` 第 3535-3540 行

**旧代码：**
```javascript
// 后端截断（删除本条及之后的消息）
if (activeSessionId !== null) {
    try {
        await window.go.main.App.TruncateAISessionAtMessage(activeSessionId, msgId);
    } catch (_) { /* 静默失败 */ }
}
```

**新代码：**
```javascript
// 后端仅删除本条消息
if (activeSessionId !== null) {
    try {
        await window.go.main.App.DeleteAIMessage(msgId);
    } catch (_) { /* 静默失败 */ }
}
```

### 变更 3：chatHistory 缓冲区 — 从截断改为过滤移除单条

**文件：** `frontend/src/js/ai-chat.js` 第 3542-3546 行

**旧代码：**
```javascript
// 截断 chatHistory 缓冲区
const idx = chatHistory.findIndex(m => m.id === msgId);
if (idx >= 0) {
    chatHistory = chatHistory.slice(0, idx);
}
```

**新代码：**
```javascript
// 从 chatHistory 中移除本条消息
chatHistory = chatHistory.filter(m => m.id !== msgId);
```

---

## 决策说明

### 未采纳方案：右键菜单添加"删除后续所有"选项

**理由：** 用户要求的是直接替换现有行为，而非保留两种模式。如果将来需要截断删除功能，可以再通过右键菜单添加二级选项。

### 消息间 gap 问题

删除中间某条消息后，对话历史会出现一条"空洞"（例如消息 ID 序列为 1→2→4→5），但后端加载历史时按 `created_at` 排序，缺失一条消息不会影响 AI 对话的连贯性 — 这类似于用户手动删除了自己说过的一句话。

---

## 验证步骤

1. **右键删除用户消息** — 仅该条消息消失，前后消息保持不变
2. **右键删除 AI 消息** — 仅该条消息消失，前后消息保持不变
3. **删除非末尾消息** — 后续消息保持正常显示，会话视图不跳转
4. **删除后发送新消息** — AI 调用正常，不报错
5. **删除最后一条消息** — 触发 `showWelcome()` 显示欢迎页
6. **流式传输中删除** — 应被 `isStreaming` 检查拦截，不执行删除
