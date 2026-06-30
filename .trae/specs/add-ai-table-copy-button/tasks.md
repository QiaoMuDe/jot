# Tasks

- [x] Task 1: AI 消息表格复制按钮 JS 逻辑 — 在 `renderMarkdown()` 中为每个 AI 消息内的 `<table>` 添加复制按钮，放置在第一行最后一列表头（`th:last-child`）内
  - [x] 1.1 遍历 `el.querySelectorAll('table')`，跳过已有按钮的（通过 `.table-copy-btn` 检测）
  - [x] 1.2 获取第一行最后一个 `th`（`table.querySelector('tr:first-child th:last-child')`）
  - [x] 1.3 创建 `<button class="table-copy-btn">复制</button>`，`title="复制表格"`
  - [x] 1.4 在 ai-chat.js 内内联 `tableToMarkdown()` 将 table 转为 Markdown 文本
  - [x] 1.5 点击事件：`navigator.clipboard.writeText()` → 成功显示 `✓ 已复制` 1.5s 恢复 → 失败显示 `✗ 复制失败` 1s 恢复
  - [x] 1.6 将按钮 `appendChild` 到目标 `th` 单元格

- [x] Task 2: AI 消息表格复制按钮 CSS 样式 — 基于 `.code-copy-btn` 样式适配，按钮在 `th` 内居右对齐、垂直居中
  - [x] 2.1 新增 `.ai-msg-assistant .table-copy-btn` 样式类（`position: absolute` 居右 + `top: 50%` 垂直居中、`opacity: 0` / `tr:hover` 显示）
  - [x] 2.2 复制成功态 `.table-copy-btn.copied`（绿色文字+边框）
  - [x] 2.3 hover 放大效果 `transform: translateY(-50%) scale(1.08)`

# Task Dependencies

- Task 2 依赖 Task 1（CSS 样式服务于 JS 生成的 DOM 结构）
