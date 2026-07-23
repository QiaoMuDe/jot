# Tasks
- [x] Task 1: 移除 HTML 中的旧右键菜单元素 `#aiSessionContextMenu`
- [x] Task 2: 移除 CSS 中的旧右键菜单样式（`.ai-session-context-menu` 及相关规则）
- [x] Task 3: 重构 JS 逻辑，让右击弹出与更多按钮相同的统一菜单
  - [ ] 移除 `sessionContextMenu` 变量引用及相关初始化
  - [ ] 移除 `closeSessionContextMenu()` / `showSessionContextMenu()` 函数
  - [ ] 移除全局点击/Escape 关闭旧右键菜单的逻辑
  - [ ] 修改 `item.addEventListener('contextmenu', ...)` 使其复用更多按钮的菜单构建逻辑，追加"导出"项
  - [ ] 保持更多按钮菜单的原有行为（置顶、重命名、删除）
  - [ ] 统一菜单添加"导出"项，复用 `ExportAISessionAsMarkdown` 逻辑
  - [ ] 右击菜单位置跟随鼠标，更多按钮菜单位置跟随按钮

## 任务依赖关系
- Task 1, 2, 3 可并行（修改不同文件）但 Task 3 是核心逻辑
