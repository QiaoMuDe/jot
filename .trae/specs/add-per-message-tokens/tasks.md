# Tasks

- [x] Task 1: 后端 — AIMessage 模型新增 `tokens` 字段 + services.Message 同步
  - 修改 `internal/models/ai_message.go`，新增 `Tokens int` 字段（gorm 默认 0）
  - 修改 `internal/services/ai_service.go` 中 `Message` 结构体，新增 `Tokens int` 字段
  - GORM AutoMigrate 自动新增列，无需手动迁移

- [x] Task 2: 后端 — SaveAIMessages 写入 tokens 字段 + LoadAISessionMessages 返回 tokens
  - `SaveAIMessages()` 在 `Create(&m)` 时填入 `Tokens: msg.Tokens`
  - `LoadAISessionMessages()` 的查询无需改动，tokens 列自动映射到 `AIMessage.Tokens`

- [x] Task 3: 前端 — chatHistory 增加 tokens 字段 + 加载/保存流程同步
  - 切换会话构建 chatHistory 时保留 tokens（`msgs.map(msg => ({ role, content, tokens: msg.tokens || 0 }))`）
  - `saveSessionMessages()` 发送时，roundMessages 每个元素增加 `tokens` 字段
  - 计算规则：用户消息 tokens = estimateTokens(content) + estimateTokens(systemContent)；AI 消息 tokens = estimateTokens(content)
  - systemContent 包含：cachedRefContext + followUpRef(前500字) + getSkillSystemPrompts()

- [x] Task 4: 前端 — updateContextSize 改为累加 tokens 而非实时估算
  - `const total = chatHistory.reduce((sum, msg) => sum + (msg.tokens || 0), 0);`
  - 切换会话时同样用 tokens 累加的方式计算总 token 数
  - 保留 `AISession.ContextTokens` 的持久化逻辑

- [x] Task 5: 前端 — AI 消息左下角展示本条 token 数
  - `createMsgActions()` 中，AI 消息的 `.ai-msg-time` 标签从 `"⏱ X.Xs"` 改为 `"⏱ X.Xs · N tokens"`（tokens > 0 时）
  - 无需新增 CSS 样式（复用 `.ai-msg-time`）

# Task Dependencies

- [Task 1] 必须先于 [Task 2]
- [Task 2] 必须先于 [Task 3]
- [Task 4]、[Task 5] 可在 [Task 3] 完成后并行进行
