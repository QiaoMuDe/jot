# AI 会话标题动态显示 + 笔记卡片右键菜单溢出修复 Spec

## Why

### 标题动态化
当前 AI 助手视图顶部始终显示固定的"AI 助手"标题。当用户进入有历史消息的会话时，标题未跟随会话内容变化，缺少上下文提示。

### 菜单溢出
笔记首页底部的卡片右键点击时，上下文菜单可能超出视口底部边界，部分菜单项被裁剪不可见，影响操作可达性。

## What Changes

### 1. AI 会话标题动态显示
- 无会话 / 欢迎语状态 → 显示 "AI 助手"
- 有活跃会话且有消息 → 显示该会话的标题（与侧栏会话项标题文字一致）
- 新建空会话（欢迎语状态） → 显示 "AI 助手"

### 2. 笔记卡片右键菜单视口溢出修复
- `showContextMenu()` 在设置菜单位置时，检测菜单是否超出视口底部
- 若超出，将菜单的 `top` 上移，使菜单完整可见
- 同时考虑菜单高度动态变化（由菜单项数量和分割线决定）

## Impact
- 仅修改前端代码，无后端变更
- 涉及 2 个模块：AI 对话页面标题、笔记首页右键菜单

## File Changes

### ai-chat.js
- `onAIChatViewActivated()` 后或 `switchSession()` 中更新 `#aiChatTitle` 文本
- `showWelcome()` 中重置标题为 "AI 助手"
- 在 `createSession()`、`saveSessionMessages()` 等会话标题可能变化的时机同步标题

### main.js
- `showContextMenu()` 中添加视口边界检测逻辑，防止菜单溢出底部

## ADDED Requirements

### Requirement: AI Chat Title Dynamic Display
The system SHALL update the AI chat view header title based on the current session state.

#### Scenario: No active session (welcome/empty)
- **WHEN** there is no active session or the welcome state is shown
- **THEN** the header title SHALL display "AI 助手"

#### Scenario: Active session with messages
- **WHEN** a session with messages is loaded (via `switchSession`)
- **THEN** the header title SHALL display the session's title (same as sidebar)

#### Scenario: New empty session created
- **WHEN** `createSession()` is called and the welcome screen is shown
- **THEN** the header title SHALL display "AI 助手"

#### Scenario: Session title renamed
- **WHEN** a session's title is renamed (inline edit)
- **THEN** the header title SHALL update to match the new title

### Requirement: Right-click Menu Viewport Overflow Prevention
The system SHALL prevent the note card context menu from overflowing the viewport bottom.

#### Scenario: Click near bottom edge
- **WHEN** the user right-clicks a card near the bottom of the viewport
- **THEN** the context menu SHALL be positioned above the cursor if it would otherwise extend beyond the viewport boundary

## MODIFIED Requirements
None.

## REMOVED Requirements
None.
