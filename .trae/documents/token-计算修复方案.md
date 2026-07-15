# Token 计算修复方案

## 背景

上下文迁移到后端后，token 计算存在两个问题：

1. **Total Token 显示偏小** — `context_tokens` 被每轮覆盖为单轮值，而非累计值
2. **用户消息 Token 跳变** — stream-done 显示 `userTokens`（含系统上下文），DB 存的是 `estimateTokens(userText)`（纯文本），切换会话重载后显示值变小

## 修改方案

### 后端 AIService 新增方法

#### `SumSessionTokens(sessionID uint) (int, error)`

- 文件: `internal/services/ai_service.go`
- SQL: `SELECT COALESCE(SUM(tokens), 0) FROM ai_messages WHERE session_id = ?`
- 返回该会话**所有消息**的 `tokens` 累计和
- 用于 stream-done 后计算真正的累计 total token

#### `UpdateAIMessageTokens(id uint, tokens int) error`

- 文件: `internal/services/ai_service.go`
- SQL: `UPDATE ai_messages SET tokens = ? WHERE id = ?`
- 用于 stream-done 后将用户消息的 `tokens` 字段更新为完整上下文 token 数

### 后端 app.go 修改

#### `CallAIStream`（~L1941-1942）

**改前**:
```go
_ = a.aiService.UpdateSessionContextTokens(sessionID, totalTokens)
```

**改后**:
```go
// 更新用户消息的 tokens 为完整上下文 token 数（含 system）
_ = a.aiService.UpdateAIMessageTokens(userMsgID, userTokens)
// 重新计算会话累计 token 并持久化
accumulated, _ := a.aiService.SumSessionTokens(sessionID)
_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)
```

#### `CallAIStreamRegenerate`（~L2353）

**改前**:
```go
_ = a.aiService.UpdateSessionContextTokens(sessionID, totalTokens)
```

**改后**:
```go
// 重新计算会话累计 token 并持久化
// 再生模式不创建新用户消息，无需更新 user 消息的 tokens
accumulated, _ := a.aiService.SumSessionTokens(sessionID)
_ = a.aiService.UpdateSessionContextTokens(sessionID, accumulated)
```

### 前端无改动

| 场景 | 当前行为 | 改后行为 | 一致性 |
|------|---------|---------|--------|
| stream-done 更新总 token | `updateContextSize` 读 `GetSessionContextTokens` | 后端存了累计值 → 显示正确累计值 | ✓ |
| 切换会话加载总 token | 同上 | 同上 | ✓ |
| stream-done 用户气泡 token | 显示 `userTokens`（system+user） | `userTokens` 值不变，但 DB 已同步更新 | ✓ |
| 切换会话重载用户气泡 token | 显示 `msg.tokens = estimateTokens(userText)` | 显示 `msg.tokens = userTokens`（已更新） | ✓ |

## 涉及文件

| 文件 | 改动 |
|------|------|
| `internal/services/ai_service.go` | 新增 `SumSessionTokens`、`UpdateAIMessageTokens` |
| `app.go` | `CallAIStream` (L1941-1942)、`CallAIStreamRegenerate` (L2353) 各约 3 行 |

## 验证步骤

1. 发送一条新消息 → 确认头部 total token 显示数量合理（应 ≈ 历史所有 message tokens 累计）
2. 继续发送多条消息 → 确认 total token 随轮次递增
3. 切换会话再切回 → 确认 total token 值不变
4. 观察用户消息气泡的 token 数 → 切换前后一致
5. 重新生成 → 确认 total token 正确（不翻倍）
6. 删除消息 → 确认 total token 不变化（仅删除 assistant 时）或减少（删除 user 时）
