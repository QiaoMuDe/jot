# Tasks

- [x] Task 1: 新增 TOC 侧栏 DOM 结构
  - [x] 在 `frontend/index.html` 的 `editor-panes` 内新增 `#tocSidebar` 侧栏元素
  - [x] 包含折叠按钮、标题"大纲"、目录列表容器 `#tocBody`
  - [x] 元素默认 `display:none`（仅在预览模式下显示）

- [x] Task 2: 修改预览布局为左右并排
  - [x] 在 `editor.css` 中新增 `editor-panes[data-preview-layout]` 样式（flex-direction: row）
  - [x] 修改 `editor-panes` 在预览模式下切换到并排布局的逻辑（`_setPreviewLayout()`）
  - [x] 确保非 `.md` 笔记不显示 TOC

- [x] Task 3: 实现标题提取逻辑（Web Worker 端）
  - [x] 在 `preview-worker.js` 中用 `marked.Lexer.lex()` 提取标题 tokens
  - [x] 通过 `postMessage` 将标题数组随 HTML 一起返回主线程
  - [x] 标题数据结构：`{ depth: number, text: string }`

- [x] Task 4: 实现标题锚点 ID 生成
  - [x] 在渲染后的 HTML 中为 `<h1>`~`<h6>` 元素添加唯一 `id` 属性
  - [x] ID 重复时自动追加数字后缀

- [x] Task 5: 实现 TOC 侧栏渲染与交互
  - [x] 根据标题数组生成层级缩进的 TOC 列表
  - [x] 点击目录项时平滑滚动预览区到对应标题位置
  - [x] 预览区滚动时自动高亮当前章节对应的目录项（scroll 事件防抖）
  - [x] 高亮项自动滚动到 TOC 可视区域内
  - [x] 空文档/无标题时隐藏 TOC

- [x] Task 6: 实现侧栏折叠/展开功能
  - [x] 点击折叠按钮收起侧栏（仅保留窄条触发按钮）
  - [x] 点击展开按钮恢复侧栏
  - [x] 折叠状态持久化到 `localStorage`

- [x] Task 7: 编写 TOC 侧栏 CSS 样式
  - [x] 侧栏宽度约 200px，带分割线（border-right）
  - [x] 标题项按层级缩进，hover/active 状态样式
  - [x] 深色/浅色主题适配（使用 CSS 变量）
  - [x] 折叠/展开过渡动画
  - [x] 小屏/非全屏时的自适应处理

# Task Dependencies

- Task 4 需要在 Task 3 之后（标题提取后才能生成锚点 ID）
- Task 5 依赖 Task 3、Task 4（需要标题数据和锚点 ID）
- Task 7 可与其他任务并行（纯样式，不依赖逻辑）
- Task 1 是最基础的 DOM 结构，应最先完成
- Task 2 修改布局后，Task 5 才能正确展示
