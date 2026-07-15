# AI 会话消息替换操作后端迁移 Spec

## Why

目前前端在 编辑/删除/重发/重新生成 四条消息操作中，都按「先 `ClearAISessionMessages` 清空全部，再 `SaveAIMessages` 全量覆盖」两步写入 DB。这两步调用逻辑相同、配对使用，理应由后端提供原子方法封装，减少前端重复代码和两次网络往返。

## What Changes

### 后端新增

1. **`AIService.ReplaceAISessionMessages(sessionID, messages)`** — 在事务中原子执行：清空会话全部消息 → 批量写入新消息，保持与 `SaveAIMessages` 一致的 auto-title 行为
2. **`App.ReplaceAISessionMessages(sessionID, messages)`** — Wails 绑定方法，调用上述服务方法

### 后端保持不变

- `ClearAISessionMessages` — 仍被「清空会话」按钮调用（仅清空，不写入）
- `SaveAIMessages` — 仍被 `stream-done` 回调使用（追加写入，非替换）
- `UpdateLastUserMessageTokens` — 仍被编辑后 token 同步使用

### 前端修改

四个操作各自将两行调用替换为一行：

| 操作 | 旧调用（2 次） | 新调用（1 次） |
|------|--------------|--------------|
| `applyEdit` (L3348-3350) | `ClearAISessionMessages` + `SaveAIMessages` | `ReplaceAISessionMessages` |
| `handleDeleteMsg` (L3399-3401) | 同上 | 同上 |
| `handleRegenerate` (L3438-3440) | 同上 | 同上 |
| `handleResend` (L3466-3468) | 同上 | 同上 |

### 不移改的

- 「清空会话」按钮（L388）仍用 `ClearAISessionMessages`，语义符合「只清不写」
- `stream-done` 回调（L2448）仍用 `SaveAIMessages`，语义符合「追加」

## Impact

- **Affected specs**: AI 对话消息管理
- **Affected code**:
  - `internal/services/ai_service.go` — 新增方法
  - `app.go` — Wails 绑定
  - `frontend/src/js/ai-chat.js` — 4 处调用替换

## ADDED Requirements

### Requirement: ReplaceAISessionMessages

The system SHALL provide an atomic replace operation for AI session messages.

#### Scenario: Success case
- **WHEN** backend receives `ReplaceAISessionMessages(sessionID, messages)`
- **THEN** it SHALL delete all existing messages for that session
- **AND** insert all provided messages
- **AND** wrap both operations in a single DB transaction
- **AND** update session `updated_at`
- **AND** generate auto-title if current title is "新对话"
- **RETURN** no error

#### Scenario: Transaction rollback
- **WHEN** any step in the replace operation fails
- **THEN** the entire operation SHALL rollback
- **AND** the session messages SHALL remain unchanged
- **RETURN** error
