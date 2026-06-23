# 任务
- [x] 任务 1：调整 index.html 中的 #topbar DOM 结构：将 .topbar-dropdown 移至 .topbar-brand 左侧，.topbar-actions 仅保留窗口控制按钮
  - [x] 步骤 1.1：在 index.html 的 #topbar 中创建 `<div class="topbar-left">` 包裹 ☰ 按钮 + 品牌标识
  - [x] 步骤 1.2：将 .topbar-dropdown 从 .topbar-actions 移动到 .topbar-left 中作为第一个子元素
  - [x] 步骤 1.3：.topbar-actions 只保留三个窗口控制按钮
  - [x] 步骤 1.4：调整 flex 布局使搜索框居中

- [x] 任务 2：调整 style.css 中 .dropdown-menu 和布局相关 CSS
  - [x] 步骤 2.1：将 .dropdown-menu 的 `right: 0` 改为 `left: 0`
  - [x] 步骤 2.2：新增 .topbar-left 样式（flex 容器，gap: 4px）
  - [x] 步骤 2.3：确认 #topbar 的 `gap: 16px` 配合新布局自然

- [x] 任务 3：验证功能
  - [x] 步骤 3.1：点击 ☰ 菜单能从左上角正确展开（CSS `left: 0`）
  - [x] 步骤 3.2：窗口控制按钮（最小化/最大化/关闭）在右侧正常工作
  - [x] 步骤 3.3：全屏时 ☰ 菜单隐藏（CSS 选择器 `.editor-fullscreen .topbar-dropdown` 仍有效）
  - [x] 步骤 3.4：搜索框位置正确，交互不变

# 任务依赖
- 无依赖
