# AI 消息删除功能 Spec

## Why
用户无法删除 AI 对话中的单条消息（用户消息或 AI 回复），只能清空整个会话或编辑。需要支持在消息下方操作栏和右键菜单中删除单条消息。

## What Changes
- 后端新增 `DeleteAIMessage(id uint) error` 方法，按 ID 删除单条消息
- 前端消息操作栏（`createMsgActions`）在 user 和 assistant 消息均添加删除按钮
- 前端右键菜单（`showAiMsgContextMenu`）在 user 和 assistant 消息均添加删除项
- 删除后自动移除 DOM 元素并清理 `chatHistory`

## Impact
- Affected specs: AI 聊天功能
- Affected code:
  - `internal/services/ai_service.go` — 新增 `DeleteAIMessage`
  - `app.go` — Wails 绑定新方法
  - `frontend/src/js/ai-chat.js` — 操作栏按钮 + 右键菜单 + 点击处理器 + 删除逻辑
  - `frontend/src/css/components/ai-chat.css` — 删除按钮样式（可选）

## ADDED Requirements
### Requirement: 单条消息删除
The system SHALL allow users to delete individual AI messages (user or assistant) from a conversation.

#### Scenario: 通过消息操作栏删除
- **WHEN** 用户将鼠标悬停在任意消息上
- **THEN** 操作栏显示删除按钮（🗑 图标）
- **WHEN** 用户点击删除按钮
- **THEN** 弹出确认对话框「确定要删除这条消息吗？」
- **WHEN** 用户确认后
- **THEN** 该消息从 DOM 中移除，从 `chatHistory` 中移除，从数据库中删除
- **AND** 如果删除的是用户消息之后的所有消息（保持上下文连续），后续消息也被删除
- **AND** 如果当前在流式生成中，禁用删除按钮

#### Scenario: 通过右键菜单删除
- **WHEN** 用户在任意消息上右键
- **THEN** 右键菜单包含「删除」选项
- **WHEN** 用户点击删除
- **THEN** 弹出确认对话框 → 确认后执行删除（同上）

#### Scenario: 删除后续消息影响
- **WHEN** 用户删除一条非末尾的消息
- **THEN** 该消息之后的所有消息（AI 回复、后续对话）一并删除
- **AND 对话上下文从删除点截断，DB 同步更新
