# 表格复制按钮 Spec

## Why

Markdown 预览模式下的代码块已有悬浮复制按钮，但表格（table）没有。用户需要在鼠标悬停表格时右上角出现复制按钮，点击后复制整个表格内容到剪贴板，方便粘贴到其他应用中。

## What Changes

- 在 `updatePreview()` 中为每个 `.md-rendered table` 添加 `.copy-table-btn` 按钮（避免重复添加）
- 点击按钮将表格转换为 Markdown 格式文本写入剪贴板（使用 `navigator.clipboard.writeText`）
- 复制成功/失败状态反馈同代码块复制按钮
- 新增 `.copy-table-btn` CSS 样式（同 `.copy-code-btn` 设计，悬停显示、垂直居中、置于右侧）

## Impact

- Affected code: `frontend/src/main.js`（`updatePreview()` 函数）、`frontend/src/style.css`（新增 `.copy-table-btn` 样式）
- 不涉及后端、数据库、模型

## ADDED Requirements

### Requirement: Table Copy Button

The system SHALL provide a copy button for tables in Markdown preview.

#### Scenario: Copy table on hover

- **WHEN** user hovers over a table in Markdown preview mode
- **THEN** a "复制" button appears at the top-right of the table

#### Scenario: Click copy button

- **WHEN** user clicks the copy button on a table
- **THEN** the entire table content is copied to clipboard as Markdown format text
- **AND** the button shows "✓ 已复制" for 1.5 seconds
- **AND** if copy fails, shows "✗ 复制失败" for 1 second

#### Scenario: Avoid duplicate buttons

- **WHEN** `updatePreview()` is called multiple times (e.g., on input)
- **THEN** already existing copy buttons are not duplicated

### MODIFIED Requirements

无

### REMOVED Requirements

无
