# Tasks

- [x] Task 1: 引用笔记选择器 Enter 键支持 — 按 Enter 触发确认引用
  - [x] 在 `ai-chat.js` 中引用笔记选择器初始化完成后，为 `refModal` 添加 `keydown` 监听
  - [x] 捕获 `e.key === 'Enter'`，排除焦点在搜索输入框 `#aiNoteRefSearch` 中的场景
  - [x] 触发 `refConfirm.click()` 执行确认引用
  - [x] 通过 `e.stopPropagation()` 防止事件冒泡

- [x] Task 2: 引用笔记选择器 ESC 返回 AI 聊天界面
  - [x] `ai-chat.js` 现有 ESC 监听（第 1186-1195 行）已会关闭浮层，无需修改
  - [x] `main.js` 全局 `handleKeyboardNavigation` ESC 分支头部添加判断：检测 `#aiNoteRefModal` 的 `display !== 'none'` 时，直接 `return` 跳过全局 ESC 导航（不切换到 grid 视图）

- [x] Task 3: 笔记查看/编辑/新建页面屏蔽 Ctrl+数字快捷键
  - [x] `main.js` `handleKeyboardNavigation` 中 Ctrl+数字分支（`(e.ctrlKey || e.metaKey)` 块）入口处添加判断
  - [x] 检测 `#viewEditor.active` 或 `#viewPreview.active` 是否存在（编辑器/查看器/新建笔记页面），若是则 `return` 跳过 Ctrl+数字处理

# Task Dependencies
- [Task 1]、[Task 2]、[Task 3] 三者独立，可并行实现
