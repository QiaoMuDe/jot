# 代码块语言标签与语法高亮切换 Spec

## Why

当前代码块已通过 highlight.js 自动检测并高亮语法，但用户无法知道检测结果是什么语言，也无法手动指定语言。当自动检测不准时（例如纯文本被误判为某语言），用户需要手动选择正确的语言重新高亮。

## What Changes

- 在 `updatePreview()` 中为每个 `<pre>` 代码块底部添加语言标签 `.code-lang-badge`
- 标签显示当前高亮语言名称（如 "JavaScript"、"Python"），由 hljs 自动检测结果决定
- 鼠标悬停代码块时标签显示（同复制按钮 hover 模式）
- 单击语言标签弹出语言选择菜单 `.code-lang-menu`，列出所有已注册语言
- 选择语言后调用 `hljs.highlightElement(block, { language })` 重新高亮并更新标签文字
- 不涉及后端、数据库、模型，仅前端 JS + CSS

## Impact

- Affected code: `frontend/src/main.js`（`updatePreview()` 函数增加语言标签 + 菜单逻辑）、`frontend/src/style.css`（新增 `.code-lang-badge`、`.code-lang-menu` 样式）
- 语言列表取自 hljs 已注册的 6 种语言：JavaScript、Python、CSS、HTML/XML、Bash/Shell、JSON

## ADDED Requirements

### Requirement: Code Language Badge

The system SHALL display a language badge at the bottom-right of code blocks in Markdown preview.

#### Scenario: Show detected language

- **WHEN** user hovers over a code block
- **THEN** a small language badge appears at the bottom-right corner showing the detected language name (e.g., "JavaScript", "Python", "plaintext" for auto-detected)
- **AND** the badge disappears when hover ends

#### Scenario: Language selection menu

- **WHEN** user clicks the language badge
- **THEN** a dropdown menu appears above the badge listing all available languages
- **AND** the currently active language is highlighted in the menu

#### Scenario: Re-highlight with selected language

- **WHEN** user selects a language from the menu
- **THEN** the code block is re-highlighted by hljs with the selected language
- **AND** the badge text updates to show the new language name
- **AND** the menu closes

### MODIFIED Requirements

无

### REMOVED Requirements

无
