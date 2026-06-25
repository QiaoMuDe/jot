# Tasks

- [x] Task 1: 工具栏 HTML 结构 — 在 `editor-panes` 顶部插入 `.editor-toolbar` 容器，包含所有按钮的 SVG 图标、标题下拉选择器、H1-H6 选项面板
  - [x] 在 `index.html` 中 `editor-panes` 内添加 `.editor-toolbar` 作为第一个子元素
  - [x] 工具栏内按钮顺序：加粗 / 斜体 / 删除线 / 行内代码 / 分隔符 / 标题下拉 / 分隔符 / 链接 / 图片 / 无序列表 / 引用 / 分割线
  - [x] 标题下拉面板：包含 `.heading-dropdown-btn`（显示 H）+ `.heading-dropdown-panel`（6 个选项：H1-H6，格式 `H1  标题 1`）
- [x] Task 2: 工具栏 CSS 样式 — 实现 36px 高度、flex 居左、28×28px 按钮、6px 圆角、CSS 变量主题适配、prefers-reduced-motion 降级
  - [x] `.editor-toolbar` flex row 布局，高度 36px，背景/边框引用 CSS 变量
  - [x] `.editor-toolbar-btn` 28×28px，圆角 6px，transition 150ms，hover/active 态
  - [x] `.editor-toolbar-divider` 1px 竖线，16px 高度，CSS 变量颜色
  - [x] `.heading-dropdown-panel` 浮动定位，140px 宽，阴影/边框/圆角，选项 hover accent 竖条
  - [x] 预览模式下 `.editor-overlay[data-mode="preview"] .editor-toolbar { display: none }`
  - [x] `prefers-reduced-motion` 降级
- [x] Task 3: 格式化操作函数 — 实现 11 个格式化函数（bold/italic/strikethrough/code/heading1-6/link/image/list/quote/hr），都基于 `cmEditor.dispatch` 操作 CM6 的 state
  - [x] `formatBold()` — 选中包裹 `**`，无选中插入 `****` 光标居中
  - [x] `formatItalic()` — 选中包裹 `*`，无选中插入 `**` 光标居中
  - [x] `formatStrikethrough()` — 选中包裹 `~~`，无选中插入 `~~~~` 光标居中
  - [x] `formatInlineCode()` — 选中包裹 `` ` ``，无选中插入 `` `` `` 光标居中
  - [x] `formatHeading(level)` — 行首插入 `#~###### `（level 1-6）
  - [x] `formatLink()` — 有选中包裹 `[]()` + 弹窗 URL，无选中插入 `[链接文字]()` + 弹窗
  - [x] `formatImage()` — 插入 `![]()` + 弹窗 URL
  - [x] `formatList()` — 行首插入 `- `
  - [x] `formatBlockquote()` — 行首插入 `> `
  - [x] `formatHr()` — 插入 `\n---\n`
- [x] Task 4: 快捷键注册 — 在 CM6 keymap 中注册 Ctrl+B/I/U 三个快捷键，指向对应的格式化函数
  - [x] `Ctrl+B` → `formatBold()`
  - [x] `Ctrl+I` → `formatItalic()`
  - [x] `Ctrl+U` → `formatStrikethrough()`
- [x] Task 5: 事件绑定与显隐控制 — 工具栏按钮 click 事件绑定 + 标题下拉展开/关闭 + 焦点保持 + 根据 `state.noteType` 和 `data-mode` 控制显隐
  - [x] 按钮 click 事件 + 格式化后 `cmEditor.focus()`
  - [x] 标题下拉：点击按钮 toggle 面板，点击外部关闭面板
  - [x] `populateEditor()` 中根据 `state.noteType` 控制 `.editor-toolbar` 显隐
  - [x] 类型切换按钮点击时同步工具栏显隐
  - [x] 编辑/预览模式切换时同步工具栏显隐

# Task Dependencies

- [Task 1] → [Task 2]（HTML 结构必须先在才能写样式，但可以并行编写）
- [Task 3] 依赖于 CM6 编辑器已初始化
- [Task 4] 依赖于 [Task 3]（快捷键调用格式化函数）
- [Task 5] 依赖于 [Task 1]（需要 DOM 元素绑定事件）
