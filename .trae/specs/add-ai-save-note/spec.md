# AI 消息保存为笔记 Spec

## Why

AI 助手中的 AI 回复内容（代码示例、写作片段、知识总结等）目前只能停留在聊天窗口，无法持久化到笔记系统中。需要一个快捷方式将 AI 回复保存为笔记，方便后续查看、编辑和整理。

## What Changes

- **新增**：`SaveAIMessageAsNote` 后端绑定，将 AI 回复内容创建为一条笔记（排除思维链 content）
- **新增**：仅 AI 回复气泡的操作按钮区增加"保存为笔记"按钮
- **修改**：`createMsgActions` 函数，为 assistant 角色增加保存按钮

## Impact

- Affected specs: `add-ai-assistant`, `add-card-note-app`
- Affected code: `app.go`, `frontend/src/js/ai-chat.js`

---

## ADDED Requirements

### Requirement: 后端保存绑定

The system SHALL provide a backend function to save AI message content as a note.

#### Scenario: 保存成功
- **WHEN** frontend calls `SaveAIMessageAsNote(content, notebookID)`
- **THEN** the system auto-generates a title: trim whitespace, take first line if non-empty else first 50 chars of first paragraph, truncate to 50 chars with "..." suffix if needed
- **AND** creates a new Note with `.md` file extension
- **AND** returns the created `models.Note` object

#### Scenario: 保存失败
- **WHEN** content is empty
- **THEN** returns an error "内容不能为空"

### Requirement: 前端保存按钮

The system SHALL add a "保存为笔记" button ONLY to AI assistant message action buttons.

#### Scenario: AI 回复可见
- **WHEN** an AI assistant message bubble is rendered
- **THEN** the action buttons include: 复制、保存为笔记、重新生成（按此顺序）

#### Scenario: 点击保存
- **WHEN** user clicks "保存为笔记" on an AI message
- **THEN** the system calls `SaveAIMessageAsNote(content, 0)` with only the assistant's response content (不含 reasoning_content）
- **AND** shows a success notification "笔记已创建"
- **AND** 通知中附带"查看"按钮，点击后切换到 grid 视图并打开该笔记

---

## MODIFIED Requirements

### Requirement: `createMsgActions` 函数（已存在）

在 `createMsgActions` 中，为 `role === 'assistant'` 的分支新增"保存为笔记"按钮，位于复制和重新生成之间。

---

## REMOVED Requirements

None.
