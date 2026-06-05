# Tasks

- [x] 任务 1：HTML 新增模式切换按钮
  - 在 `index.html` 的 `.editor-header-actions` 中关闭按钮左侧，插入 3 个 `.mode-btn` 按钮（纯文本/分栏/预览）
  - 默认「纯文本」按钮带有 `.active` 类
  - 步骤：读 HTML → 插入按钮 → 确认结构正确

- [x] 任务 2：CSS 实现三种模式布局 + 按钮样式
  - 编写 `.mode-btn` / `.mode-btn.active` 按钮样式
  - 编写 `.editor-mode-edit`：textarea 100%，预览区隐藏
  - 编写 `.editor-mode-split`：flex 行，textarea 和预览区各 50%，带分隔线
  - 编写 `.editor-mode-preview`：textarea 隐藏，预览区 100%
  - 将 `.editor-body` 改为 `display: flex; flex-direction: column`，子元素 `.editor-textarea` 和 `.md-rendered` 统一 `flex: 1; min-height: 0`
  - 步骤：读现有 CSS → 追加样式 → 确认不破坏现有布局

- [x] 任务 3：JS 实现模式切换逻辑 + 防抖渲染
  - 获取 3 个按钮的 DOM 引用
  - 编写 `switchEditorMode(mode)` 函数：
    - 更新 `editorOverlay.dataset.mode = mode`
    - 切换 3 个按钮的 `.active` 类
    - 如果是分栏/预览模式，立即渲染 `marked.parse(textarea.value)` 到预览区
  - 在 `onEditorInput` 中增加防抖渲染调用（300ms）
  - 步骤：读现有 JS → 追加切换函数 → 绑定事件 → 测试防抖渲染

- [x] 任务 4：构建验证
  - 运行 `npx vite build` 确认 0 errors
  - 运行 `golangci-lint fmt ./... && golangci-lint run ./...` 确认 0 issues

# 任务依赖
- 无（所有任务可以顺序执行）
