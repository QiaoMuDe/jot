# 修复会话消息显示问题 + 折叠侧栏 Spec

## Why
当前多会话有两类问题：1）切换到别的会话再切换回来，消息角色错乱（AI 消息显示在用户侧，用户消息显示在 AI 侧），且思维链内容丢失。2）会话侧栏固定占据 220px 空间，无法折叠，默认应该收起。

## What Changes
- **FIX**: `Message` 结构体新增 `ReasoningContent` 字段，`LoadAISessionMessages` 返回时包含思维链内容
- **FIX**: `addMessage` 函数新增 `reasoningContent` 参数，加载历史消息时重建可折叠思考区域
- **FIX**: 重写 `switchSession` 确保角色映射正确，消除消息错乱
- **FIX**: 加载历史消息时关闭流式事件监听残留
- **NEW**: 会话侧栏折叠/展开功能，默认折叠（collapsed = true）
- **NEW**: 折叠状态使用 localStorage 持久化

## Impact
- Affected specs: `persist-ai-sessions`（已完成的会话持久化 spec，当前修复其遗留 bug）
- Affected code:
  - `internal/services/ai_service.go`：`Message` 结构体追加 `ReasoningContent`
  - `frontend/src/js/ai-chat.js`：修复 `switchSession`/`addMessage`/`onAIChatViewActivated`
  - `frontend/src/css/components/ai-chat.css`：侧栏折叠/展开动画

## ADDED Requirements

### Requirement: Message 结构体支持 ReasoningContent
系统 SHALL 在 `services.Message` 结构体中新增 `ReasoningContent` 字段。

- **WHEN** `LoadAISessionMessages` 从数据库加载消息
- **THEN** 返回的消息包含 `ReasoningContent` 字段
- **AND** 前端加载后渲染可折叠思考区域

### Requirement: 侧栏折叠/展开
系统 SHALL 在会话侧栏右边框上提供折叠/展开切换按钮。

#### Scenario: 默认折叠
- **WHEN** 用户进入 AI 助手视图
- **THEN** 会话侧栏默认收起（width: 0），只露出折叠按钮
- **AND** 折叠按钮图标为 `▶`

#### Scenario: 展开侧栏
- **WHEN** 用户点击折叠按钮或按钮图标为 `▶`
- **THEN** 侧栏以 CSS transition 展开到 220px
- **AND** 按钮图标变为 `◀`

#### Scenario: 折叠侧栏
- **WHEN** 用户点击折叠按钮或按钮图标为 `◀`
- **THEN** 侧栏以 CSS transition 收起
- **AND** 状态保存到 localStorage

### Requirement: 消息角色正确显示
系统 SHALL 确保加载历史消息时，每条消息的角色（user/assistant）正确映射到对应的气泡样式。

- **FIX**: `switchSession` 在加载消息前先清理所有监听器和状态
- **FIX**: `addMessage` 对加载的消息正确渲染思考区域

## MODIFIED Requirements

### Requirement: switchSession 修复（现有）
- **MODIFIED**: 加载消息前增加 `if (isStreaming) return` 保护
- **MODIFIED**: 先清空 `chatHistory`，再加载新历史
- **MODIFIED**: 对每条 assistant 消息，如果有 `reasoningContent` 则渲染思考区域

### Requirement: onAIChatViewActivated 修复（现有）
- **FIX**: 当视图激活时，如果已有激活的 session 且有消息，不重新加载，避免覆盖消息。只有在 `activeSessionId === null` 时才自动加载第一个会话。

## REMOVED Requirements
无
