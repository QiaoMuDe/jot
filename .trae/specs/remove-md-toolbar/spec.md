# 移除 MD 格式化工具栏 Spec

## Why

工具栏功能与 Markdown 直接输入语法相比缺乏独特价值，且引入了额外的 UI 复杂度（焦点管理、CSS 样式、渲染逻辑）。用户作为开发者熟悉 MD 语法，手打比点工具栏更快。移除工具栏可减少 ~380 行代码，简化编辑器区域，消除一类 UI 交互 bug。

## What Changes

- **HTML**：移除 `#editorToolbar` 及其内部所有按钮、分隔线、标题下拉面板
- **CSS**：移除 `.editor-toolbar`、`.heading-dropdown-*`、标题按钮相关全部样式
- **JS（格式化函数）**：移除 `formatBold/Italic/Strikethrough/InlineCode/Heading/Link/Image/List/Blockquote/Hr` 共 10 个函数
- **JS（事件绑定）**：移除 `initEditorToolbar()` 及其调用；移除 `tbBold/I/Code/Heading/Link/Image/List/Quote/Hr` 等 DOM 引用
- **JS（快捷键）**：从 CM6 `keymap.of()` 中移除 `Ctrl+B/I/U` 三条快捷键
- **JS（工具栏显隐控制）**：移除 `openEditor`、`switchToMarkdownMode`、类型切换中的工具栏显隐逻辑
- **JS（工具栏设置）**：移除 `loadToolbarSetting()` 函数及其调用；移除 `mdToolbarToggle` DOM 引用
- **HTML（设置页）**：移除"Markdown 笔记显示格式化工具栏"toggle 行及其提示文字
- **后绑定文件**：`wailsjs/go/main/App.d.ts` 和 `App.js` 无变更（不涉及后端绑定方法）

## Impact

- Affected specs: `add-md-formatting-toolbar/`（完全逆操作）、`add-toolbar-toggle-setting/`（完全逆操作）、`fix-md-toolbar-toggle-logic/`（不再需要）
- Affected code: `frontend/index.html`、`frontend/src/css/components/editor.css`、`frontend/src/main.js`
- 不涉及后端，不涉及绑定方法

## REMOVED Requirements

### Requirement: MD 格式化工具栏 UI

**Reason**: 用户直接手打 Markdown 语法即可替代，移除减少代码量和维护负担。

**Migration**: 无。工具栏所有功能（加粗/斜体/删除线/行内代码/标题/链接/图片/列表/引用/分割线）均可通过手打 MD 语法实现。

### Requirement: Ctrl+B/I/U 格式化快捷键

**Reason**: 与工具栏 UI 一起移除，保持一致。用户仍可直接输入 `**text**`、`*text*`、`~~text~~` 等 MD 语法。

**Migration**: 无。

### Requirement: MD 工具栏开关设置

**Reason**: 工具栏已移除，设置项不再有意义。

**Migration**: 从设置页删除该行，从 JS 中删除加载/保存逻辑。后端 `settings` 表中 `md_toolbar_enabled` 残留记录不产生任何影响。

### Requirement: 工具栏显隐控制

**Reason**: 工具栏 DOM 已移除，无需控制显隐。

**Migration**: 移除 `openEditor()`、`switchToMarkdownMode()`、类型切换 3 处的工具栏 `display` 控制代码。
