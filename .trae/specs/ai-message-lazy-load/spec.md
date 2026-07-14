# AI 消息懒加载与发送流程重构 Spec

## Why

当前架构中，前端维护完整的 `chatHistory` 数组，每次发送消息都将整个历史（含所有 user/assistant 消息）序列化后传给后端。随着对话变长，存在以下问题：

- **前端内存膨胀**：长对话（数百条）时 `chatHistory` 数组占用大量内存
- **冗余数据传输**：每次发送都跨 Wails Bridge 传输完整历史，实际模型需要的消息后端完全可以从 DB 获取
- **消息不一致风险**：前端 `chatHistory` 和后端 DB 可能不同步
- **首屏加载慢**：打开长对话会话，所有消息一次性加载和渲染

本 spec 的目标：**后端 DB 成为唯一的消息真相源，前端只维护"当前视图窗口"的消息子集**。

## What Changes

### 核心变化

1. **NEW**: 后端新增游标分页查询 `LoadAISessionMessagesPaginated`
2. **NEW**: 后端新增单条消息保存方法 `SaveUserMessage` / `SaveAssistantMessage`，替代现有的批量 `SaveAIMessages`
3. **MODIFIED**: `CallAIStream` 签名简化 — 不再接收 `messages []services.Message`（历史消息），改为只接收 `content string`（当前用户消息文本）
4. **NEW**: 后端内部新增 `loadSessionMessagesForModel` 方法：从 DB 加载完整会话消息，构建最终发给模型的 messages 数组
5. **MODIFIED**: 前端 `chatHistory` 从"完整历史"变为"视口窗口"，初始仅加载最近 3 轮（6 条）消息
6. **MODIFIED**: 前端滚动到消息列表顶部时触发懒加载，追加更早的消息
7. **MODIFIED**: 前端发送/重发/再生流程适配新架构
8. **MODIFIED**: 切换会话时也使用懒加载初始 6 条

### 架构对比

#### 迁移前
```
前端:
  chatHistory = [msg1, msg2, ..., msgN]  // 全部消息
  onSend():
    chatHistory.push(userMsg)
    CallAIStream(chatHistory, ...元数据)   // 传完整历史

后端 CallAIStream(messages, ...):
  注入 system → 构建 messages → AI API
  流完成 → 保存 user + assistant 到 DB
```

#### 迁移后
```
前端:
  displayedMessages = [msg_{n-5}, ..., msg_n]  // 仅最近 6 条（视口窗口）
  onSend():
    CallAIStream(sessionID, content, ...元数据)  // 仅传当前消息

后端 CallAIStream(sessionID, content, ...):
  1. 保存 user 消息到 DB → 获取 userMessageID（调用 SaveUserMessage）
  2. 从 DB 加载该会话全部消息
  3. 注入 system 上下文
  4. 完整 messages → AI API
  5. 若 AI 调用失败 → 删除刚保存的 user 消息（回滚）
  6. 流完成 → 保存 assistant 到 DB → 获取 assistantMessageID（调用 SaveAssistantMessage）
  7. 通过 ai:stream-done 返回 userMessageID + assistantMessageID
```

## Impact

- Affected code:
  - `internal/services/ai_service.go` — 新增 `LoadSessionMessagesPaginated` 分页查询方法；新增 `SaveUserMessage` / `SaveAssistantMessage` 单条保存方法
  - `app.go` — `CallAIStream` 签名变更（移除 `messages`，改为 `sessionID` + `content`）；预处理逻辑不再从参数取消息，改为从 DB 加载；流完成保存适配单条保存
  - `frontend/src/js/ai-chat.js` — 重写 `chatHistory` 为视口窗口、修改 `onSend`/`startStreaming`、新增滚动懒加载逻辑
- 无需新增文件

## ADDED Requirements

### Requirement: 后端分页加载消息

系统 SHALL 提供游标分页查询，支持前端滚动懒加载。

#### Scenario: 分页查询
- **WHEN** 前端调用 `LoadAISessionMessagesPaginated(sessionID uint, cursorID uint, limit int)`
- **THEN** 返回该会话中 `id < cursorID` 的消息（按 `created_at DESC` 排序，最多 `limit` 条）
- **AND** 返回结果附带 `hasMore` 标记，指示是否还有更早的消息
- **AND** 当 `cursorID == 0` 时，返回该会话最新的 `limit` 条消息

#### 响应结构
```go
type PaginatedMessages struct {
    Messages []AIMessage `json:"messages"`
    HasMore  bool        `json:"hasMore"`
}
```

### Requirement: 后端新增单条消息保存方法

系统 SHALL 新增单条消息保存方法，替代现有的批量 `SaveAIMessages`，以支持分步保存（先 user 后 assistant）并获取每条消息的 ID。

#### Scenario: 保存 user 消息
- **WHEN** 发送模式下需要保存用户消息
- **THEN** 调用 `SaveUserMessage(sessionID uint, content string) (uint, error)` 创建一条 `{role: "user", content: content, session_id: sessionID}` 的消息并保存到 DB
- **AND** 返回新创建消息的 ID（即 `userMessageID`）
- **AND** 该方法将 user 消息的 `Tokens`、`ThinkingElapsed`、`TotalElapsed` 等字段初始化为 0

#### Scenario: 保存 assistant 消息
- **WHEN** 流完成后需要保存 assistant 回复
- **THEN** 调用 `SaveAssistantMessage(sessionID uint, content string, reasoningContent string, tokens int, thinkingElapsed float64, totalElapsed float64, searchSources string, recallCards string) (uint, error)` 创建一条 `{role: "assistant"}` 的消息
- **AND** 返回新创建消息的 ID（即 `assistantMessageID`）
- **AND** 保存后更新会话的 `updated_at` 时间戳

### Requirement: 后端消息发送新流程

系统 SHALL 提供新的 `CallAIStream` 实现，由后端自行管理消息历史。

#### Scenario: 发送消息（正常模式）
- **WHEN** 前端调用 `CallAIStream(sessionID uint, content string, ...元数据)`（不再传 `messages` 数组）
- **THEN** 后端调用 `SaveUserMessage(sessionID, content)` 将用户消息保存到 DB → 获取 `userMessageID`
- **THEN** 后端从 DB 加载该会话的**所有消息**（`LoadSessionMessages`，按 `created_at ASC`）
- **THEN** 执行现有的 system 上下文注入流程（6 项注入）
- **THEN** 将完整的 messages 数组发送给 AI API
- **THEN** 流式输出回前端
- **THEN** 流完成后，调用 `SaveAssistantMessage(...)` 将 assistant 回复保存到 DB → 获取 `assistantMessageID`
- **AND** `ai:stream-done` 事件返回 `userMessageID` 和 `assistantMessageID`

#### Scenario: AI 调用失败回滚
- **WHEN** AI API 调用失败（网络错误、API 返回错误、超时等）
- **THEN** 后端删除之前通过 `SaveUserMessage` 保存的 user 消息（调用 `DeleteAIMessage(userMessageID)`）
- **AND** 通过 `ai:stream-error` 事件向前端推送错误信息
- **AND** 前端收到错误后，从 `displayedMessages` 中移除对应的 user 消息占位
- **RATIONALE** 避免 DB 中残留孤立的 user 消息。如果用户重试，系统会重新保存 user 消息并重试完整流程

### Requirement: 前端消息视口窗口

前端 SHALL 仅维护当前可见的消息子集，而非完整历史。

#### Scenario: 初始化加载
- **WHEN** 用户切换到一个会话（或打开 AI 助手）
- **THEN** 前端调用 `LoadAISessionMessagesPaginated(sessionID, 0, 6)` 加载最近 6 条消息
- **THEN** 前端将返回的消息存入 `displayedMessages` 数组（替代原来的 `chatHistory`）
- **AND** 如果有 `hasMore === true`，在消息列表顶部显示"加载更多"提示

#### Scenario: 滚动懒加载
- **WHEN** 用户滚动到消息列表顶部（或点击"加载更多"）
- **THEN** 前端以当前最早消息的 ID 作为 `cursorID`，调用分页接口加载更早的 6 条
- **THEN** 前端将加载的消息 `unshift` 到 `displayedMessages` 数组头部
- **THEN** 前端保持滚动位置不变（用户在视觉上看到消息列表向上扩展）
- **AND** 具体实现：加载前记录 `scrollHeight`，DOM 插入后将 `scrollTop` 设置为新 `scrollHeight - 旧 scrollHeight + 原 scrollTop`

#### Scenario: 发送消息后更新（含自动滚动策略）
- **WHEN** 流式完成后端通过 `ai:stream-done` 事件推送新消息 ID 和内容
- **THEN** 前端将新 user 消息和 assistant 回复 `push` 到 `displayedMessages` 尾部
- **THEN** 前端判断用户当前是否在消息列表底部（`scrollTop + clientHeight >= scrollHeight - threshold`）
- **AND** 如果用户在底部 → 自动滚动到底部显示新消息
- **AND** 如果用户已向上滚动查看历史 → **不**自动滚动，仅在底部显示"[新消息]"提示按钮，用户点击后滚动到底部

### Requirement: 前端适配重发/再生/删除

#### Scenario: 重新发送（编辑用户消息后重发）
- **WHEN** 用户编辑了一条历史 user 消息并点击重发
- **THEN** 前端调用 `DeleteAIMessagesAfter(msgID)` 删除该消息之后的所有消息
- **THEN** 前端从 `displayedMessages` 中移除被删消息之后的所有条目
- **THEN** 前端调用 `CallAIStream(sessionID, editedContent, ...)` 发送新消息
- **AND** 后端从 DB 加载剩余消息（不含已删的），构建消息列表发给模型

#### Scenario: 重新生成（Regenerate）
- **WHEN** 用户对最后一条 assistant 消息点击重新生成
- **THEN** 前端调用后端 `DeleteAIMessage(msgID)` 删除最后一条 assistant 消息
- **THEN** 前端从 `displayedMessages` 中移除最后一条 assistant 消息
- **THEN** 前端从 `displayedMessages` 中获取最后一条 user 消息的 `content`，调用 `CallAIStream(sessionID, lastUserMsgContent, ...)`，标记 `isRegenerate=true`
- **AND** 后端在 `isRegenerate=true` 时，不重复保存 user 消息（因 DB 已存在），直接加载全部消息 → 调模型
- **AND** 注意：连续多次 regenerating 时，每次都要从当前最新的 `displayedMessages` 中取 user 消息 content

#### Scenario: 删除消息
- **WHEN** 用户删除某条消息及其之后的所有内容
- **THEN** 前端调用后端 `DeleteAIMessagesAfter(msgID)`
- **THEN** 前端从 `displayedMessages` 中移除该消息及其之后的所有条目
- **THEN** 前端重新从 DB 加载最新消息刷新显示（确保与后端一致）

### Requirement: 切换会话适配

#### Scenario: 切换会话
- **WHEN** 用户切换到另一个会话
- **THEN** 前端先取消当前正在进行的流式请求（调用取消函数 + 重置 `isStreaming`）
- **THEN** 前端清空笔记引用、技能、上传文件等状态（复用现有副作用逻辑）
- **THEN** 前端加载新会话的 session config（复用现有逻辑）
- **THEN** 前端使用懒加载方式加载该会话的最新 6 条消息（调用 `LoadAISessionMessagesPaginated(sessionID, 0, 6)`）
- **THEN** 前端清空 `displayedMessages`，填入新会话的消息
- **THEN** 前端更新侧栏高亮和会话标题

## MODIFIED Requirements

### Requirement: CallAIStream 函数签名

#### 旧签名
```go
func (a *App) CallAIStream(streamGen int, messages []services.Message, thinkingEnabled bool,
    searchSources []string, cardRecallEnabled bool, sessionID uint, isRegenerate bool,
    skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint,
    followUpRefContent string, uploadedFiles []AIChatFileResult)
```

#### 新签名
```go
func (a *App) CallAIStream(streamGen int, sessionID uint, content string, thinkingEnabled bool,
    searchSources []string, cardRecallEnabled bool, isRegenerate bool,
    skillIds []string, referencedNoteIDs []uint, roleplayNoteIDs []uint,
    followUpRefContent string, uploadedFiles []AIChatFileResult)
```

变更：删除了 `messages []services.Message` 参数，新增 `content string` 参数（当前用户消息内容）。

### Requirement: 后端消息预处理流程

**发送模式下（`isRegenerate=false`）**：
1. 调用 `SaveUserMessage(sessionID, content)` 保存 user 消息到 DB → 获取 `userMessageID`
2. 从 DB 加载该会话全部消息（`created_at ASC`）
3~8. 现有 system 上下文注入流程（不变）
9. 完整 messages → AI API
10. **如果 AI API 调用失败** → 调用 `DeleteAIMessage(userMessageID)` 回滚，通过 `ai:stream-error` 通知前端
11. 流完成 → 调用 `SaveAssistantMessage(...)` 保存 assistant 到 DB → 获取 `assistantMessageID`
12. 通过 `ai:stream-done` 返回 `userMessageID` + `assistantMessageID`

**再生模式下（`isRegenerate=true`）**：
1. 从 DB 加载该会话全部消息（不新增 user 消息）
2~8. 现有 system 上下文注入流程（不变）
9. 完整 messages → AI API
10. **如果 AI API 调用失败** → 直接通过 `ai:stream-error` 通知前端（无需回滚，因为未新增任何消息）
11. 流完成 → 调用 `SaveAssistantMessage(...)` 保存 assistant 到 DB（替换旧的 assistant 回复）
12. 通过 `ai:stream-done` 返回 `assistantMessageID`（`userMessageID` 复用已有记录的 ID）

### Requirement: 前端 `chatHistory` → `displayedMessages`

前端将 `chatHistory` 数组重命名为 `displayedMessages`，语义从"完整历史"变为"当前视口可见的消息"。

- `displayedMessages` 初始最多 6 条
- 滚动顶部懒加载追加更早消息
- 发送新消息后追加新消息
- 不再作为发送消息时的数据源（发送时只传 `sessionID` + `content`）

### Requirement: `ai:stream-done` 事件扩展

`ai:stream-done` 事件负载新增字段，确保前端能正确更新 `displayedMessages`：

```json
{
  "userMessageID": 101,       // 新增：刚保存的 user 消息 ID（再生模式下为 0）
  "assistantMessageID": 102,  // 新增：刚保存的 assistant 消息 ID
  "content": "...",           // 已有
  "tokens": 123,              // 已有
  ...
}
```

### Requirement: `ai:stream-error` 事件扩展

`ai:stream-error` 事件在发送模式下新增 `userMessageID` 字段，便于前端清理已保存但未完成的 user 消息占位：

```json
{
  "userMessageID": 101,       // 新增：需要回滚的 user 消息 ID（发送模式）
  "error": "API request failed",
  ...
}
```

## REMOVED Requirements

### Requirement: 前端 `chatHistory` 作为消息发送源

**Reason**: 消息历史的管理职责完全移至后端 DB，前端不再需要将历史消息传给后端。
**Migration**: 删除 `startStreaming()` 中构建 `messages` 数组并传给 `CallAIStream` 的逻辑；改为仅传 `sessionID` + `content`。

### Requirement: 后端 `SaveAIMessages` 批量保存

**Reason**: 新流程需要分步保存（先 user 再 assistant），且需要获取每条消息的 ID。批量保存无法满足需求。
**Migration**: 用 `SaveUserMessage` 和 `SaveAssistantMessage` 替代。`SaveAIMessages` 保留不删除（可能在其他地方使用），但新流程不再调用。
