# Tasks

## 前置说明
两项任务无依赖关系，可并行执行。

## 任务列表

- [x] Task 1: AI 会话标题动态更新
  - 1.1 在 `switchSession()` 中将 `#aiChatTitle` 文本设置为当前会话的标题
  - 1.2 在 `showWelcome()` 中将 `#aiChatTitle` 文本重置为 "AI 助手"
  - 1.3 在 `createSession()` 中将 `#aiChatTitle` 文本设置为 "AI 助手"
  - 1.4 在 `saveSessionMessages()`（保存后标题可能变更）或 `loadSessionList()` 后检查是否需同步标题
  - 1.5 在重命名会话回调（startInlineEdit 的 finish 函数）中同步标题

- [x] Task 2: 笔记卡片右键菜单视口溢出修复
  - 2.1 在 `showContextMenu()` 中设置菜单 `left`/`top` 后，使用 `requestAnimationFrame` 或 setTimeout 等待渲染
  - 2.2 获取菜单的 `offsetHeight`，检查是否超出 `window.innerHeight`
  - 2.3 若超出，调 `top = window.innerHeight - menuHeight - 8`（留 8px 边距）
  - 2.4 确保 `top` 不低于 8px（防止菜单上边缘溢出顶部）
  - 2.5 保持现有的 `transform-origin` 逻辑不变

# Task Dependencies
- 无依赖关系，两项任务可并行完成。
