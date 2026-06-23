# 从编辑模式返回查看模式按钮 Spec

## Why

查看笔记时（只读模式），右上角有一个铅笔按钮可以进入编辑模式。但进入编辑模式后，没有对应的"切回查看模式"按钮，用户只能保存或关闭编辑器重新点开。这在只想快速浏览内容时不方便。

## What Changes

- 新增一个"返回查看模式"按钮（眼睛图标），位于编辑器右上角
- 仅当用户**从查看模式点击编辑按钮进入编辑模式**时才显示
- 从网格视图直接编辑（点击笔记卡片编辑按钮）或新建笔记时不显示
- 点击该按钮原地切换为只读查看模式，不走 closeEditor（避免动画闪烁）

## Impact

- Affected specs: 编辑器查看/编辑模式切换
- Affected code: `frontend/index.html`（新增按钮）、`frontend/src/main.js`（显示/隐藏逻辑 + 点击事件）、`frontend/src/style.css`（按钮样式）

## ADDED Requirements

### Requirement: 编辑器中"返回查看模式"按钮

The system SHALL provide a "返回查看模式" button in the editor header that allows users to switch from edit mode back to read-only view mode.

#### Scenario: 从查看模式进入编辑后返回
- **WHEN** 用户在查看模式下点击铅笔编辑按钮
- **THEN** 编辑模式下右上角显示"返回查看模式"按钮（眼睛图标）
- **AND WHEN** 用户点击该按钮
- **AND** 笔记 id 存在（`state.editingNoteId`）
- **THEN** 调用 `openEditor(noteId, true)` 原地切换回只读查看模式

#### Scenario: 正常编辑/新建时不显示
- **WHEN** 用户从网格视图点击笔记卡片编辑按钮打开编辑模式
- **OR WHEN** 用户新建笔记
- **THEN** "返回查看模式"按钮不显示

#### Scenario: 关闭编辑器后状态重置
- **WHEN** 用户关闭编辑器（任何方式）
- **THEN** "返回查看模式"按钮的内部状态被重置，下次打开不意外显示
