# AI 助手消息表格复制按钮 Spec

## Why

AI 助手回复中的 Markdown 表格目前没有复制按钮，用户需要手动选中复制，体验不佳。代码块已有悬浮复制按钮（`.code-copy-btn`），表格也应具备类似功能。

## What Changes

- 在 `renderMarkdown()`（`ai-chat.js`）中为每个 AI 消息内的 `<table>` 添加复制按钮
- 复制按钮放置在第一行（表头行）的最后一列（`th:last-child`）单元格内，居右对齐、垂直居中
- 点击复制按钮将整个表格转换为 Markdown 格式文本写入剪贴板
- 复制成功/失败状态反馈同代码块复制按钮
- 新增 `.ai-msg-assistant .table-copy-btn` CSS 样式，参考 `.code-copy-btn` 的设计

## Impact

- Affected code: `frontend/src/js/ai-chat.js`（`renderMarkdown()` 函数）、`frontend/src/css/components/ai-chat.css`（新增 `.table-copy-btn` 样式）
- 不涉及后端、数据库、模型

## ADDED Requirements

### Requirement: AI Table Copy Button

The system SHALL provide a copy button for tables in AI assistant reply messages.

#### Scenario: Copy table on hover

- **WHEN** user hovers over a table in AI assistant message
- **THEN** a "复制" button appears in the last column header (`th:last-child`) of the first row, right-aligned and vertically centered

#### Scenario: Click copy button

- **WHEN** user clicks the copy button on a table
- **THEN** the entire table content is copied to clipboard as Markdown format text
- **AND** the button shows checkmark + "已复制" for 1.5 seconds
- **AND** if copy fails, shows "复制失败" for 1 second

#### Scenario: Avoid duplicate buttons

- **WHEN** `renderMarkdown()` is called multiple times (e.g., on re-render)
- **THEN** already existing copy buttons are not duplicated

### MODIFIED Requirements

无

### REMOVED Requirements

无
