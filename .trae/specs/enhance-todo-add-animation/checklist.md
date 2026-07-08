# Checklist

## CSS 动画
- [x] `todo.css` 中新增 `.todo-item.todo-shift` 样式，使用 `transform: translateY()` + `transition`
- [x] `todo.css` 中新增 `@keyframes todoItemEnter` 关键帧动画
- [x] `todo.css` 中新增 `.todo-item.todo-item-enter` class 引用 `todoItemEnter`
- [x] 下移使用 `cubic-bezier(0.34, 1.56, 0.64, 1)` 缓动
- [x] 入场使用 `ease-out` 缓动
- [x] `prefers-reduced-motion` 降级支持

## JS 动画编排
- [x] `addTodo()` 创建新条目后先触发已有条目下移
- [x] 下移距离通过 `getBoundingClientRect` 动态计算（FLIP 技术）
- [x] 新条目先以不可见方式 prepend 占位，下移完成后再展示
- [x] 新条目有独立的入场动画（`todoItemEnter`）
- [x] 动画结束后正确清理 class（无残留 transform）
- [x] 非 `'active'` 筛选场景自动切换并 fallback 到 `loadTodos()`

## 视觉验证（需本地构建后确认）
- [ ] 新增待办时，已有条目先整体下移一个位置
- [ ] 新条目从上方优雅滑入插入空位
- [ ] 两段动画衔接丝滑，无闪烁/卡顿
- [ ] 多次快速添加时动画表现稳定
- [ ] 不同主题（深色/浅色）下动画正常
- [ ] 空列表新增时正确显示
