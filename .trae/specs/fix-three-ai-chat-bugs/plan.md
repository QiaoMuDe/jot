# 修复 AI 助手模块三个已核实 Bug

## 问题列表

| # | 问题 | 严重度 | 位置 |
|---|------|--------|------|
| 1 | Scroll Handler 泄漏 | 中高 | `ai-chat.js:1613-1660` |
| 2 | 编辑内容未写回 DB | 高 | `ai-chat.js:3332-3381` + `app.go` |
| 3 | 再生模式 token 不更新 | 低 | `app.go:2365` + `ai-chat.js:2262-2276` |

---

## 修复方案

### 修复 1：Scroll Handler 泄漏

**问题**：`scrollHandler` 每次 `switchSession` 时创建新的函数对象，`removeEventListener` 无法移除旧 listener，累积泄漏。

**改动**：`frontend/src/js/ai-chat.js`

- 在模块作用域（`switchSession` 外部）声明 `let _scrollHandler = null;`
- `switchSession` 内改为：

```javascript
if (_scrollHandler) {
    messagesEl.removeEventListener('scroll', _scrollHandler);
}
_scrollHandler = async () => {
    // ... 原有逻辑不变
};
messagesEl.addEventListener('scroll', _scrollHandler, { passive: true });
```

### 修复 2：编辑内容未写回 DB

**问题**：`applyEdit` 调用 `TruncateAISessionAfterMessage` 删除后续消息，但未更新被编辑消息自身的 `content`，导致 `CallAIStreamRegenerate` 从 DB 加载旧内容。

**改动**：`frontend/src/js/ai-chat.js:3357` 后添加：

```javascript
// 更新被编辑消息的内容
await window.go.main.App.UpdateAIMessageContent(msgId, newContent);
```

### 修复 3：再生模式 token 不更新

**问题**：`CallAIStreamRegenerate` stream-done 中 `userMsgID = uint(0)`，前端跳过 `.user-tokens` 更新。

**改动**：`app.go`

在 `CallAIStreamRegenerate` 的 stream-done 回调中，`runtime.EventsEmit` 之前添加：

```go
// 查找会话中最后一条用户消息并更新其 tokens
var lastUserMsgID uint
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "user" {
        lastUserMsgID = messages[i].ID
        break
    }
}
_ = a.aiService.UpdateAIMessageTokens(lastUserMsgID, userTokens)
```

将 `runtime.EventsEmit(..., uint(0), ...)` 改为 `runtime.EventsEmit(..., lastUserMsgID, ...)`

---

## 文件改动清单

| 文件 | 行号 | 改动 | 改动量 |
|------|------|------|--------|
| `frontend/src/js/ai-chat.js` | ~L1610-1660 | scrollHandler 提升到模块级 | +3 行，-0 行 |
| `frontend/src/js/ai-chat.js` | ~L3357 | 新增 UpdateAIMessageContent 调用 | +1 行 |
| `app.go` | ~L2356-L2365 | 查找 lastUserMsgID + 替换 uint(0) | +8 行 |

## 验证方法

1. **修复 1**：切换会话 3 次，用 DevTools Elements 面板检查 `messagesEl` 的 scroll 事件监听器数量，应为 1
2. **修复 2**：编辑一条用户消息，观察 AI 回复是否基于编辑后的内容（而非旧内容）
3. **修复 3**：点击重新生成，观察用户消息气泡的 token 数字是否与切换会话后一致
4. **回归**：发送新消息、重发、删除消息功能正常
