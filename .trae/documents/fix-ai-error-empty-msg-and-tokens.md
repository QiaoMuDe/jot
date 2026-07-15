# 修复计划：AI 出错时空消息写入 & Token 不显示

## 摘要

修复两个预存在的 bug，均在 AI 流式出错时触发：
- **Issue 1**: 出错时用户消息底部不显示 Token 数，会话总 Token 也不更新
- **Issue 2**: 出错时后端往 DB 写入一条空内容的 assistant 消息，切换会话回来显示空白 AI 气泡

## 根因分析

### Issue 2 — 空消息写入

`internal/aicli/client.go` 第 28-90 行的 `Stream()` 方法存在设计缺陷：

```go
// 第 67-78 行：有错误时通知
if err != nil {
    ...
    callbacks.OnError(...)  // ← 调用 OnError
}
// 第 80-89 行：耗时计算 + 无条件调 OnDone
...
if callbacks.OnDone != nil {
    callbacks.OnDone(fullContent.String(), ...)  // ← 也调 OnDone，fullContent 为空
}
```

**有错误时 `OnError` 和 `OnDone` 都会被调用。** `OnDone` 传入的 `fullContent.String()` 是空字符串，导致 `app.go` 第 1926 行的 `SaveAIMessage` 向 DB 写入一条 `content=""` 的 assistant 消息。

### Issue 1 — Token 不显示

用户消息创建时 `createMsgActions(text, 'user', undefined, 0)` 固定传 `tokens=0`，默认不显示 Token 数（[ai-chat.js 第 3221 行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3221)）。

Token 数的真正更新发生在 `ai:stream-done` 事件回调中（[ai-chat.js 第 2285-2288 行](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2285-L2288)）。出错时此事件不触发，Token 数一直为 0（隐藏状态）。

## 修改方案

### 修改 1：`internal/aicli/client.go` — 防止空消息写入

**改动**：第 87 行的 `OnDone` 调用加 `err == nil` 守卫。

```go
// before
if callbacks.OnDone != nil {
    callbacks.OnDone(fullContent.String(), elapsedThinking, elapsedTotal)
}

// after
if err == nil && callbacks.OnDone != nil {
    callbacks.OnDone(fullContent.String(), elapsedThinking, elapsedTotal)
}
```

**效果**：AI 出错时 `OnDone` 不会被调用，`app.go` 不会再保存空 assistant 消息到 DB。

### 修改 2：`frontend/src/js/ai-chat.js` — 出错时显示 Token

**改动位置**：`ai:stream-error` 事件处理器（第 2429 行），在清理逻辑之后增加 Token 更新。

**具体代码**：

```javascript
// 在 ai:stream-error 处理器末尾，清理追问引用之后，增加：

// 出错时更新用户消息 token 显示
const lastUserMsgEl = messagesInnerEl.querySelector('.ai-msg-user:last-child');
if (lastUserMsgEl) {
    const contentDiv = lastUserMsgEl.querySelector('.msg-content');
    if (contentDiv) {
        const text = contentDiv.textContent || '';
        // 匹配后端 estimateTokens 公式
        let chineseCount = 0;
        for (const ch of text) {
            if (/[\u4e00-\u9fff]/.test(ch)) chineseCount++;
        }
        const otherCount = [...text].length - chineseCount;
        const estimated = Math.ceil(chineseCount / 1.5 + otherCount / 4);
        const tokensEl = lastUserMsgEl.querySelector('.user-tokens');
        if (tokensEl && estimated > 0) {
            tokensEl.textContent = formatTokens(estimated) + ' tokens';
        }
    }
}
// 刷新会话 token 显示
updateContextSize();
```

**效果**：出错后用户消息底部显示估算的 Token 数，会话总 Token 数也从后端刷新显示。

## 影响范围

| 文件 | 修改类型 | 影响 |
|------|---------|------|
| `internal/aicli/client.go:87` | 1 行条件判断 | 切断空消息写入路径 |
| `frontend/src/js/ai-chat.js:2450` | 约 20 行新增代码 | 出错时更新 Token 显示 |

- 无数据库 schema 变更
- 无 CSS 变更
- 无 Wails binding 变更（不改后端 Go 函数签名）

## 验证步骤

1. 发送消息 → AI 主动出错 → 用户消息底部显示 Token 数
2. 发送消息 → AI 主动出错 → 不显示空的 AI 气泡
3. 发送消息 → AI 主动出错 → 切换会话再回来 → 不出现空白的 AI 消息
4. 正常流式完成 → Token 数和空消息行为无退化（`err == nil` 时走原路径） 
