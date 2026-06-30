# Tasks
- [x] 任务 1: 修改 `data-view.css` 移除 `.data-content` 上的 `overflow-y: auto` 和 `scrollbar-gutter: stable`，并调整相关属性
- [x] 任务 2: 修改 `scrollbar.css` 移除 `.data-content` 的自定义滚动条样式引用
- [x] 附加: 更新 `main.js` 中 `getScrollContainer()` 和 `initScrollbarAutoHide()` 对 `.data-content` 的引用
- [x] 附加: 移除 `main-content.css` 中 `#viewData.view` 的 `padding-right: 0` 特殊规则

# Task Dependencies
- [任务 2] 依赖于 [任务 1]（确保滚动行为正确后清理样式）
