# 任务列表

- [ ] 任务 1: 安装 `mermaid` 依赖
  - 步骤 1.1: 运行 `npm install mermaid` 安装依赖
  - 注：使用 `mermaid/render` 子路径导入（`import mermaid from 'mermaid/render'`）以减小包体积
  - 注：Mermaid 渲染在主线程进行，不在 Worker 中处理

- [ ] 任务 2: 在 `theme-config.js` 中新增 `isDarkTheme` 明暗主题映射表
  - 步骤 2.1: 根据 14 个主题的视觉特性，标记每个主题为 `true`（暗色）或 `false`（亮色）
  - 步骤 2.2: 导出 `isDarkTheme` 对象
  - 步骤 2.3: 不在映射表中的主题默认返回 `'default'`（亮色主题回退策略）

- [ ] 任务 3: 实现 Mermaid 渲染核心函数（主线程）
  - 步骤 3.1: 在 `main.js` 中导入 `mermaid` 和 `isDarkTheme`
  - 步骤 3.2: 创建 `initMermaid()` 函数 — 一次性初始化 mermaid 引擎（`startOnLoad: false`, `securityLevel: 'loose'`），只需调用一次
  - 步骤 3.3: 创建 `getMermaidTheme()` 函数 — 根据当前 `data-theme` 查询 `isDarkTheme` 映射表，返回 `'dark'` 或 `'default'`
  - 步骤 3.4: 创建 `setupMermaidBlock(pre)` 函数 — 为单个 Mermaid 代码块设置交互结构：
    - 用 `.pre-wrapper` 包裹（与已有代码块结构一致）
    - 创建 `.mermaid-rendered` 容器（初始为空，默认隐藏）
    - 添加 `.mermaid-toggle` 按钮（初始文字"渲染"）
    - 用 `data-mermaid-code` 属性存储源码文本
    - 用 `data-mermaid-processed` 标记防止重复处理
  - 步骤 3.5: 创建 `renderSingleMermaid(pre)` 函数 — 渲染单个 Mermaid 代码块：
    - 从 `data-mermaid-code` 获取源码
    - 调用 `getMermaidTheme()` 获取当前主题
    - 调用 `mermaid.render(id, code)` 生成 SVG
    - 成功 → 插入 SVG 到 `.mermaid-rendered`，隐藏 `pre`
    - 失败 → 在 `.mermaid-rendered` 中显示红色错误提示
    - 按钮文字更新为"源码"
  - 步骤 3.6: 创建 `toggleMermaidView(btn)` 函数 — 切换渲染/源码视图：
    - 当前为源码视图 → 调用 `renderSingleMermaid(pre)` 重新渲染
    - 当前为渲染视图 → 隐藏 `.mermaid-rendered`，显示 `pre`
  - 步骤 3.7: 创建 `renderMermaidBlocks(container)` 函数 — 遍历容器中所有未处理的 `pre code.language-mermaid`，为每个调用 `setupMermaidBlock`
  - 步骤 3.8: 暴露 `renderMermaidBlocks` 到 `window` 供 AI 模块使用

- [ ] 任务 4: 在笔记预览中集成 Mermaid
  - 步骤 4.1: 在 Worker onmessage 处理器中 `_applyPreviewDOMHelpers()` 后调用 `renderMermaidBlocks(els.mdRendered)`
  - 步骤 4.2: 在 `_applyPreviewDOMHelpers()` 中 hljs 高亮逻辑跳过 `code.language-mermaid` 元素
  - 注：预览中不自动渲染 Mermaid，仅设置交互结构（默认显示源码）

- [ ] 任务 5: 在 AI 消息渲染中集成 Mermaid
  - 步骤 5.1: 在 `ai-chat.js` 的 `renderMarkdown()` 末尾调用 `window.renderMermaidBlocks(el)`
  - 步骤 5.2: 流式渲染阶段跳过 Mermaid 处理，仅在最终渲染时统一设置
  - 步骤 5.3: 在 `renderMarkdown()` 中 hljs 高亮逻辑跳过 `code.language-mermaid` 元素
  - 步骤 5.4: 通过 `window.renderMermaidBlocks` 共享函数（main.js 已暴露）

- [ ] 任务 6: 添加 Mermaid 相关 CSS 样式
  - 步骤 6.1: 在 `editor.css` 中新增 `.mermaid-rendered` 容器样式（居中、溢出处理、背景色）
  - 步骤 6.2: 新增 `.mermaid-toggle` 切换按钮样式（与复制按钮风格一致，定位在右上角 `right: 72px` 复制按钮左侧）
  - 步骤 6.3: 新增 `.mermaid-error` 错误提示样式（红色文字、适当内边距）
  - 步骤 6.4: 新增 `.pre-wrapper.has-mermaid` 的 hover 显隐规则（切换按钮与复制按钮协调显示）
  - 步骤 6.5: 通过 `mermaid.initialize({ theme: 'dark'|'default' })` 控制 SVG 主题适配

## 任务依赖关系

- 任务 2 无依赖
- 任务 1 无依赖
- 任务 3 依赖任务 1 和任务 2
- 任务 4 依赖任务 3（复用 `renderMermaidBlocks` 函数）
- 任务 5 依赖任务 3（复用 `renderMermaidBlocks` 函数和 `window` 暴露）
- 任务 6 无依赖，可与任务 3 并行

## 不需要实现的功能（已移除）

- ~~全局 `reRenderMermaidCharts()` 函数~~ — 按需渲染单个图表，不全局重绘
- ~~在 `applyTheme()` 中触发 Mermaid 重绘~~ — 主题切换不自动重绘
- ~~自动渲染 Mermaid 图表（页面加载/Markdown 渲染完成后）~~ — 需用户点击"渲染"按钮才触发
- ~~Worker 中处理 Mermaid 渲染~~ — 全部在主线程处理
