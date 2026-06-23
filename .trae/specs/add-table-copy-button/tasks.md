# Tasks

- [x] Task 1: 表格复制按钮 JS 逻辑 — 在 `updatePreview()` 中为每个 `.md-rendered table` 添加 `.copy-table-btn` 按钮，实现点击复制整个表格为 Markdown 格式的功能
  - [x] 1.1 遍历 `els.mdRendered.querySelectorAll('table')`，跳过已有 `.copy-table-btn` 的（通过 `table-wrapper` 类检测）
  - [x] 1.2 创建 `<button class="copy-table-btn">复制</button>`，`title="复制表格"`
  - [x] 1.3 实现 `tableToMarkdown(tableEl)` 函数，将 HTML table 转为 Markdown 表格语法文本
  - [x] 1.4 点击事件：调用 `tableToMarkdown` → `navigator.clipboard.writeText()` → 成功显示 `SVGS.checkmark + ' 已复制'` 1.5s 恢复 → 失败显示 `SVGS.xmark + ' 复制失败'` 1s 恢复
  - [x] 1.5 用 `<div class="table-wrapper">` 包裹 table 作为定位容器，按钮 `appendChild` 到 wrapper

- [x] Task 2: 表格复制按钮 CSS 样式 — 基于 `.copy-code-btn` 样式适配 table（因为 table 需要父容器 wrapper 才能做 `position: relative`）
  - [x] 2.1 新增 `.md-rendered .table-wrapper` 样式类（`position: relative`，给内部复制按钮做定位参考；`margin: 1.2em 0` 与代码块一致）
  - [x] 2.2 新增 `.copy-table-btn` 样式（position absolute 垂直居中于 wrapper、opacity 0 / hover 显示、hover scale 放大）

# Task Dependencies

- Task 2 依赖 Task 1（CSS 样式服务于 JS 生成的 DOM 结构）
