# Tasks

- [x] Task 1: CSS 新增下移和入场动画关键帧
  - 在 `frontend/src/css/components/todo.css` 中新增：
    - `.todo-item.todo-shift` — 使用 `transform: translateY()` + `transition` 实现下移
    - `@keyframes todoItemEnter` — 新条目入场：从 `translateY(-30px) scale(0.96) opacity(0)` 到最终位置
    - `.todo-item.todo-item-enter` — 应用 `todoItemEnter` 动画
  - 缓动曲线：`cubic-bezier(0.34, 1.56, 0.64, 1)`（带轻微弹性）
  - 时长：下移 280ms，入场 350ms

- [x] Task 2: 重写 addTodo() 的动画编排逻辑
  - 在 `frontend/src/main.js` 中重写 `addTodo()`：
    - 调用 `CreateTodo` API 获取新条目数据
    - 使用 FLIP 技术实现已有条目的平滑下移（`getBoundingClientRect` 动态计算位移）
    - 新条目先以 `opacity: 0` 不可见方式 prepend（占位但不闪烁）
    - 380ms 后新条目以 `todo-item-enter` 入场动画展示
    - 动画结束后清理 class、transform、todoAnimating 标志
    - 提取 `buildTodoItemHTML()`、`insertNewTodoItem()`、`updateTodoStatsAfterAdd()` 辅助函数

- [x] Task 3: 处理筛选切换场景
  - 当前筛选不是 `'active'` 时，切换到 `'active'` 筛选并用 `loadTodos()` 刷新
  - 保持现有行为不变

- [x] Task 4: CSS 动画属性微调
  - `todo-shift` 使用 `!important` 覆盖 `.todo-item` 基础 transition，确保只对 transform 生效
  - `todo-item-enter` 在 `animationend` 后清理
  - `prefers-reduced-motion` 降级由全局 CSS 处理（animations.css 中 `*` 选择器覆盖所有动画）

# Task Dependencies
- [Task 1] (CSS) → [Task 2] (JS) — CSS 动画类先就位，JS 才能引用
- [Task 3] 可独立实现
- [Task 4] 依赖于 [Task 1] + [Task 2]
