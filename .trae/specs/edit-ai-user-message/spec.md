# 用户消息编辑功能 Spec

## Why

AI 对话中用户发送消息后经常需要修正错别字、补充细节或调整措辞，当前必须「清空对话重新发送」，体验割裂。支持原地编辑用户消息可大幅提升流畅度。

## What Changes

- 用户消息气泡的悬浮操作栏增加「编辑」按钮（仅 `role === 'user'`）
- 点击编辑按钮后消息文本原地变为 `<textarea>` 编辑模式
- 按 Enter 或点击确认按钮保存编辑内容
- 按 Escape 或点击取消按钮恢复原文
- 编辑确认后**自动清除该消息之后的所有消息**（user + assistant），更新 `chatHistory` 和数据库
- 编辑完成后**自动触发流式回复**（相当于用户重新发送了编辑后的消息）
- 后端新增 `UpdateAIMessageContent(id, content)` 方法用于更新单条消息内容
- 后端新增 `DeleteAIMessagesAfter(sessionID, messageID)` 方法用于删除后续消息

## Impact

- Affected code: [ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js)、[ai-chat.css](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css)、[app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go)、[ai_service.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go)
- No breaking changes to existing data or features

## ADDED Requirements

### Requirement: Backend — UpdateAIMessageContent

The system SHALL provide a method to update a single AI message's content by its ID.

#### Scenario: Success case
- **WHEN** the frontend calls `App.UpdateAIMessageContent(id, newContent)`
- **THEN** the database record with that ID has its `content` field updated
- **AND** the method returns `nil` error

### Requirement: Backend — DeleteAIMessagesAfter

The system SHALL provide a method to delete all AI messages in a session that were created after a given message ID.

#### Scenario: Success case
- **WHEN** the frontend calls `App.DeleteAIMessagesAfter(sessionID, messageID)`
- **THEN** all messages in that session with `created_at > message.created_at` are deleted
- **AND** the method returns the count of deleted messages

### Requirement: Frontend — Edit button on user messages

The system SHALL show an "edit" button on user message bubbles' action bar.

#### Scenario: Normal display
- **GIVEN** a user message bubble (`role === 'user'`)
- **WHEN** the message is rendered in the chat
- **THEN** the `.action-buttons` area includes an edit button (pencil icon)
- **AND** the edit button follows the same hover-to-show pattern as other action buttons

### Requirement: Frontend — Inline editing mode

The system SHALL support inline editing of user message content.

#### Scenario: Enter edit mode
- **WHEN** the user clicks the edit button on a user message
- **THEN** the message content `<div>` is replaced with a `<textarea>` pre-filled with the current content
- **AND** a confirm button (checkmark) and a cancel button (X) are displayed
- **AND** the textarea auto-focuses and selects all text
- **AND** the action buttons (copy, edit) are hidden during edit mode

#### Scenario: Confirm edit with Enter
- **WHEN** the user presses `Ctrl+Enter` or clicks the confirm button while in edit mode
- **THEN** if the new content is different from the original:
  - The message bubble's content is updated to the new text
  - `_contextMsgContent` is also updated for right-click menu consistency
  - `chatHistory` is truncated to remove this message and all subsequent messages
  - All DOM message bubbles after this one are removed from `messagesEl`
  - Backend `UpdateAIMessageContent` is called to persist the change
  - Backend `DeleteAIMessagesAfter` is called to remove subsequent messages in the session
  - A new stream is automatically started (`startStreaming(false)`) with the updated chatHistory
- **WHEN** the new content is identical to the original
- **THEN** the edit mode is simply exited without any changes

#### Scenario: Cancel edit with Escape
- **WHEN** the user presses `Escape` or clicks the cancel button
- **THEN** edit mode exits, the original content is restored

#### Scenario: Edit while streaming
- **WHEN** the user clicks the edit button while `isStreaming === true`
- **THEN** nothing happens (edit button is disabled or click is ignored)

### Requirement: Frontend — Edge cases

#### Scenario: Empty content after edit
- **WHEN** the user confirms with empty or whitespace-only content
- **THEN** the edit is NOT saved, the textarea border briefly flashes red (validation feedback)
- **AND** edit mode remains active

#### Scenario: Very long content
- **WHEN** editing very long messages (>5000 chars)
- **THEN** the textarea scrolls naturally, no content truncation occurs

#### Scenario: Session not yet saved (activeSessionId === null)
- **WHEN** the user edits a message in an unsaved session
- **THEN** this case does not occur — user messages are only created after `createSession()` in `onSend()`
