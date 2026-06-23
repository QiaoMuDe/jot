# Tasks

- [ ] Task 1: 语言标签与选择菜单 JS 逻辑 — 在 `updatePreview()` 中为每个 `<pre>` 添加语言标签和选择菜单
  - [ ] 1.1 遍历 `els.mdRendered.querySelectorAll('pre')`，跳过已有 `.code-lang-badge` 的
  - [ ] 1.2 读取 `<code>` 元素上的 hljs 语言类（如 `language-javascript`），若无则显示 "Auto"
  - [ ] 1.3 创建 `<span class="code-lang-badge">语言名</span>` 并 append 到 `<pre>` 底部
  - [ ] 1.4 创建 `<div class="code-lang-menu">` 作为语言选项容器，内含所有已注册语言列表项，默认隐藏
  - [ ] 1.5 点击 badge 切换菜单显示/隐藏；点击菜单项执行语言切换
  - [ ] 1.6 语言切换逻辑：获取 `<code>` 元素原始文本 → 清除 hljs 高亮 → `hljs.highlightElement(code, { language })` → 更新 badge 文字 → 关闭菜单
  - [ ] 1.7 点击菜单外部区域自动关闭菜单（全局 click 事件监听）

- [ ] Task 2: 语言标签与菜单 CSS 样式
  - [ ] 2.1 `.code-lang-badge` 样式（position absolute 于 pre 底部右侧、opacity 0 / pre:hover 显示、小字等宽字体、圆角背景）
  - [ ] 2.2 `.code-lang-menu` 样式（position absolute 从 badge 位置弹出、列表式布局、hover 高亮、弹簧动画过渡）

# Task Dependencies

- Task 2 依赖 Task 1（CSS 服务于 JS 生成的 DOM 结构）
