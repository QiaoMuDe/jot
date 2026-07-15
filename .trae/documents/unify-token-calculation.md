# 统一 Token 计算方法计划

## 问题

当前 `app.go` 中有 **4 处重复**的 "累计 system 消息 + 最后一条 user 消息" 的循环计算代码，结构完全一样，改动需同步改 4 处，易遗漏。

## 当前状态

现有工具函数：

```go
// app.go:2432 — 输入单条文本，返回估算 token 数
func estimateTokens(text string) int { ... }
```

4 处重复的模式完全一样：

```go
userTokens := 0
for _, msg := range messages {
    if msg.Role == "system" {
        userTokens += estimateTokens(msg.Content)
    }
}
for i := len(messages) - 1; i >= 0; i-- {
    if messages[i].Role == "user" {
        userTokens += estimateTokens(messages[i].Content)
        break
    }
}
// ↓ 以下 3 行各不同（msgID 变量名不同），不算重复
_ = a.aiService.UpdateAIMessageTokens(msgID, userTokens)
accumulated, _ := a.aiService.SumSessionTokens(sessionID)
_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)
```

分布位置：

| # | 位置 | 行号 | 场景 |
|---|------|------|------|
| 1 | `CallAIStream.onComplete` | ~1887 | 流式成功 |
| 2 | `CallAIStream.onError` | ~1954 | 流式失败 |
| 3 | `CallAIStreamRegenerate.onComplete` | ~2333 | 重新生成成功 |
| 4 | `CallAIStreamRegenerate.onError` | ~2397 | 重新生成失败 |

## 方案

### 新增纯计算函数：`estimateUserTokens(messages) int`

只做计算，不做 DB 更新。输入 `[]services.Message`，返回 system 消息 + 最后一条 user 消息的估算 token 总和。

```go
// estimateUserTokens 计算会话中 system 消息与最后一条 user 消息的估算 token 数
func estimateUserTokens(messages []services.Message) int {
    tokens := 0
    for _, msg := range messages {
        if msg.Role == "system" {
            tokens += estimateTokens(msg.Content)
        }
    }
    for i := len(messages) - 1; i >= 0; i-- {
        if messages[i].Role == "user" {
            tokens += estimateTokens(messages[i].Content)
            break
        }
    }
    return tokens
}
```

### 4 处调用替换

每处的循环计算块（~13 行）缩为 1 行计算 + 3 行 DB 更新（共 4 行）：

```go
// before (13 行重复代码)
userTokens := 0
for _, msg := range messages { ... }  // system 循环
for i := len(messages) - 1; i >= 0; i-- { ... }  // user 消息
_ = a.aiService.UpdateAIMessageTokens(msgID, userTokens)
accumulated, _ := a.aiService.SumSessionTokens(sessionID)
_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)

// after (4 行，计算部分统一)
userTokens := estimateUserTokens(messages)
_ = a.aiService.UpdateAIMessageTokens(msgID, userTokens)
accumulated, _ := a.aiService.SumSessionTokens(sessionID)
_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)
```

### 影响范围

| 文件 | 改动 | 行数 |
|------|------|------|
| `app.go` | 新增 `estimateUserTokens()` 函数 | +12 行 |
| `app.go` | 4 处重复块各缩为 4 行 | -36 行 |

- 无 DB schema、Wails binding、services 层、前端变更

## 验证步骤

1. `go build ./...` 编译通过
2. 正常流式完成 → token 正确显示
3. 流式出错 → token 正确显示，无空消息写入
4. 重新生成完成/出错 → token 正确
