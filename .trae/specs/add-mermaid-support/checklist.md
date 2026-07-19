# 检查清单

## 依赖
- [x] `mermaid` 依赖已安装，package.json 中包含 `mermaid` 条目（使用 `import mermaid from 'mermaid'` 默认导入）

## 主题映射（theme-config.js）
- [x] `theme-config.js` 导出 `isDarkTheme` 对象，覆盖全部 14 个主题
- [x] 每个主题映射到 `true`（暗色）或 `false`（亮色），不在映射表中时默认返回 `'default'`

## main.js 核心函数
- [x] 导入 `mermaid`（通过 `import mermaid from 'mermaid'` 默认导入）
- [x] 导入 `isDarkTheme`（来自 `theme-config.js`）
- [x] `initMermaid()` — 一次性初始化 mermaid 引擎，只调用一次
- [x] `getMermaidTheme()` — 根据 `data-theme` 查询 `isDarkTheme` 映射表，返回 `'dark'|'default'`
- [x] `setupMermaidBlock(pre)` — 为 Mermaid 代码块设置交互结构（.mermaid-rendered 容器 + 切换按钮），默认显示源码
  - [x] 用 `data-mermaid-processed` 标记防止重复处理
  - [x] 用 `data-mermaid-code` 属性存储原始源码
- [x] `renderSingleMermaid(pre)` — 渲染单个代码块，根据当前主题执行 `mermaid.render()`
  - [x] 成功：显示 SVG
  - [x] 失败：显示错误信息（红色文字），可切回源码
- [x] `toggleMermaidView(btn)` — 在渲染视图和源码视图之间切换，切回渲染时重新调用 `renderSingleMermaid`
- [x] `renderMermaidBlocks(container)` — 遍历容器中所有 `pre code.language-mermaid`，为每个调用 `setupMermaidBlock`

## 笔记预览集成
- [x] Worker onmessage 处理器在 `_applyPreviewDOMHelpers()` 后调用 `renderMermaidBlocks(els.mdRendered)`
- [x] 笔记预览中 Mermaid 代码块默认显示源码，不自动渲染
- [x] hljs 高亮逻辑跳过 `code.language-mermaid`，避免冲突

## AI 消息集成
- [x] `renderMermaidBlocks` 通过 `window.renderMermaidBlocks` 暴露给 AI 模块
- [x] `ai-chat.js` 的 `renderMarkdown()` 末尾调用 `window.renderMermaidBlocks(el)`
- [x] 流式渲染阶段不处理 Mermaid，仅在最终渲染时处理
- [x] hljs 高亮逻辑跳过 `code.language-mermaid`，避免冲突

## CSS 样式（editor.css）
- [x] `.mermaid-rendered` 容器样式（居中、溢出处理、背景）
- [x] `.mermaid-toggle` 切换按钮样式（与复制按钮风格一致，定位在右上角 `right: 72px` 复制按钮左侧）
- [x] `.mermaid-error` 错误提示样式（红色文字）
- [x] `.mermaid-toggle` 和 `.copy-code-btn` 在 hover 时正确显隐
- [x] `.pre-wrapper.has-mermaid` 的 hover 显隐规则

## 验证
- [ ] 笔记预览中 Mermaid 代码块默认显示源码，点击"渲染"后显示 SVG（需手动运行验证）
- [ ] AI 消息中 Mermaid 代码块默认显示源码，点击"渲染"后显示 SVG（需手动运行验证）
- [ ] 渲染失败时显示错误提示，并可切回源码（需手动运行验证）
- [ ] 切换主题后，再次点击渲染时使用新主题（需手动运行验证）
- [ ] 切换页面后返回，Mermaid 图表交互正常（需手动运行验证）
