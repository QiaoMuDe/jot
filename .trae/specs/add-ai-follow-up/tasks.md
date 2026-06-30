# Tasks

- [x] Task 1: AI 回复气泡操作栏添加「追问」按钮
  - [x] 在 `createMsgActions()` 的 assistant 分支中追加追问按钮
  - [x] 按钮文字「追问」，class 为 `ai-followup-btn`
  - [x] 点击处理：获取输入框元素 `#aiInput`，将 `value` 设为 `> 关于上条回复：`，调用 `inputEl.focus()`

- [x] Task 2: 样式适配
  - [x] 追问按钮继承 `.ai-msg-actions button` 统一样式（间距、hover、颜色）
  - [x] 增加 `.ai-followup-btn` 覆盖宽度为 `auto`、`padding: 0 6px`、`font-size: 0.75rem` 以适应文字按钮

## Task Dependencies

- Task 1 独立，无前置依赖
- Task 2 依赖于 Task 1 中按钮添加后的视觉效果评估
