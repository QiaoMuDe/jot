# 代码高亮主题配置 Spec

## Why

当前 CM6 通用语法高亮模块中，`codeHighlightStyle` 是一套硬编码的 Monokai Dimmed 配色方案（紫色关键字、金色字符串等）。用户希望在设置页中增加一个代码高亮主题选择器，将当前配色定义为默认主题，并预留后续新增更多主题的能力。

## What Changes

- **cm6-syntax-highlight.js**: 将 `codeHighlightStyle` 重构为主题系统，定义 `codeHighlightThemes` 对象，当前 Monokai Dimmed 配色作为 `monokai-dimmed` 主题。`getHighlightExtension()` 新增 `themeName` 参数以支持选择不同主题。
- **main.js**: 新增 `code_highlight_theme` 设置项（默认 `monokai-dimmed`），`loadCodeHighlightThemeSetting()` 加载，`initCodeMirror()` 传递 themeName 参数，设置变更时即时更新编辑器高亮。
- **index.html**: 设置页「编辑器选项」新增代码高亮主题选择器，使用分段控件（segmented control），当前仅一个「默认 Monokai」选项。
- **CSS**: 复用已有的 `.segmented-control` 分段控件样式，无需新增。

## Impact

- Affected specs: 编辑器选项 → 代码高亮主题选择
- Affected code:
  - `frontend/src/js/cm6-syntax-highlight.js` — 主题系统重构
  - `frontend/src/main.js` — 加载/保存/应用主题设置
  - `frontend/index.html` — 设置页 UI 新增

## ADDED Requirements

### Requirement: 代码高亮主题系统

The system SHALL support a configurable code syntax highlighting theme that can be selected in the settings page.

#### Scenario: 默认主题
- **WHEN** 用户首次启动应用
- **THEN** 代码高亮使用 `monokai-dimmed` 主题（即当前 Monokai Dimmed 配色方案）
- **AND** 设置页显示「默认 Monokai」为选中态

#### Scenario: 在设置页切换主题
- **WHEN** 用户在设置页「编辑器选项」中点击代码高亮主题分段控件按钮
- **THEN** 高亮主题立即切换
- **AND** 当前已打开的编辑器（如果有）即时应用新主题
- **AND** 设置持久化到后端 `code_highlight_theme` 键

#### Scenario: 后续新增主题
- **WHEN** 开发者在 `cm6-syntax-highlight.js` 的 `codeHighlightThemes` 中添加新主题
- **THEN** 前端无需额外代码改动即可自动在设置页显示新选项
- **AND** 原有代码兼容，不会破坏现有功能

## MODIFIED Requirements

### Requirement: getHighlightExtension 工厂函数

原函数签名 `getHighlightExtension(fileExt)` 修改为 `getHighlightExtension(fileExt, themeName)`。

- `themeName` 默认值 `'monokai-dimmed'`，向后兼容
- 内部从 `codeHighlightThemes[themeName]` 获取对应 `HighlightStyle`，未知 themeName 回退到 `codeHighlightThemes['monokai-dimmed']`

### Requirement: initCodeMirror

原函数签名增加 `themeName` 参数（可选，默认 `'monokai-dimmed'`）：

```js
function initCodeMirror(container, content = '', readOnly = false, useSyntaxHighlight = true, fileExt = '.md', themeName = 'monokai-dimmed')
```

内部调用 `getHighlightExtension(fileExt, themeName)` 传入 themeName。