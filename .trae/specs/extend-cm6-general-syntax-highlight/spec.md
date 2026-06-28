# Extend CM6 General Syntax Highlight Spec

## Why

当前 CM6 语法高亮只支持 Markdown，但笔记已支持文件扩展名（`.js`、`.py`、`.go` 等），编辑器可以感知笔记类型。扩展为通用语法高亮后，打开不同语言的代码笔记就能自动获得正确的语法着色。

## What Changes

### cm6-syntax-highlight.js 模块重构

- **新建 `codeHighlightStyle`** — 独立的 `HighlightStyle.define()`，只含编程语言通用 tag（keyword、number、string、functionName、typeName、atom 等）
- **`jotHighlightStyle` → `mdHighlightStyle`** — 改名以区分职责，保留 MD 专有 tag（heading、strong、link、quote、list 等）
- **删除静态 `mdHighlight` 导出**，替换为 `getHighlightExtension(fileExt, enabled)` 工厂函数
- **语言解析器选择逻辑**：
  - 有 `@codemirror/lang-*` 原生包的优先使用
  - 其余语言通过 `@codemirror/legacy-modes` + `StreamLanguage` 加载

### 依赖变更

| 操作 | 包名 |
|---|---|
| 保留（已有） | `@codemirror/lang-markdown` |
| 新增 | `@codemirror/lang-javascript` |
| 新增 | `@codemirror/lang-css` |
| 新增 | `@codemirror/lang-html` |
| 新增 | `@codemirror/lang-json` |
| 新增 | `@codemirror/lang-python` |
| 新增 | `@codemirror/legacy-modes` |

### main.js 调用处调整

- `useMdHighlight` → 全局 toggle（不再区分 `.md` 与纯文本，toggle 控制全开关）
- 导入从 `{ jotTheme, mdHighlight }` → `{ jotTheme, getHighlightExtension }`
- 扩展数组调用从 `...(useMdHighlight ? mdHighlight : [])` → `...(useSyntaxHighlight ? getHighlightExtension(fileExt) : [])`

### 设置项调整 **BREAKING**

- 标签：从「纯文本编辑器启用 Markdown 语法高亮」→「启用 CM6 语法高亮」
- 提示：从「开启后，纯文本笔记也会显示 Markdown 语法高亮」→「关闭后所有笔记不再显示代码语法高亮」
- 存储键：从 `md_highlight_plain` → `cm_syntax_highlight`（旧值不兼容，重新编译后重置）
- 初始化逻辑：从 `.md 笔记始终高亮 + toggle 控制纯文本` 改为 toggle 全局控制

## Impact

- Affected specs: `extract-cm6-highlight-module`（刚提取的模块会被大幅改写）
- Affected code:
  - `frontend/src/js/cm6-syntax-highlight.js` — 核心改动
  - `frontend/src/main.js` — import 和调用处调整
  - `frontend/index.html` — 设置标签文案
  - `frontend/package.json` — 新增 6 个依赖

## ADDED Requirements

### Requirement: 代码语法高亮样式

The system SHALL provide a `codeHighlightStyle` that defines visual styles for programming language syntax nodes.

#### Scenario: Common programming tags have correct colors
- **GIVEN** `codeHighlightStyle` is defined
- **THEN** it SHALL include at least: `keyword`, `number`, `string`, `regexp`, `variableName`, `typeName`, `className`, `functionName`, `definition(functionName)`, `comment`, `docComment`, `atom`, `propertyName`, `attributeName`, `namespace`, `operator`, `punctuation`, `meta`, `labelName`, `escape`, `character`
- **AND** all color values SHALL reference CSS variables for theme compatibility

### Requirement: 根据扩展名选择语言解析器

The system SHALL select the correct CM6 language parser based on the note's file extension.

#### Scenario: Native parser preferred for known extensions
- **GIVEN** file extension is `.js`, `.ts`, `.jsx`, `.tsx`, `.css`, `.html`, `.json`, or `.py`
- **WHEN** `getHighlightExtension(ext)` is called
- **THEN** it SHALL use the corresponding `@codemirror/lang-*` native parser

#### Scenario: Legacy parser fallback
- **GIVEN** file extension is not covered by native parsers (e.g. `.go`, `.rs`, `.java`, `.sql`, `.sh`, `.yaml`)
- **WHEN** `getHighlightExtension(ext)` is called
- **THEN** it SHALL use `StreamLanguage` + `@codemirror/legacy-modes` to load the parser

#### Scenario: Unknown extension
- **GIVEN** file extension is unknown/unrecognized
- **WHEN** `getHighlightExtension(ext)` is called
- **THEN** it SHALL return empty array `[]` (no highlighting)

### Requirement: 设置全局开关

The system SHALL provide a global toggle for CM6 syntax highlighting.

#### Scenario: Toggle ON
- **GIVEN** the toggle is ON (default)
- **WHEN** a note is opened in the editor
- **THEN** the correct language parser and highlight style SHALL be applied based on file extension

#### Scenario: Toggle OFF
- **GIVEN** the toggle is OFF
- **WHEN** any note is opened in the editor
- **THEN** no syntax highlighting SHALL be applied, regardless of file extension

## MODIFIED Requirements

### Requirement: 设置页面文案

The setting label SHALL be updated from `纯文本编辑器启用 Markdown 语法高亮` to `启用 CM6 语法高亮`, and the hint from `开启后，纯文本笔记也会显示 Markdown 语法高亮` to `关闭后所有笔记不再显示代码语法高亮`. The storage key SHALL change from `md_highlight_plain` to `cm_syntax_highlight`.

### Requirement: initCodeMirror highlight parameter

The `initCodeMirror()` function's highlight parameter SHALL change from `useMdHighlight` (boolean, MD-specific) to `useSyntaxHighlight` (boolean, global toggle), and the extensions array SHALL use the factory function `getHighlightExtension(fileExt)` instead of the static `mdHighlight` array.

## REMOVED Requirements

### Requirement: mdHighlight static export

**Reason**: Replaced by the more flexible `getHighlightExtension(fileExt, enabled)` factory function.

**Migration**: Any code importing `mdHighlight` from `cm6-syntax-highlight.js` SHALL import `getHighlightExtension` instead and pass the file extension.

### Requirement: md_highlight_plain storage key

**Reason**: Replaced by `cm_syntax_highlight` with new semantics (global toggle, not MD-specific).

**Migration**: None. Old key will be ignored after rebuild.