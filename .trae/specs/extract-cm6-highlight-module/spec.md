# Extract CM6 Syntax Highlight Module Spec

## Why

CM6 Markdown 语法高亮和编辑器样式定义（`jotTheme` + `jotHighlightStyle`，约 85 行）目前内联在 `initCodeMirror()` 函数中，增加了函数复杂度，不利于独立维护和复用。将其提取为独立模块可使 `main.js` 职责更清晰，高亮样式可独立修改和测试。

## What Changes

- **新建** `frontend/src/js/` 目录 + `frontend/src/js/cm6-syntax-highlight.js` 模块文件
  - 将 `jotTheme`（`EditorView.theme(...)`）移至模块并导出
  - 将 `jotHighlightStyle`（`HighlightStyle.define(...)`）移至模块并导出
  - 导出预组合的 `mdHighlight` 扩展数组（`[markdown(), syntaxHighlighting(jotHighlightStyle)]`）
- **修改** `frontend/src/main.js`
  - 删除内联的 `jotTheme` 和 `jotHighlightStyle` 定义
  - 从 `./js/cm6-syntax-highlight.js` 导入 `{ jotTheme, mdHighlight }`
  - `initCodeMirror()` 中的扩展数组引用 `jotTheme` 和 `mdHighlight`

## Impact

- Affected specs: none
- Affected code:
  - `frontend/src/main.js` — 删除 ~85 行内联定义 + 添加 1 行 import
  - `frontend/src/js/cm6-syntax-highlight.js` — 新建文件，~85 行

## ADDED Requirements

### Requirement: CM6 语法高亮模块

The system SHALL provide a dedicated module `frontend/src/js/cm6-syntax-highlight.js` that encapsulates all CM6 syntax highlighting and editor theme definitions.

#### Scenario: Module exports required objects
- **WHEN** `cm6-syntax-highlight.js` is imported
- **THEN** it SHALL export `jotTheme` (CM6 EditorView theme config)
- **AND** it SHALL export `mdHighlight` (pre-composed array of `markdown()` + `syntaxHighlighting(jotHighlightStyle)`)

#### Scenario: Editor initialization uses extracted module
- **WHEN** `initCodeMirror()` is called with `useMdHighlight = true`
- **THEN** the extensions array SHALL use imported `jotTheme` and `mdHighlight` instead of inline definitions
- **AND** the visual result SHALL be identical to before extraction

## MODIFIED Requirements

### Requirement: initCodeMirror function

The `initCodeMirror()` function in `main.js` SHALL no longer contain inline `jotTheme` or `jotHighlightStyle` definitions. These SHALL be imported from the new module.

## REMOVED Requirements

None.