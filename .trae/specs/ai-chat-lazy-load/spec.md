# AI 会话消息懒加载架构改造 Spec

## Why

当前每次发送 AI 消息时，前端将整个会话的 `chatHistory[]`（可能数百条）通过 Wails RPC 全量传给后端，后端再拼接上下文调用 AI 模型。数据从前端传回后端的路径属于冗余——后端自己就有 DB，完全可以直接从数据库加载历史。同时，首次切换会话时全量加载所有消息，对长会话（50+ 条）有性能开销。

**核心目标**：将 `chatHistory` 从"API 上下文 + DOM 渲染源"双重职责中解放出来，改为后端自主管理上下文，前端只负责渲染和交互。

## What Changes

### 后端改动

1. **`CallAIStream` 签名修改（BREAKING）** — 不再接收 `messages []services.Message` 参数，改为从 DB 自行加载。新增 `CallAIStream(sessionID uint, userText string, ...metadata)` 和 `CallAIStreamRegenerate(sessionID uint, ...metadata)`，前者保存用户消息后流式，后者复用会话中最后一条用户消息。

2. **新增 `LoadAISessionMessagesPaginated`** — 分页加载消息，支持 `limit` + `beforeID`（游标分页，基于 ID 降序），返回按 `created_at ASC` 排序的指定条数消息。

3. **新增 `TruncateAISessionAtMessage`** — 删除指定消息及该消息之后的所有消息（用于删除/重发操作）。

4. **新增 `TruncateAISessionAfterMessage`** — 删除指定消息之后的所有消息，保留该消息本身（用于编辑/重新生成操作）。

5. **`ai:stream-done` 事件扩展** — 额外携带 `userMsgID` 和 `assistantMsgID`（新入库消息的数据库 ID），供前端挂载到 DOM 上。

6. **新增 `GetSessionContextTokens`** — 返回指定会话的 `context_tokens` 字段值，供前端 Token 显示。

### 前端改动

7. **`chatHistory` 变量降级** — 不再作为 API 上下文发送，仅作为渲染缓冲区（只存当前已加载的可见消息片段）。所有直接操作 `chatHistory.splice` 的代码改为基于 `msgID` 的后端调用。

8. **`onSend()` 重构** — 不再构建 `chatHistory` 传给后端，改为调用新签名 `CallAIStream(sessionID, userText, 元数据)`。

9. **`switchSession()` 重构** — 改为调用 `LoadAISessionMessagesPaginated` 加载最近 6 条消息，渲染后监听滚动到顶部加载更早消息。

10. **消息 DOM 挂载 `data-msg-id`** — 所有渲染的消息气泡都附带 `dataset.msgId`，供编辑/删除/重发/再生操作读取。

11. **编辑/删除/重发/再生重构** — 从 `chatHistory.splice` 改为基于 `msgID` 调用 `TruncateAISessionAtMessage` / `TruncateAISessionAfterMessage`，然后调用新 `CallAIStream` 或重新加载。

12. **`updateContextSize()` 重构** — 不再遍历 `chatHistory`，改为从 `window._sessionTokens` 缓存或调用 `GetSessionContextTokens` 读取。

13. **删除 `_pendingTokenSync` 相关逻辑** — 编辑后不再需要前端手动同步 token，后端自主管理。

### 后端保持不变

- `SaveAIMessages` — 仍被 `CallAIStream` 内部使用（追加写入，外部不直接调用）
- `ClearAISessionMessages` — 仍被「清空会话」按钮使用
- `ReplaceAISessionMessages` — 编辑/删除/重发/再生不再使用，但保留不删

## Impact

- **Affected specs**: AI 对话消息管理、消息发送、会话切换、消息编辑/删除/重发/再生、Token 显示
- **Affected code**:
  - `internal/services/ai_service.go` — 新增 4 个方法，修改 1 个
  - `app.go` — 修改 `CallAIStream`，新增 3 个绑定方法
  - `frontend/src/js/ai-chat.js` — 重构 `chatHistory`、`onSend`、`switchSession`、4 个操作函数、`updateContextSize` 等 ~15 处
  - `frontend/src/js/ai-chat.js` — 删除 `_pendingTokenSync`、`estimateTokens`（前端死代码）

## ADDED Requirements

### Requirement: CallAIStream 新签名

The system SHALL provide a new `CallAIStream` that loads message history from DB instead of from frontend.

#### Scenario: 正常发送消息
- **WHEN** user sends a new message
- **THEN** frontend calls `CallAIStream(sessionID, userText, metadata...)`
- **AND** backend saves the user message to DB
- **AND** backend loads all existing messages from DB
- **AND** backend builds 8-step context
- **AND** backend calls AI model
- **AND** backend saves assistant message to DB
- **AND** backend emits `ai:stream-done` with `userMsgID`, `assistantMsgID`, `totalTokens`

#### Scenario: 重新生成（regenerate）
- **WHEN** user clicks regenerate on an AI message
- **THEN** frontend calls `TruncateAISessionAfterMessage(sessionID, msgID)` (msgID of the user message before the AI)
- **AND** backend deletes all messages after that user message
- **AND** frontend calls `CallAIStreamRegenerate(sessionID, metadata...)`
- **AND** backend loads remaining messages from DB
- **AND** backend uses the last user message as input
- **AND** backend calls AI, saves response, emits stream-done

### Requirement: 分页消息加载

The system SHALL provide paginated loading of AI session messages.

#### Scenario: 切换会话
- **WHEN** user switches to a session
- **THEN** frontend calls `LoadAISessionMessagesPaginated(sessionID, limit=6, beforeID=0)`
- **AND** backend returns the latest 6 messages (or fewer if session has less)
- **AND** frontend renders them in order

#### Scenario: 加载更早消息
- **WHEN** user scrolls to the top of the message list
- **THEN** frontend calls `LoadAISessionMessagesPaginated(sessionID, limit=6, beforeID=oldestMsgID)`
- **AND** backend returns 6 messages older than the current oldest
- **AND** frontend prepends them to the message list

### Requirement: 基于 msgID 的消息操作

The system SHALL provide truncation operations based on message ID.

#### Scenario: 编辑消息
- **WHEN** user edits a message and confirms
- **THEN** frontend calls `TruncateAISessionAfterMessage(sessionID, msgID)`
- **AND** backend deletes all messages after the specified message
- **AND** frontend calls `CallAIStream(sessionID, newUserText, metadata...)`
- **AND** backend generates new AI response

#### Scenario: 删除消息
- **WHEN** user deletes a message
- **THEN** frontend calls `TruncateAISessionAtMessage(sessionID, msgID)`
- **AND** backend deletes the specified message and all messages after it
- **AND** frontend reloads the session

## REMOVED Requirements

### Requirement: 前端全量发送 chatHistory

**Reason**: 后端已能从 DB 自行加载历史，无需前端传递。
**Migration**: `CallAIStream` 签名改为 `(sessionID, userText, metadata...)`，前端不再传 `messages[]`。

### Requirement: `_pendingTokenSync` 编辑后同步 token

**Reason**: 后端自主管理消息保存，不再需要前端手动同步 token。
**Migration**: 删除 `_pendingTokenSync` 变量及相关逻辑。