# Tasks

- [x] Task 1: 创建 Web Worker 文件 `preview-worker.js`
  - 新增 `frontend/src/preview-worker.js`
  - 在 Worker 内 `import { marked } from 'marked'` 导入
  - 监听主线程 `message`，接收 Markdown 内容
  - 调用 `marked.parse(content)` 解析为 HTML
  - 通过 `postMessage({ html })` 返回给主线程
  - 注明：Worker 内不做 hljs 高亮（hljs 需要 DOM 环境），hljs 在 Worker 返回后由主线程执行

- [x] Task 2: 修改 `main.js` — 实现内容哈希缓存
  - 新增 `_lastPreviewContent` 变量缓存上一次内容
  - `updatePreview()` 入口处比较 `content === _lastPreviewContent`，相同则直接 return
  - 不同则更新 `_lastPreviewContent`

- [x] Task 3: 修改 `main.js` — 集成 Web Worker 渲染
  - 初始化 Worker 实例：`new Worker(new URL('./preview-worker.js', import.meta.url), { type: 'module' })`
  - `initPreviewWorker()` 函数创建 Worker + 设置 onmessage
  - 重写 `updatePreview()`：
    - 先检查哈希缓存
    - 显示加载状态（`.md-rendered` 内添加 `加载中…` DOM）
    - `worker.postMessage(content)` 触发 Worker 解析
    - Worker `onmessage` 回调中：
      - `els.mdRendered.innerHTML = html`
      - 执行 `hljs.highlightElement()` 遍历代码块
      - 执行 `_applyPreviewDOMHelpers()`（复制按钮/语言标签/表格按钮）
      - 隐藏加载状态
    - 无 Worker 或 Worker 正忙时回退到主线程同步渲染

- [x] Task 4: 新增 CSS 加载状态样式
  - 在 `style.css` 中新增 `.md-rendered-loading` 类
  - 旋转圈动画（`@keyframes mdLoadingSpin`）+ 脉动动画（`@keyframes mdLoadingPulse`）
  - "加载中…" 文字水平居中 + 浅色文字 + 旋转圈指示器
  - Worker 返回后通过 remove() 移除

# Task Dependencies

- Task 4 与 Task 1-3 无依赖，可并行
- Task 2 与 Task 3 有顺序依赖（先实现哈希缓存，再集成 Worker）
