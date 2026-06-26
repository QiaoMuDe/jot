# Tasks

## Task 依赖关系

- Task 1（HTML）先于 Task 2（JS）和 Task 3（CSS），因为 JS 和 CSS 依赖于新 DOM 结构
- Task 2 和 Task 3 可以并行（JS 函数查询新类名，CSS 样式定义新类名，两者独立）
- Task 4（功能性验证）在所有 Task 完成后执行

---

- [x] Task 1: 替换 index.html 中 MD 语法页面的所有卡片 HTML 结构
  - SubTask 1.1: 移除 `#viewMdRef` 内部旧 `.md-ref-content` 中的所有卡片 `.md-ref-card`
  - SubTask 1.2: 为 10 张卡片编写新的 HTML 结构（编辑器窗口风格源码面板 + 文档风格预览面板）
    - 卡片通用结构：`.md-ref-card > .md-ref-card-header(.md-ref-badge) + .md-ref-card-body(.md-ref-source-panel + .md-ref-preview-panel) + .md-ref-card-footer(.md-ref-card-footnote + .md-ref-try-btn)`
    - 源码面板结构：`.md-ref-source-panel > .md-ref-editor-bar(.md-ref-editor-dots(3×.md-ref-editor-dot) + .md-ref-editor-filename + .md-ref-editor-copy-btn) + pre > code.md-ref-source`
    - 预览面板结构：`.md-ref-preview-panel > .md-ref-preview`
  - SubTask 1.3: 保留 10 个 `<script type="text/plain" class="md-ref-source">` 标签中的源码内容不变
  - SubTask 1.4: 保留 10 个 `.md-ref-try-btn` 按钮及事件绑定逻辑（仅类名适配）
  - SubTask 1.5: 保留页面顶部 `view-header`（标题「MD 语法」+ 返回按钮）
  - **验证**：`npm run build` 无 HTML parse 错误

- [x] Task 2: 重写 main.js 中 MD 语法页面的 JS 逻辑
  - SubTask 2.1: 重写 `renderMdRefCards()` — 移除 `_mdRefRendered` 标志，每次进入视图都重新渲染
  - SubTask 2.2: 新增 `setupIntersectionObserver()` — 创建 IntersectionObserver 监听 `.md-ref-card` 入视口，添加 `.visible` class 触发动画，`rootMargin: '0px 0px -100px 0px'`
  - SubTask 2.3: 调整 `setupRefCopyButtons()` — 查询 `button.md-ref-editor-copy-btn` 而非旧的 `.md-ref-copy-btn`，复制逻辑不变
  - SubTask 2.4: 调整 `setupMdRefTryButtons()` — 适配新 HTML 结构，查询 `.md-ref-card-body` 下的 `.md-ref-source` 源码文本，核心逻辑不变
  - SubTask 2.5: 保留 `openMdRefTryEditor()` 不变
  - **验证**：所有卡片正确渲染，marked 解析 + hljs 高亮正常

- [x] Task 3: 重写 style.css 中 MD 语法页面的所有样式
  - SubTask 3.1: 移除所有旧 `.md-ref-*` 样式代码块（从 `.md-ref-content` 到 `.md-ref-try-btn:active` 全部移除，约 lines 4623-4925）
  - SubTask 3.2: 编写 Bento Grid 布局样式（`.md-ref-content` CSS Grid，特定卡片 `grid-column: span 2`）
  - SubTask 3.3: 编写卡片样式（`.md-ref-card` 圆角/阴影/背景/过渡）
  - SubTask 3.4: 编写编辑器标题栏样式（`.md-ref-editor-bar` 交通灯圆点/文件名）
  - SubTask 3.5: 编写源码面板样式（`pre` 深色背景/等宽字体/圆角）
  - SubTask 3.6: 编写预览面板样式（文档卡片白色背景/排版精致）
  - SubTask 3.7: 编写 badge 样式（`.md-ref-badge` 图标+文字组合）
  - SubTask 3.8: 编写复制按钮样式（`.md-ref-editor-copy-btn` 在标题栏右侧）
  - SubTask 3.9: 编写新「试试」按钮样式（`.md-ref-try-btn` 带箭头动画）
  - SubTask 3.10: 编写卡片 hover 动效（`translateY(-2px)` + 阴影加深）
  - SubTask 3.11: 编写交错入场动画（`.md-ref-card` 初始 opacity+translate，`.visible` 触发过渡，`transition-delay` 递增 60ms）
  - SubTask 3.12: 编写预览区的 Markdown 渲染样式（h1-h6/p/ul/ol/table/blockquote/code/pre/hr）
  - SubTask 3.13: 编写响应式断点（`<768px` 全单列）
  - **验证**：`npm run build` 无 CSS 编译错误

- [x] Task 4: 设计与交互验证
  - SubTask 4.1: 验证所有 10 张卡片正确渲染，源码面板 + 预览面板内容完整
  - SubTask 4.2: 验证 6 套主题切换后颜色正常（交通灯圆点颜色固定不变）
  - SubTask 4.3: 验证滚动入场动画正常（IntersectionObserver 触发）
  - SubTask 4.4: 验证 hover 动效（卡片上浮 + 阴影加深 + 按钮箭头动画）
  - SubTask 4.5: 验证复制功能正常运行（集成到标题栏）
  - SubTask 4.6: 验证「打开编辑器试试」功能正常
  - SubTask 4.7: 验证宽屏（>1200px）/窄屏（<768px）布局响应正确
  - SubTask 4.8: 验证 `npm run build` 构建通过
