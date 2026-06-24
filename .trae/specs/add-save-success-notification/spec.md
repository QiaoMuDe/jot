# 添加保存成功通知

## Why

用户创建或编辑笔记后，无论通过保存按钮、Ctrl+S 快捷键还是退出前保存，都没有任何成功提示。编辑器关闭后页面直接回到笔记列表，用户不确定操作是否成功。其他操作（如删除、备份）已有 Toast 通知，保存操作缺少反馈不一致。

## What Changes

- 在 `createNote()` 保存成功后添加「笔记已创建」通知
- 在 `updateNote()` 保存成功后添加「笔记已更新」通知
- 退出前保存（`handleAppExit()` + `saveEditorContent()`）不添加通知（应用即将退出，通知无意义）
- 仅后端 API 调用成功时显示通知，失败时不显示

## Impact

- Affected code: `frontend/src/main.js` — `createNote()` 和 `updateNote()` 函数

## ADDED Requirements

### Requirement: 创建笔记成功通知

系统 SHALL 在笔记创建成功后显示成功通知。

#### Scenario: 新建笔记保存

- **WHEN** 用户填入标题和内容，点击保存按钮或按 `Ctrl+S`
- **THEN** 笔记创建成功，关闭编辑器，回到笔记列表，右上角显示 `nm.show('笔记已创建', 'success')` 通知

### Requirement: 编辑笔记成功通知

系统 SHALL 在笔记更新成功后显示成功通知。

#### Scenario: 编辑笔记保存

- **WHEN** 用户编辑已有笔记，点击保存按钮或按 `Ctrl+S`
- **THEN** 笔记更新成功，关闭编辑器，回到笔记列表，右上角显示 `nm.show('笔记已更新', 'success')` 通知

### Requirement: 退出前保存不显示通知

系统 SHALL NOT 在退出前保存时显示通知。

#### Scenario: Ctrl+Q 保存并退出

- **WHEN** 用户按 `Ctrl+Q`，在确认对话框选「保存并退出」
- **THEN** 调用 `saveEditorContent()` 保存内容，直接 `Quit()`，不显示通知
