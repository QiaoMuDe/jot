# Tasks
- [x] Task 1: 在 HTML 的 `.calendar-sidebar` 中新增统计区域结构（位于 `.calendar-grid` 之后）
- [x] Task 2: 在 CSS 中新增 `.calendar-stats` 及相关子元素样式
- [x] Task 3: 在 JS 中新增统计计算和渲染函数
  - [x] 新增 `computeMonthStats(counts)` 函数：从 `counts` 对象计算总笔记数、记天数、最长连续天数
  - [x] 在 `renderCalendar()` 末尾调用统计渲染函数
  - [x] 新增 `renderMonthStats(stats)` 函数：将统计数据渲染到 DOM

## 任务依赖关系
- Task 1（HTML 结构）需先于 Task 3（JS 渲染）
- Task 2（CSS 样式）无依赖，可并行
