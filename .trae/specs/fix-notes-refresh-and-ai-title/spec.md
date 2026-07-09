# 笔记首页刷新 + AI 会话标题修复 Spec

## Why

### 笔记首页刷新问题
Ctrl+1 快捷键切换到笔记首页时没有调用 `loadNotes()`，导致页面展示的是之前缓存的笔记列表，而非从数据库获取的最新数据。用户在别处修改笔记后返回首页，看到的可能是过时内容。

### AI 会话标题问题
后端 `SaveAIMessages` 已有首条消息自动生成标题的逻辑（取第一条 user 消息前 30 字），但前端流式完成回调中未主动刷新会话列表和标题显示。因此用户发送第一条消息后，UI 仍显示 "新对话"（`updateChatTitle()` 依赖的 `sessions` 数组和 `chatHistory.length` 未同步），导致标题实际已写入数据库但 UI 未更新。切换会话时触发 `loadSessionList()` 后才看到正确标题。

## What Changes

### 1. 笔记首页刷新修复
将缺失 `loadNotes()` 调用的导航路径补全，确保每次回到笔记首页都从数据库加载最新列表。

### 2. AI 会话标题修复
在流式完成（`ai:stream-done`）回调末尾，增加 `loadSessionList()` 和 `updateChatTitle()` 调用，使后端自动生成的标题立即反映到 UI。

## Impact

- Affected specs: 笔记列表导航、AI 对话
- Affected code: `frontend/src/main.js`、`frontend/src/js/ai-chat.js`
- 无后端变更
- 两项修改相互独立，可并行实现

## ADDED Requirements

### Requirement: 笔记首页始终加载最新列表
The system SHALL always load the latest notes list from the database when navigating to the notes homepage.

#### Scenario: 使用 Ctrl+1 快捷键返回首页
- **WHEN** 用户按下 Ctrl+1 快捷键切换回笔记首页
- **THEN** 系统调用 `loadNotes()` 从数据库加载最新的笔记列表

### Requirement: AI 会话标题首条消息立即更新
The system SHALL update the session title in the UI immediately after the first message response completes.

#### Scenario: 首条消息发送完成
- **WHEN** 用户发送第一条消息且 AI 流式响应完成
- **THEN** 侧栏会话列表显示后端自动生成的标题（取首条 user 消息前 30 字）
- **AND** 顶部标题（`#aiChatTitle`）同步更新为该会话的标题

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
