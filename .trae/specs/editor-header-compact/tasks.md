# Tasks

- [x] Task 1: 压缩 `.editor-header` padding-top
  - [x] SubTask 1.1: 打开 `frontend/src/style.css` L967-972
  - [x] SubTask 1.2: 将 `padding: 12px 12px 0` 改为 `padding: 4px 12px 0`
  - [x] SubTask 1.3: 验证编辑/新建/查看三种模式打开笔记时,header 按钮离顶部 4px

- [x] Task 2: 压缩 `.editor-input`（标题）padding
  - [x] SubTask 1.1: 打开 `frontend/src/style.css` L1044-1050
  - [x] SubTask 1.2: 将 `padding: 8px 0` 改为 `padding: 2px 0`
  - [x] SubTask 1.3: 验证标题上下间距明显收紧,字号/字重不变,focus 高亮正常

- [x] Task 3: 压缩 `.editor-section`（标签区）margin-bottom
  - [x] SubTask 1.1: 打开 `frontend/src/style.css` L1136-1138
  - [x] SubTask 1.2: 将 `margin: 2px 0 24px` 改为 `margin: 2px 0 6px`
  - [x] SubTask 1.3: 验证标签和正文间距为 6px(无需额外间距),3 种模式都生效

- [x] Task 4: 验证整体效果
  - [x] SubTask 1.1: 启动 `wails dev`,目视检查 header / 标题 / 标签视觉重心明显上移
  - [x] SubTask 1.2: 验证 CodeMirror 编辑器可见行数 +1（在小窗口下）
  - [x] SubTask 1.3: 验证 Markdown 预览模式首屏内容增加
  - [x] SubTask 1.4: 验证 6 个主题（default/nord/monokai-pro/light/tokyo-night/dark）下样式正常
  - [x] SubTask 1.5: 验证 fullscreen 全屏模式下也生效
  - [x] SubTask 1.6: 验证没有视觉错位（按钮悬停、标题 focus、标签 hover 状态）

# Task Dependencies
- [Task 1, 2, 3] 互相独立,可并行实施
- [Task 4] 依赖 [Task 1, 2, 3] 全部完成
