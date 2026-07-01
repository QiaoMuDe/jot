# AI 消息右键上下文菜单 Spec

## Why

目前 AI 对话中的操作按钮（复制、保存为笔记、重新生成、追问）仅在鼠标悬浮在消息气泡上时显示。用户期望能通过鼠标右键点击消息直接弹出菜单，包含这些操作项，提升操作便捷性和发现性。

## What Changes

- 在 `index.html` 中新增一个 `#aiMsgContextMenu` 右键菜单元素，复用现有的 `.context-menu` 样式
- 在 `ai-chat.js` 中为用户消息和 AI 回复消息添加 `contextmenu` 事件监听
- 右击消息时弹出菜单，菜单位置跟随鼠标坐标
- 菜单项根据消息角色动态生成：
  - 用户消息：复制
  - AI 回复：复制、保存为笔记、重新生成、追问
- 实现右键菜单的显示/隐藏逻辑（点击外部关闭、点击菜单项关闭）
- 右键菜单样式使用现有 `.context-menu` 样式体系，无需额外 CSS 变量

## Impact

- Affected specs: AI 助手相关
- Affected code:
  - `frontend/index.html` — 新增 `#aiMsgContextMenu` DOM 元素
  - `frontend/src/js/ai-chat.js` — 新增右键菜单创建/显示/隐藏逻辑，为消息气泡绑定 `contextmenu` 事件
  - `frontend/src/css/components/ai-chat.css` — 可选微调（如有必要）

## ADDED Requirements

### Requirement: 右键菜单功能

The system SHALL 支持在 AI 消息气泡上右键弹出上下文菜单。

#### Scenario: 右键用户消息
- **WHEN** 用户在 AI 对话页面对用户消息气泡点击鼠标右键
- **THEN** 弹出上下文菜单，包含「复制」菜单项
- **AND** 点击「复制」执行复制消息内容的操作

#### Scenario: 右键 AI 回复消息
- **WHEN** 用户在 AI 对话页面对 AI 回复消息气泡点击鼠标右键
- **THEN** 弹出上下文菜单，包含「复制」「保存为笔记」「重新生成」「追问此条回复」菜单项
- **AND** 每个菜单项功能与现有的悬浮操作按钮一致

#### Scenario: 菜单关闭
- **WHEN** 右键菜单已打开
- **AND** 用户在菜单外部点击、或按 Escape 键、或点击某个菜单项
- **THEN** 右键菜单关闭

#### Scenario: 菜单定位
- **WHEN** 右键菜单弹出
- **THEN** 菜单位置跟随鼠标点击位置
- **AND** 菜单在靠近屏幕边缘时自动调整方向，避免溢出视口
