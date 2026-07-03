# 每条消息记录 Token 消耗 & AI 消息展示单个 Token 数 Spec

## Why

当前 Token 总数只算了纯对话文本，未计入引用笔记、联网搜索、卡片召回等系统上下文，且总 Token 不会保存到库，切换会话时重新估算。用户要求改为每条消息独立记录 Token 消耗，总数为各消息之和，AI 消息在左下角额外展示本条消耗的 Token 数。

## What Changes

- **AIMessage 模型** 新增 `tokens int` 字段（数据库 `tokens` 列）
- **services.Message 结构体** 同步新增 `Tokens int` 字段
- **SaveAIMessages** 将每条消息的 Tokens 写入数据库
- **LoadAISessionMessages** 返回 tokens 字段
- **前端 chatHistory** 每条消息对象增加 `tokens` 字段
- **前端 saveSessionMessages()** 发送时携带每条消息的 tokens
- **前端 updateContextSize()** 改为只从 chatHistory 累加各消息的 tokens（不再实时估算）
- **前端会话切换** 从数据库加载 tokens 显示（与已有逻辑一致）
- **前端 addMessage()/createMsgActions()** 在 AI 消息的耗时标签旁增加 Token 数显示（如 "⏱ 5.7s · 342 tokens"）
- **context_tokens 总数字段与持久化保持不变**

## Impact

- 修改模型: `internal/models/ai_message.go`
- 修改传输结构: `internal/services/ai_service.go` 中 `Message` 结构体
- 修改保存逻辑: `internal/services/ai_service.go` 中 `SaveAIMessages()`
- 修改前端核心: `frontend/src/js/ai-chat.js`（chatHistory、saveSessionMessages、updateContextSize、addMessage、createMsgActions）
- 不变: `app.go` 无需改动（Wails 绑定无签名变更）；`AISession.ContextTokens` 和页面头部 Token 展示保留

## ADDED Requirements

### Requirement: 每条消息记录 tokens

系统 SHALL 为每条消息独立记录其消耗的 token 数，并持久化到数据库。

#### Scenario: 用户发送消息
- **WHEN** 用户在 AI 对话中输入一条消息并发送
- **THEN** system 消息体（引用笔记 + 联网搜索 + 卡片召回 + 技能提示词 + 追问引用）计入该用户消息的 tokens 字段
- **AND** 该用户消息和 AI 回复消息一并保存到数据库，各自带上 tokens 值

#### Scenario: AI 回复完成
- **WHEN** AI 回复流式输出完成
- **THEN** AI 回复内容调用 `estimateTokens()` 计算 token 数
- **AND** 该 token 数写入该 AI 消息的 tokens 字段
- **AND** 保存到数据库

#### Scenario: 切换会话
- **WHEN** 用户切换到另一个会话
- **THEN** 从数据库加载每条消息的 tokens 字段
- **AND** 页面头部显示的总 token = 所有消息 tokens 之和

### Requirement: AI 消息展示单个 Token 数

系统 SHALL 在每条 AI 消息左下角展示本条回复消耗的 token 数。

#### Scenario: AI 消息渲染
- **WHEN** AI 消息渲染完毕，左下角已有耗时标签（`.ai-msg-time`）
- **THEN** 在耗时标签中追加 token 数，格式为 `"⏱ 5.7s · 342 tokens"`（tokens > 0 时显示）
- **AND** 使用相同的 `.ai-msg-time` 样式（灰小字，不换行）

## MODIFIED Requirements

### Requirement: updateContextSize 改为累加而非估算

- **修改前**: `const total = chatHistory.reduce((sum, msg) => sum + estimateTokens(msg.content), 0);`
- **修改后**: `const total = chatHistory.reduce((sum, msg) => sum + (msg.tokens || 0), 0);`
- 切换会话时也从数据库加载的 tokens 字段累加
- `AISession.ContextTokens` 的持久化逻辑不变（`updateContextSize` 仍会调用 `UpdateSessionContextTokens`）

### Requirement: saveSessionMessages 携带 tokens

- saveSessionMessages(roundMessages) 中，roundMessages 数组每个元素增加 `tokens` 字段
- 用户消息: tokens = estimateTokens(userContent) + estimateTokens(systemContent)
- AI 消息: tokens = estimateTokens(aiContent)
- systemContent 包含: cachedRefContext + followUpRef(前500字) + getSkillSystemPrompts() + 联网搜索结果 + 卡片召回结果
- 后端 services.Message 结构体新增 `Tokens int` 字段以接收该值

### Requirement: LoadAISessionMessages 返回 tokens

后端 `LoadAISessionMessages()` 返回的 JSON 中，每条消息包含 `tokens` 字段（数据库 `tokens` 列的值）。
前端切换会话时构建 chatHistory 保留 tokens 字段：
```js
chatHistory = msgs.map(msg => ({
    role: msg.role,
    content: msg.content,
    tokens: msg.tokens || 0
}));
```

## REMOVED Requirements

无。
