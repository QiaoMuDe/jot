# 退出时清除新建笔记草稿

## Why

用户新建笔记（`state.editingNoteId === null`）时，内容通过 `SaveDraft()` 存入临时草稿表。退出程序时，无论选择「保存并退出」还是「不保存」，草稿都应被清除，避免下次启动时残留旧草稿导致弹窗询问"是否恢复"。

## What Changes

- 在 `handleAppExit()` 的「保存并退出」和「不保存」分支中，当为新建笔记（`!state.editingNoteId`）时，调用 `ClearDraft()` 清除草稿
- 不涉及 `cancel` 分支（用户取消退出，不清除）

## Impact

- Affected code: `frontend/src/main.js` — `handleAppExit()` 函数

## ADDED Requirements

### Requirement: 退出时清除新建笔记草稿

系统 SHALL 在退出程序时，对于新建笔记（`state.editingNoteId === null`），在保存或丢弃后调用 `ClearDraft()` 清除临时草稿。

#### Scenario: 保存并退出（新建笔记）

- **WHEN** 用户打开新建笔记，输入内容，按 `Ctrl+Q`
- **AND** 在确认对话框选择「保存并退出」
- **THEN** 调用 `SaveDraft()` 保存内容，再调用 `ClearDraft()` 清除草稿，然后 `Quit()`

#### Scenario: 不保存（新建笔记）

- **WHEN** 用户打开新建笔记，输入内容，按 `Ctrl+Q`
- **AND** 在确认对话框选择「不保存」
- **THEN** 调用 `ClearDraft()` 清除草稿，然后 `Quit()`

#### Scenario: 编辑已有笔记退出

- **WHEN** 用户打开已有笔记（`state.editingNoteId` 不为 null），按 `Ctrl+Q`
- **THEN** 不调用 `ClearDraft()`，正常保存或退出

#### Scenario: 取消退出

- **WHEN** 用户在确认对话框选择「取消」
- **THEN** 不清除草稿，不退出
