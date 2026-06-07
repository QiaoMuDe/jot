# Tasks

- [x] Task 1: Go 后端启用 Frameless 模式并清理 DWM 代码
  - [x] SubTask 1.1: `main.go` 中设置 `Frameless: true`，添加 `CSSDragProperty` 和 `CSSDragValue`
  - [x] SubTask 1.2: 移除 `BackgroundColour` 和 `Windows` 选项中的 `CustomTheme`
  - [x] SubTask 1.3: `app.go` 中移除 `getWindowsOptions()`、`ApplyWindowTheme()`、`findMainWindow()`、`setWindowAttribute()` 及所有 Windows API 常量
  - [x] SubTask 1.4: 移除 `main.js` 中调用 `ApplyWindowTheme` 的代码

- [x] Task 2: 前端添加自定义标题栏 HTML
  - [x] SubTask 2.1: `frontend/index.html` 中在 `#topbar` 上方插入 `#windowTitleBar`
  - [x] SubTask 2.2: 标题栏包含：左侧可拖拽区域（应用名 "Jot"）、右侧三个控制按钮（最小化/最大化/关闭）

- [x] Task 3: 前端添加自定义标题栏 CSS
  - [x] SubTask 3.1: `frontend/src/style.css` 中添加 `.window-titlebar` 基础样式（高度 32px、flex 布局、user-select: none）
  - [x] SubTask 3.2: 添加 `.window-titlebar-drag` 拖拽区域样式（flex: 1、--wails-draggable: drag）
  - [x] SubTask 3.3: 添加 `.window-titlebar-controls` 按钮容器样式
  - [x] SubTask 3.4: 添加 `.window-btn` 按钮样式（hover 效果、无拖拽）
  - [x] SubTask 3.5: 使用 CSS 变量 `--topbar-bg`、`--text-primary`、`--border` 确保主题适配
  - [x] SubTask 3.6: 调整 `#topbar` 样式，移除顶部圆角或调整与标题栏的衔接

- [x] Task 4: 前端绑定窗口控制按钮事件
  - [x] SubTask 4.1: `frontend/src/main.js` 中导入 `WindowMinimise`、`WindowToggleMaximise`、`Quit` 运行时方法
  - [x] SubTask 4.2: 为最小化按钮绑定 `WindowMinimise()`
  - [x] SubTask 4.3: 为最大化/还原按钮绑定 `WindowToggleMaximise()`，并切换图标
  - [x] SubTask 4.4: 为关闭按钮绑定 `Quit()`
  - [x] SubTask 4.5: 监听窗口最大化状态事件（`wails:window:maximise` / `wails:window:unmaximise`）更新按钮图标

- [x] Task 5: 验证与测试
  - [x] SubTask 5.1: 编译运行，确认窗口无边框正常显示
  - [x] SubTask 5.2: 测试拖拽功能（标题栏空白区域拖拽移动窗口）
  - [x] SubTask 5.3: 测试双击最大化/还原
  - [x] SubTask 5.4: 测试三个控制按钮功能
  - [x] SubTask 5.5: 切换全部 6 套主题，确认标题栏颜色跟随变化
  - [x] SubTask 5.6: 确认失去焦点时标题栏不会变黑
  - [x] SubTask 5.7: 运行 `golangci-lint fmt ./... && golangci-lint run ./...` 通过

# Task Dependencies

- Task 2 依赖 Task 1（先清理后端代码）
- Task 3 依赖 Task 2（先有 HTML 结构）
- Task 4 依赖 Task 2（先有 HTML 元素）
- Task 5 依赖 Task 3 和 Task 4
