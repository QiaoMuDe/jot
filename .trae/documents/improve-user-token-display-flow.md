# 用户消息 Token 提前展示方案

## 现状分析

当前消息处理流程：

```
用户发送消息 → SaveAIMessage 保存用户消息(已算 token) → 返回 msgID
              → addMessage(tokens=0) → 用户气泡 token 显示为 0
              → startStreaming() → AI 流式回复
              → AI 流完成 → OnDone 回调：
                  1. 计算完整上下文 token（含 system 提示词）
                  2. 保存 AI 消息、更新用户消息 token 到 DB
                  3. 发射 ai:stream-done(userTokens)
              → 前端更新用户气泡 token 显示
```

**问题**：`SaveAIMessage` 已计算了用户消息自身的 token 数（`estimateTokens(content)`），但只返回了 `msgID`，前端传递硬编码 `0`，导致用户消息的 token 要等 AI 完全回复完才显示。

## 改动方案

解耦用户消息 token 展示与 AI 流式流程：`SaveAIMessage` 一并返回 token 数，前端直接使用。

### 涉及文件

| 文件 | 改动类型 |
|------|----------|
| `app.go` | 修改 `SaveAIMessage` 返回结构 |
| `ai-chat.js` | 修改 `onSend()` 和 `handleResend()` |

### Step 1: 后端 — 修改 `SaveAIMessage` 返回结构（`app.go:2560-2576`）

**原因**：当前只返回 `msgID`，前端拿不到 token 数，只能传 0。

**改动**：
- 定义返回结构体 `SaveAIMessageResult`（含 `MsgID uint` 和 `Tokens int`）
- `SaveAIMessage` 方法返回类型从 `(uint, error)` 改为 `(SaveAIMessageResult, error)`
- 将已计算的 `estimateTokens(content)` 值通过结构体返回

```go
type SaveAIMessageResult struct {
    MsgID  uint `json:"msgID"`
    Tokens int  `json:"tokens"`
}

func (a *App) SaveAIMessage(sessionID uint, content string, role string) (SaveAIMessageResult, error) {
    tokens := estimateTokens(content)
    msg := services.Message{
        Role:    role,
        Content: content,
        Tokens:  tokens,
    }
    msgID, err := a.aiService.SaveAIMessage(sessionID, msg)
    if err != nil {
        return SaveAIMessageResult{}, err
    }
    return SaveAIMessageResult{MsgID: msgID, Tokens: tokens}, nil
}
```

### Step 2: 前端 — 修改 `onSend()`（`ai-chat.js:2049-2063`）

**原因**：从返回结果中提取 token 并传递给渲染函数。

**改动**：
- 从 `SaveAIMessage` 返回结果中提取 `msgID` 和 `tokens`
- 将 tokens 传给 `addMessage` 和 `createMsgActions`（替代硬编码 0）

```javascript
// 之前
userMsgId = await window.go.main.App.SaveAIMessage(activeSessionId, text, 'user');
addMessage(text, 'user', undefined, undefined, undefined, 0, userMsgId || undefined);
userMsgEl.appendChild(createMsgActions(text, 'user', undefined, 0));

// 之后
const result = await window.go.main.App.SaveAIMessage(activeSessionId, text, 'user');
const userMsgId = result?.msgID || 0;
const userTokens = result?.tokens || 0;
addMessage(text, 'user', undefined, undefined, undefined, userTokens, userMsgId || undefined);
userMsgEl.appendChild(createMsgActions(text, 'user', undefined, userTokens));
```

### Step 3: 前端 — 修改 `handleResend()`（`ai-chat.js:3640-3654`）

**原因**：与 `onSend()` 相同的逻辑，重新发送时也要提前展示 token。

**改动**：与 Step 2 相同的变更。

```javascript
// 之前
newUserMsgId = await window.go.main.App.SaveAIMessage(activeSessionId, content, 'user');
addMessage(content, 'user', undefined, undefined, undefined, 0, newUserMsgId || undefined);
newUserMsgEl.appendChild(createMsgActions(content, 'user', undefined, 0));

// 之后
const resendResult = await window.go.main.App.SaveAIMessage(activeSessionId, content, 'user');
const newUserMsgId = resendResult?.msgID || 0;
const resendTokens = resendResult?.tokens || 0;
addMessage(content, 'user', undefined, undefined, undefined, resendTokens, newUserMsgId || undefined);
newUserMsgEl.appendChild(createMsgActions(content, 'user', undefined, resendTokens));
```

### 不变的部分

- **AI 流式流程完全不变**：`CallAIStream`、`startStreaming`、流式回调等原封不动
- **`ai:stream-done` 事件更新 token 的逻辑保留**：AI 完成后仍会用完整上下文 token 数（含 system 提示词）更新显示，从 `用户自身 token` → `完整上下文 token`，更精确
- **`addMessage` 和 `createMsgActions` 函数不变**：它们已支持 tokens 参数
- **后端 `ai_service.go` 不变**：`SaveAIMessage` 和 `UpdateAIMessageTokens` 逻辑不动

## 最终流程

```
用户发送消息 → SaveAIMessage(保存 + 计算自身 token) → 返回 {msgID, tokens}
              → addMessage(tokens=真实值) → 立即显示 "8 tokens"
              → startStreaming() → AI 流式回复（独立进行）
              → AI 流完成 → OnDone：
                  1. 计算完整上下文 token
                  2. 更新 DB
                  3. 发射 ai:stream-done(userTokens=完整值)
              → 前端更新显示 "158 tokens"（更精确）
```

## 验证步骤

1. 构建项目 `wails build` 或前端 dev 模式
2. 发送一条用户消息，观察 token 是否立即显示（而非 AI 回复完才出现）
3. 验证 AI 回复完成后 token 是否更新为完整上下文值
4. 测试「重新发送」功能 token 是否也提前展示
5. 测试「新建会话」场景
