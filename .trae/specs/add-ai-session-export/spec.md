# AI 对话导出 Spec

## Why

用户无法将 AI 对话内容持久化到外部文件，也无法方便地重命名会话（当前仅通过双击触发内联编辑，路径不够直观）。右键菜单提供统一入口，提升操作效率。

## What Changes

- **新增**：会话侧栏右键菜单（重命名 + 导出），右击会话列表项时弹出
- **新增**：`ExportAISessionAsMarkdown` 后端绑定，支持将对话导出为 `.md` 文件
- **修改**：将会话内联编辑逻辑提取为独立函数，供双击和右键菜单复用
- **修改**：右侧内容区点击时自动关闭右键菜单

## Impact

- Affected specs: `add-ai-assistant`, `persist-ai-sessions`
- Affected code: `frontend/index.html`, `frontend/src/js/ai-chat.js`, `frontend/src/css/components/ai-chat.css`, `app.go`

---

## ADDED Requirements

### Requirement: 会话右键菜单

The system SHALL provide a context menu when the user right-clicks an AI session item in the sidebar.

#### Scenario: 右键弹出菜单
- **WHEN** user right-clicks an AI session list item
- **THEN** a context menu appears near the cursor position
- **AND** the menu contains "重命名" and "导出" options

#### Scenario: 点击菜单项关闭菜单
- **WHEN** user clicks a menu item or clicks anywhere outside the menu
- **THEN** the context menu closes

### Requirement: 重命名功能

The system SHALL allow renaming an AI session via the context menu.

#### Scenario: 右键重命名
- **WHEN** user clicks "重命名" in the context menu
- **THEN** the session title enters inline edit mode (same as double-click behavior)

### Requirement: 导出对话

The system SHALL export the current AI conversation as a Markdown file.

#### Scenario: 导出成功
- **WHEN** user clicks "导出" in the context menu
- **THEN** a save file dialog opens with `.md` file filter and default filename `{会话标题}.md`
- **AND** after saving, a success notification is shown

#### Scenario: 用户取消导出
- **WHEN** user cancels the save file dialog
- **THEN** no action is taken

### Requirement: 导出格式

The exported `.md` file SHALL contain the full conversation with proper formatting.

#### Scenario: 导出格式正确
- **WHEN** the conversation is exported
- **THEN** the file format SHALL be:
  ```markdown
  # {会话标题}

  ---

  **User**:
  message content

  ---

  **AI Assistant**:
  response content

  ---

  **User**:
  message content

  ---
  ```
- **AND** messages with `reasoning_content` SHALL include a "思考过程" section:
  ```markdown
  **AI Assistant**:
  > 思考过程：
  > reasoning content

  response content
  ```

---

## MODIFIED Requirements

### Requirement: 会话列表渲染（已存在）

更新 `renderSessionList` 为每个会话项添加 `contextmenu` 事件监听。

---

## REMOVED Requirements

None.
