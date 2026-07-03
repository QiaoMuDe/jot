# Token 计算与持久化修复计划

## 当前问题

### 问题 1：核心 BUG — `userTokens` 是累积值（后端 app.go）

后端 [app.go#L844-L848](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L844-L848) 的 `userTokens` 计算累加了 `messages` 数组中 **所有 system + user 消息**，而不是只计算当前这条用户消息的 token。

```go
userTokens := 0
for _, msg := range messages {
    if msg.Role == "system" || msg.Role == "user" {
        userTokens += estimateTokens(msg.Content) // 累加所有
    }
}
```

`messages` 包含：system 上下文 + **所有历史对话**。导致：

* 每轮用户消息显示的 token 数偏大（包含了之前所有用户消息和 system 上下文）

* `updateContextSize` 总计重复计算（前序消息已有自己的 tokens，又被当前 userTokens 包含一次）

### 问题 2：`handleResend` 和 `applyEdit` 的 `chatHistory.push` 缺少 `tokens: 0`

* [ai-chat.js#L3218](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3218)：`handleResend` 中 `chatHistory.push({ role: 'user', content })` 缺少 `tokens: 0`

* [ai-chat.js#L3078](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3078)：`applyEdit` 中 `chatHistory.push({ role: 'user', content: newContent })` 缺少 `tokens: 0`

虽然 stream-done 会补充赋值 `lastMsg.tokens = userTokens`，但中间时刻 `tokens` 为 `undefined`，可能导致 `updateContextSize` 的中间结果错误。

### 问题 3：编辑消息后 DB 中用户消息 token 为 0

`applyEdit` 执行流程：

1. `ClearAISessionMessages` + `SaveAIMessages(chatHistory)` — 用户消息 `tokens: 0` 被写入 DB
2. `startStreaming(true)` — isRegenerate=true，后端 **只保存 assistant**，不更新用户消息 token

结果：DB 中编辑后的用户消息 `tokens = 0`，切换会话重新加载时 tokens 显示为 0。

### 已核实：删除消息流程正常

`handleDeleteMsg` 不涉及流式调用，直接 `chatHistory.splice(idx)` 移除消息（连带移除 tokens），然后 `updateContextSize()` 重新汇总剩余消息的 tokens。DB 也同步更新。**无需修改。**

### 已核实：重新生成 AI（handleRegenerate）流程正常

`handleRegenerate` 截断 chatHistory（移除 AI 消息），保留已有正确 tokens 的用户消息，`startStreaming(true)` 后端只存 assistant。stream-done 更新 `lastMsg.tokens = userTokens`。**各环节正确。**

***

## 修复方案

### 修复 1：后端 — `userTokens` 改为仅计算本轮 system 上下文 + 最后一条用户消息

**文件**：[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L844-L848)

**改动**：将 `userTokens` 的计算从"累加所有 system+user"改为"累加当前 system 上下文 + 最后一条用户消息"。

```go
// 改前
userTokens := 0
for _, msg := range messages {
    if msg.Role == "system" || msg.Role == "user" {
        userTokens += estimateTokens(msg.Content)
    }
}

// 改后
userTokens := 0
// 本轮 system 上下文（含 skill prompts、笔记引用、搜索注入、卡片召回）
for _, msg := range messages {
    if msg.Role == "system" {
        userTokens += estimateTokens(msg.Content)
    }
}
// 最后一条用户消息（当前轮次用户的输入）
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "user" {
        userTokens += estimateTokens(messages[i].Content)
        break
    }
}
```

**效果**：

* 每条用户消息的 token = 本轮注入的 system 上下文 + 用户输入

* `updateContextSize` 总计 = Σ(各轮用户消息 token) + Σ(AI 消息 token) — 无重复计数

* 每轮各自的 system 上下文仅在该轮计费，不会跨轮次累计

### 修复 2：前端 — 补全 `chatHistory.push` 的 `tokens: 0`

**文件**：[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js)

**改动 a** — L3218 `handleResend`：

```js
// 改前
chatHistory.push({ role: 'user', content });
// 改后
chatHistory.push({ role: 'user', content, tokens: 0 });
```

**改动 b** — L3078 `applyEdit`：

```js
// 改前
chatHistory.push({ role: 'user', content: newContent });
// 改后
chatHistory.push({ role: 'user', content: newContent, tokens: 0 });
```

### 修复 3：编辑消息后同步用户消息 token 到 DB

**方案**：在 `applyEdit` 中，流完成后将 chatHistory（已包含 stream-done 更新的 `userTokens` 值）重新持久化到 DB。

**文件**：[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js)

**改动** — 在 `applyEdit` 的流结束后，监听 stream-done 更新完 chatHistory 后再同步 DB。

有两种实现方式：

**方式 A（推荐 — 在 stream-done 统一处理）：**
在 `applyEdit` 中，启动流式后等待其完成。利用已有的 `startStreaming(true)` 的 `isRegenerate` 机制，在 stream-done 处理器的末尾，检测是否需要将更新后的 chatHistory 重新同步到 DB。

具体：在 `applyEdit` 中设置一个标志 `_pendingTokenSync = true`，然后在 stream-done 处理器末尾检查此标志，若为 true 则执行 `ClearAISessionMessages + SaveAIMessages(chatHistory)` 刷新 DB。

**方式 B（更简洁 — 修改 applyEdit 的保存逻辑）：**
将 `applyEdit` 中的 `SaveAIMessages(chatHistory)` 从"流前保存"改为"流后保存"：

1. 流前只执行 `ClearAISessionMessages`（清空 DB）
2. 流结束后（stream-done 已更新 `lastMsg.tokens`），再执行 `SaveAIMessages(chatHistory)` 持久化完整且正确的数据

```js
// applyEdit 改动后的核心逻辑：
// 1. 清空 DB（已存在）
await window.go.main.App.ClearAISessionMessages(activeSessionId);
// 2. 不在此处 SaveAIMessages，让 stream-done 后再保存
// 3. 启动流
startStreaming(true);
// 4. 在 stream-done 处理器的末尾，检测是否需要持久化
```

**推荐方式 B**，更简洁且利用 stream-done 处理器的最终状态。

但方式 B 有一个问题：stream-done 处理器是事件监听，`applyEdit` 无法直接 await 它完成。需要使用标志位方式。

**最终采用 方式 A**：

* 在 `applyEdit` 中设置 `_pendingTokenSync = true`

* 在 stream-done 处理器末尾，如果 `_pendingTokenSync` 为 true，执行 `ClearAISessionMessages + SaveAIMessages(chatHistory)`，然后复位标志

### 顺便修改：handleRegenerate 的流后同步

`handleRegenerate` 也有类似情况：流前保存了 chatHistory，流后 stream-done 更新了 `lastMsg.tokens`，但 DB 中的用户消息 token 没有更新。不过 regenerate 场景的用户消息 token 在最初的 `onSend` 流程中已被后端正确保存，而编辑场景的用户消息 token 在 DB 中是 0。

为了简洁且避免过度修改，regenerate 场景暂不做 DB 同步（已有正确的 token），仅修复编辑场景。

***

## 涉及文件

| 文件                           | 改动                                             |
| ---------------------------- | ---------------------------------------------- |
| `app.go`                     | 修复 `userTokens` 计算逻辑（L844-848）                 |
| `frontend/src/js/ai-chat.js` | 补全 `handleResend` 的 `tokens: 0`（L3218）         |
| `frontend/src/js/ai-chat.js` | 补全 `applyEdit` 的 `tokens: 0`（L3078）            |
| `frontend/src/js/ai-chat.js` | 编辑消息流后同步 DB（`applyEdit` + stream-done handler） |

## 验证步骤

1. **正常发送**：发送一条消息 → 确认用户消息显示正确 token 数（仅含本轮 system 上下文 + 用户输入）
2. **连续多轮**：发送多条消息 → 确认每条消息 token 独立计数，总计不重复
3. **重新发送**：点击用户消息的重新发送按钮 → 流完成后 token 正确更新
4. **重新生成 AI**：点击 AI 消息的重新生成按钮 → 流完成后 AI token 正确更新，用户消息 token 保持不变
5. **编辑消息**：编辑用户消息后发送 → 流完成后 token 正确更新，切换会话重新加载后 token 仍然正确
6. **删除消息**：删除某条消息 → 后续消息和总计正确更新
7. **切换会话**：切换会话 → 所有消息 token 显示正确，总计正确

