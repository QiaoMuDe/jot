# Fix AI Stream Error State — 用户消息 ID 提前获取

## Why

AI 流式回复出错后，用户消息无法右键弹出菜单、点击重新发送/删除均无反应，必须切换会话才能恢复。原因是：

1. 用户消息的 `data-msgId` 依赖 `ai:stream-done` 事件赋值，出错时永不触发，导致 `handleResend/handleDeleteMsg` 因 `msgId` 为 `NaN` 而静默返回
2. `addErrorMessage()` 创建的 `.ai-msg-error` 没有绑定右键菜单，右键无反应

## What Changes

### 后端

- 新增 `SaveAIMessage(content, sessionId, role)` Go 方法：向指定会话插入一条消息，返回 `msgId`
- `CallAIStream()` 内部不再独立保存用户消息，而是由前端先调 `SaveAIMessage` 拿到 ID 后传入，`CallAIStream` 直接使用传入的 `userMsgID`

### 前端

- **`onSend()`**：先调 `SaveAIMessage` 拿到 `userMsgId`，然后 `addMessage(text, 'user', ..., userMsgId)` 直接带上 ID，同时传给 `startStreaming`
- **`startStreaming()` 参数**：接收 `userMsgID`，传给 `CallAIStream`
- **错误处理**：`streamingEl.remove()` 保持不变，错误仅通过 `window.showNotification` 通知，不再调用 `addErrorMessage()` 创建无菜单的错误气泡

## Impact

- Affected code:
  - `frontend/src/js/ai-chat.js` — `onSend`, `startStreaming` 参数, `ai:stream-error` 处理器, `handleRegenerate`
  - `app.go` — `SaveAIMessage` 新增, `CallAIStream` 参数修改
  - `internal/services/ai_service.go` — 抽取独立的消息保存逻辑
- 无数据库 schema 变更
- 无 CSS 变更

## MODIFIED Requirements

### Requirement: 用户发送消息

#### Scenario: 正常发送
- **WHEN** 用户输入文本并点击发送
- **THEN** 前端先调用 `SaveAIMessage(content, sessionId, 'user')` 保存用户消息到 DB
- **AND** 拿到返回的 `msgId`
- **AND** `addMessage(text, 'user', ..., msgId)` 创建消息气泡时带上 `data-msgId`
- **AND** `startStreaming(text, userMsgId)` 将 ID 传入流式流程
- **AND** `CallAIStream(myGen, ..., userMsgId)` 后端不再新建用户消息

#### Scenario: 流式出错
- **WHEN** AI 流式回复过程中发生错误
- **THEN** `streamingEl.remove()` 正常执行移除 AI 气泡
- **AND** 仅通过 `window.showNotification(err, 'error')` 通知用户，不再调用 `addErrorMessage()`
- **AND** `isStreaming = false`，按钮状态恢复
- **AND** 用户消息的 `data-msgId` 已存在，`handleResend/handleDeleteMsg` 正常工作

#### Scenario: 出错后重新发送
- **WHEN** 用户在用户消息上右键 → 选择"重新发送"
- **THEN** `handleResend(msgEl)` 被调用
- **AND** `msgEl.dataset.msgId` 已存在（`SaveAIMessage` 提前保存）
- **AND** 流程正常执行：DOM 清理 → 后端截断 → 重建用户消息 → `startStreaming`

## ADDED Requirements

### Requirement: SaveAIMessage 后端方法

#### Scenario: 成功保存
- **WHEN** 前端调用 `SaveAIMessage(content, sessionId, role)`
- **THEN** 后端将消息插入 `ai_messages` 表
- **AND** 返回新消息的 `id`（int64）

### Requirement: CallAIStream 参数变更

- **WHEN** 前端调用 `CallAIStream(..., userMsgID)`
- **THEN** 后端跳过保存用户消息步骤，直接使用传入的 `userMsgID` 作为上一条用户消息
- **AND** 后续流程（搜索、AI 调用、保存 AI 回复）不变

## REMOVED Requirements

### Requirement: CallAIStream 内部保存用户消息
**Reason**: 改为由前端先保存用户消息并拿到 ID，确保 DOM 和 DB 的消息 ID 一致
**Migration**: 前端在所有入口（新消息、重新发送）都先调 `SaveAIMessage`

### Requirement: addErrorMessage 错误气泡
**Reason**: 无右键菜单的纯文本错误气泡对用户无实际作用，改用通知替代
**Migration**: `ai:stream-error` 处理器中移除 `addErrorMessage()` 调用
