# Simplify AI Message Actions Spec

## Why
当前消息底部操作栏右侧有一排按钮（复制、编辑、重新发送、保存为笔记、重新生成、追问、删除），悬停才显示，占用横向空间且视觉杂乱。用户希望将其简化为一个三点按钮（⋮），点击弹出与右键菜单一致的操作菜单，且操作栏不再隐藏，而是与 token 消耗信息一起常驻显示。

## What Changes
- 将 `.action-buttons` 中所有独立操作按钮替换为一个横排三点按钮（⋯）
- 横排三点按钮点击弹出垂直菜单（与右键菜单相同的菜单项 + 相同的样式）
- 操作栏（`.ai-msg-actions`）始终 visible（移除 opacity 渐显逻辑），与 token 信息在同一行常驻
- 删除 `collapseActionsIfNeeded` 及其相关折叠逻辑
- 删除 `toggleActionPopup` 函数（旧的水平弹出逻辑）
- 删除 `.action-popup` 相关 CSS（旧的弹出菜单样式）
- 更新 `user-tokens` 定位/SVG 大小等适配样式
- 移除 `.action-buttons` 的 `opacity`/`transition` 悬停渐显

## Impact
- Affected specs: AI Chat 消息渲染
- Affected code:
  - `frontend/src/js/ai-chat.js` — `createMsgActions`, `collapseActionsIfNeeded`, `toggleActionPopup`, `handleCopy`, `handleDeleteMsg` 等
  - `frontend/src/css/components/ai-chat.css` — `.ai-msg-actions`, `.action-buttons`, `.action-popup`, `.more-btn`, `.user-tokens` 等
  - `frontend/index.html` — 可能不需要修改

## ADDED Requirements
### Requirement: 三点按钮菜单
The system SHALL provide a single three-dot (⋮) button in the message action bar.

#### Scenario: 点击三点按钮弹出菜单
- **WHEN** user clicks the ⋯ button on a message
- **THEN** a vertical popup menu appears (same menu items as right-click context menu)
- **AND** the menu SHALL be positioned near the button
- **AND** clicking outside or on a menu item SHALL close the menu

#### Scenario: 菜单项执行操作
- **WHEN** user clicks a menu item in the popup
- **THEN** the corresponding action SHALL execute (copy, edit, resend, save, regenerate, follow-up, delete)
- **AND** the menu SHALL close after selection

### Requirement: 操作栏常驻显示
The action bar SHALL always be visible alongside token info.

#### Scenario: 常驻显示
- **WHEN** any message is displayed
- **THEN** the `.ai-msg-actions` bar SHALL be visible without hover
- **AND** the three-dot button SHALL be visible at all times

## MODIFIED Requirements
### Requirement: 删除旧的操作按钮布局
**Change**: Remove individual action buttons (copy, edit, resend, save, regenerate, follow-up, delete) from the action bar. Replace with a single ⋮ button.

### Requirement: 删除折叠逻辑
**Change**: Remove `collapseActionsIfNeeded` function and all `.collapsed`/`.narrow-mode`/`.wide-mode` related logic for user message buttons.

## REMOVED Requirements
### Requirement: 旧的 action-popup 水平弹出菜单
**Reason**: Replaced by vertical context-menu style popup.
**Migration**: Use the new ⋮ button popup which mirrors the right-click context menu.

### Requirement: 操作按钮悬停渐显
**Reason**: Action bar should always be visible.
**Migration**: `.action-buttons` opacity set to 1 at all times.
