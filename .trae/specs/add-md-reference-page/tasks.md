# Tasks

- [ ] Task 1: 在 index.html 中添加 Markdown 语法手册视图 HTML 结构和更多菜单项
  - SubTask 1.1: 在更多菜单中添加 `data-action="md-ref"` 的菜单项，标题「Markdown 语法手册」，提示 `Ctrl+8`
  - SubTask 1.2: 在 `#mainContent` 内创建 `#viewMdRef` 视图，包含 view-header（返回按钮 + 标题「Markdown 语法手册」）和 `.md-ref-content` 容器
  - SubTask 1.3: 在 `.md-ref-content` 中为 10 个语法类别编写完整的展示卡片 HTML（源码区 + 预览区），每个分类的示例代码需精心设计以展示语法要点
- [ ] Task 2: 在 main.js 中注册视图并挂载菜单项事件
  - SubTask 2.1: 在 `els` 对象中添加 `viewMdRef: $('viewMdRef')`
  - SubTask 2.2: 在 `switchView()` 的 `viewMap` 中添加 `md-ref` 映射
  - SubTask 2.3: 在更多菜单的 click 事件处理中添加 `item.dataset.action === 'md-ref'` 分支
  - SubTask 2.4: 为 mdRefBackBtn 添加返回事件监听
  - SubTask 2.5: 在 Ctrl+快捷键映射中添加 `Ctrl+8 → switchView('md-ref')`
  - SubTask 2.6: 在快捷键帮助数组中添加对应条目
  - SubTask 2.7: 在 `showTargetView` 的 switch 中添加 `case 'md-ref'`（目前无异步加载，留空即可）
- [ ] Task 3: 编写 CSS 样式美化 Markdown 语法手册页面
  - SubTask 3.1: 定义 `.md-ref-content` 基本布局（滚动容器、内边距）
  - SubTask 3.2: 定义 `.md-ref-card` 卡片样式（白底、圆角、阴影、hover 效果）
  - SubTask 3.3: 定义两栏布局 `.md-ref-panel`（源码列 `~45%` + 预览列 `~55%`）
  - SubTask 3.4: 定义源码区样式（深色半透明背景、等宽字体、圆角）
  - SubTask 3.5: 定义预览区样式（浅色背景、内边距、MD 渲染样式）
  - SubTask 3.6: 定义交错淡入动画 `.md-ref-card` 的 `@keyframes` 和 `animation-delay` 阶梯
  - SubTask 3.7: 定义卡片左上角标签（`.md-ref-badge`）样式
  - SubTask 3.8: 定义复制按钮样式（与表格复制按钮风格统一）
  - SubTask 3.9: 确保与编辑器已有的 Markdown 渲染样式兼容
- [ ] Task 4: 实现复制功能
  - SubTask 4.1: 编写 `setupRefCopyButtons()` 函数，为每个源码区的 `<pre>` 添加复制按钮
  - SubTask 4.2: 复制按钮使用 JavaScript 定位（与表格复制按钮一致），hover 放大效果
  - SubTask 4.3: 复制时显示「已复制 ✓」反馈，500ms 后恢复
  - SubTask 4.4: 在 `init()` 中调用 `setupRefCopyButtons()`
- [ ] Task 5: 数据管理与回收站视图折叠侧栏逻辑同步
  - 无需额外操作，`switchView()` 中非 grid 视图的自动折叠逻辑已覆盖 `md-ref`

## Task Dependencies

- Task 1 先于 Task 2 / Task 3 / Task 4
- Task 2 和 Task 3 可以并行
- Task 4 依赖于 Task 1（DOM 结构存在）
