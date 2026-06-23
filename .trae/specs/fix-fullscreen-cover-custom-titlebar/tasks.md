# Tasks

- [x] Task 1: 修改 `.editor-panel.fullscreen` CSS 样式，使其不覆盖 `#topbar` 区域
  - 步骤 1: 将 `height: 100vh` 改为 `calc(100vh - 56px)`（56px 为 `#topbar` 高度）
  - 步骤 2: 调整 `top` 或 `transform` 使面板定位在 `#topbar` 下方

- [x] Task 2: 修改 `.editor-overlay.fullscreening` 样式，使其不遮挡 `#topbar`
  - 步骤 1: 将遮罩层全屏时的 `top: 0` 改为 `top: 56px`
  - 步骤 2: 调整 `height` 为 `calc(100vh - 56px)`
  - 步骤 3: 确保过渡动画平滑

- [x] Task 3: 验证全屏模式下自定义标题栏可见且窗口控制按钮可交互
  - 步骤 1: 在查看笔记页面测试全屏切换
  - 步骤 2: 在新建笔记页面测试全屏切换
  - 步骤 3: 验证最小化/最大化/关闭按钮可正常点击

# Task Dependencies

- [Task 2] 依赖于 [Task 1]（样式结构关联）
- [Task 3] 依赖于 [Task 1] 和 [Task 2]
