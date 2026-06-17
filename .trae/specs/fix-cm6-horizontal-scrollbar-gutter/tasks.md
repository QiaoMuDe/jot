# Tasks

- [x] Task 1: 调整行号 gutter 背景与层级
  - [x] SubTask 1.1: 在 `frontend/src/main.js` 的 CM6 主题中，将 `.cm-gutters` 的 `backgroundColor` 从 `transparent` 改为 `var(--card-bg)`
  - [x] SubTask 1.2: 在 `frontend/src/style.css` 中覆盖 `.cm-gutters` 背景色并设置 `z-index`，确保优先级高于默认样式
  - [x] SubTask 1.3: 调整 `.cm-lineNumbers .cm-gutterElement` 的 padding/color，保证行号在背景上清晰可读
- [x] Task 2: 跨模式验证
  - [x] SubTask 2.1: 在编辑页纯文本模式下输入超长行，确认水平滚动条不遮挡行号
  - [x] SubTask 2.2: 在查看页纯文本模式下打开含超长行的笔记，确认效果一致
  - [x] SubTask 2.3: 切换浅色/深色主题，确认 gutter 背景跟随主题变化

# Task Dependencies
无。
