# Tasks

- [x] Task 1: HTML — FAB 组 DOM 结构调整与 AI 按钮插入
  - 在 `frontend/index.html` 的 `#fabGroup` 中：
    1. 将 `backToTopBtn` 移到 `fabNewNote` 下方
    2. 在 `fabNewNote` 和 `backToTopBtn` 之间插入新的 `fabAI` 按钮（AI 图标 SVG）
    3. 移除 `flex-direction: column-reverse` 依赖（通过 CSS 改为 `column`）

- [x] Task 2: CSS — FAB 组样式适配
  - 在 `frontend/src/css/components/main-content.css` 中：
    1. `.fab-group` 从 `column-reverse` 改为 `column`
    2. 新增 `.fab-group.scrolled` 样式：`bottom` 从 `28px` 变为 `90px`，`transition` 动画
    3. 新增 `.fab-ai` 按钮样式（区别于 `.fab-add` 的 accent 色，使用紫色）
    4. `.fab-top` 样式微调：移除 `transform: translateY(10px)`，保留其余显隐逻辑

- [x] Task 3: JS — DOM 引用、事件绑定与滚动逻辑更新
  - 在 `frontend/src/main.js` 中：
    1. 在 `els` 对象中新增 `fabAI` 引用（`$('fabAI')`）
    2. 在 `initEventListeners` 中为 `fabAI` 绑定点击事件 → 调用 `switchView('ai-chat')`
    3. 重写 `initScrollbarAutoHide` 中的滚动监听逻辑：
       - 当 `scrollTop > 300`：给 `fabGroup` 加 `.scrolled` 类，同时 `backToTopBtn` 显示（`.visible`）
       - 当 `scrollTop <= 300`：移除 `.scrolled` 类，`backToTopBtn` 隐藏

# Task Dependencies
- Task 2 依赖 Task 1（CSS 样式依赖 HTML 中已有的按钮结构）
- Task 3 依赖 Task 1 和 Task 2（JS 逻辑依赖 DOM 元素存在和 CSS 类定义）
