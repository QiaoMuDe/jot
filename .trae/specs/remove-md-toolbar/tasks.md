# Tasks

- [x] Task 1: 移除工具栏 HTML — 从 `index.html` 删除 `#editorToolbar` 整个块（第 157-202 行，含所有按钮、分隔线、标题下拉面板）

- [x] Task 2: 移除工具栏 CSS — 从 `editor.css` 删除 `.editor-toolbar`、`.editor-toolbar-btn`、`.editor-toolbar-btn-wrapper`、`.editor-toolbar-divider`、`.heading-dropdown-panel`、`.heading-dropdown-item`、`.hd-tag`、`.hd-label` 及预览模式隐藏工具栏的全部样式（约 130 行）
  - 保留 `.editor-panes` 的 flex 布局样式（该容器继续使用）

- [x] Task 3: 移除 JS 格式化函数、快捷键、工具栏初始化 — 从 `main.js` 删除：
  - [x] 10 个 `format*()` 函数（第 6175-6353 行）
  - [x] `initEditorToolbar()` 函数（第 6355-6390 行）及在 `init()` 中的调用
  - [x] CM6 keymap 中 Ctrl+B/I/U 三条快捷键（第 298-300 行）
  - [x] `els` 对象中所有工具栏按钮引用（`editorToolbar`, `tbBold`, `tbItalic`, `tbStrikethrough`, `tbCode`, `tbHeading`, `headingDropdown`, `tbLink`, `tbImage`, `tbList`, `tbQuote`, `tbHr`）

- [x] Task 4: 移除工具栏设置项 UI 和 JS 逻辑 — 从以下位置删除：
  - [x] `index.html` 中 `mdToolbarToggle` 所在行（第 335-344 行）
  - [x] `main.js` 中 `loadToolbarSetting()` 函数及在 `init()` 中的调用
  - [x] `main.js` 中 `mdToolbarToggle` 的 change 事件监听（第 3978-3987 行附近）
  - [x] `els.mdToolbarToggle` 引用

- [x] Task 5: 移除工具栏显隐控制 — 从 `main.js` 以下 3 处删除 `.editor-toolbar` `style.display` 控制代码：
  - [x] `openEditor()` 中（第 2552-2556 行）
  - [x] 类型切换 handler 中（第 3919-3922 行）
  - [x] `switchToMarkdownMode()` 中（第 4720-4723 行）

- [x] Task 6: 构建验证 — `npm run build` 通过，无错误

# Task Dependencies

- Task 1~5 互不依赖，可并行执行
- Task 6 依赖前 5 个任务全部完成
