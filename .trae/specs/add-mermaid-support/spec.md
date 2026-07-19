# Mermaid 图表渲染支持

## 为什么

笔记和 AI 消息中的 ` ```mermaid ` 代码块目前仅作为纯文本展示，无法渲染为可视化图表。需要支持 Mermaid 图表的渲染，并能在渲染视图和源码视图之间切换。

## 变更内容

- 新增 `mermaid` 依赖（通过 `import mermaid from 'mermaid'` 默认导入）
- `main.js` 中实现 Mermaid 渲染核心函数（主线程渲染，因 mermaid/render 需要 DOM 环境）
- 笔记预览和 AI 消息中，Mermaid 代码块默认显示源码，用户点击"渲染"按钮后渲染为 SVG
- `theme-config.js` 中增加 `isDarkTheme` 明暗主题映射表
- 每次点击渲染时，根据当前系统主题和映射表确定 Mermaid 主题（dark/default），按需渲染单个图表
- CSS 新增 `.mermaid-toggle` 切换按钮样式 + `.mermaid-rendered` 容器样式 + `.mermaid-error` 错误提示样式
- 现有 `.pre-wrapper` 结构扩展：Mermaid 代码块内部增加渲染区域和切换按钮
- 不涉及 `preview-worker.js` 的变更（Mermaid 渲染在主线程进行）

## 影响范围

- 前端依赖：新增 `mermaid` 包
- 影响文件：`main.js`、`ai-chat.js`、`theme-config.js`、`editor.css`
- 影响功能：笔记预览渲染、AI 消息渲染
- 不涉及后端变更
- 主题切换时**不需要**全局重新渲染 Mermaid 图表（下次点击渲染时自动使用当前主题）

## 核心设计原则

### 按需渲染，不自动执行

Mermaid 图表不在页面加载或 Markdown 渲染完成后自动渲染。仅在用户点击"渲染"按钮时触发单次渲染。这避免了不必要的性能开销。

### 无全局重绘

主题切换时不需要重新渲染任何已渲染的 Mermaid 图表。用户下次点击切换按钮时，根据当前主题重新渲染即可。

### 函数职责单一

- `initMermaid()` — 一次性初始化 mermaid 引擎（仅需调用一次）
- `getMermaidTheme()` — 根据当前系统主题返回 `'dark'` 或 `'default'`
- `setupMermaidBlock(mermaidCode, pre)` — 为 Mermaid 代码块设置交互结构（.mermaid-rendered 容器 + 切换按钮），默认显示源码
- `renderSingleMermaid(pre)` — 渲染单个代码块：提取源码 → 调用 `mermaid.render()` → 替换 SVG 或显示错误
- `toggleMermaidView(btn)` — 在渲染视图和源码视图之间切换

### mermaid 导入方式

使用默认导入（mermaid v11 不再支持子路径导入）：

```js
import mermaid from 'mermaid';
```

## 新增需求

### 需求：Mermaid 图表格交互

系统须支持在笔记预览和 AI 消息中渲染 ` ```mermaid ` 代码块。

#### 场景：默认显示源码
- **给定** 笔记内容或 AI 消息包含 ` ```mermaid ` 代码块
- **当** Markdown 渲染完成
- **则** 代码块默认以源码形式显示（不会自动渲染），右上角显示"渲染"按钮，右下角显示语言标签 "Mermaid"

#### 场景：成功渲染
- **给定** 一个显示着 Mermaid 源码的代码块
- **当** 用户点击"渲染"按钮
- **则** 调用 `mermaid.render()` 生成 SVG，隐藏源码，显示渲染后的 SVG 图表，原"渲染"按钮变为"切换"按钮（在渲染图和源码之间切换）

#### 场景：渲染失败
- **给定** 一个包含无效 Mermaid 语法的代码块
- **当** 用户点击"渲染"按钮且 `mermaid.render()` 抛出异常
- **则** 不显示 SVG，在代码块内显示错误信息（红色文字提示语法错误），同时保留"显示源码"按钮方便用户切换回源码视图排查

#### 场景：渲染视图与源码视图切换
- **给定** 一个已渲染（或渲染失败）的 Mermaid 代码块
- **当** 用户点击切换按钮
- **则** 在渲染后的 SVG（或错误信息）和原始 ` ```mermaid ` 源码之间切换显示

#### 场景：再次点击渲染时使用当前主题
- **给定** Mermaid 图表处于源码视图或渲染视图
- **当** 用户点击切换/渲染按钮触发重新渲染
- **则** 根据当前系统主题对应的 Mermaid 主题（dark/default）重新渲染

### 需求：明暗主题映射

系统须提供 `isDarkTheme` 映射表，用于标记每个主题的明暗属性。

#### 场景：映射表定义
- **给定** `theme-config.js`
- **当** 导出 `isDarkTheme` 对象
- **则** 每个主题名称映射到 `true`（暗色）或 `false`（亮色），具体映射关系如下：

```js
export const isDarkTheme = {
    'default': false,          // 默认（亮色）
    'light': false,            // 浅色
    'dark': true,              // 深色
    'catppuccin-latte': false, // 暖咖
    'nord': false,             // 北极
    'gruvbox-light': false,    // 旧纸
    'one-dark-pro': true,      // 暗夜
    'quiet-light': false,      // 静谧
    'ysgrifennwr': false,      // 暖笺
    'tokyo-night': true,       // 夜幕
    'eye-protection': false,   // 护眼
    'dracula': true,           // 德古拉
    'alice': false,            // 爱丽丝
    'lightmind': false,        // 山林
};
```

#### 场景：回退策略
- **给定** 系统主题不在 `isDarkTheme` 映射表中（如未来新增主题）
- **当** `getMermaidTheme()` 查询不到对应值
- **则** 回退返回 `'default'`（亮色主题）

## 已修改需求

### 需求：Mermaid 渲染核心函数（主线程）

由于 `mermaid` 库需要 DOM 环境，渲染在主线程进行，不在 Worker 中处理。

`main.js` 中实现以下函数：

- `initMermaid()` — 一次性初始化 mermaid 引擎，设置 `startOnLoad: false`、`securityLevel: 'loose'`。**只需要调用一次**，后续渲染不需要重新初始化。
- `getMermaidTheme()` — 根据 `document.documentElement.getAttribute('data-theme')` 查询 `isDarkTheme` 映射表，返回 `'dark'` 或 `'default'`；查询不到时默认返回 `'default'`。
- `setupMermaidBlock(pre)` — **为单个 Mermaid 代码块设置交互结构**：
  - 查找 `pre code.language-mermaid`
  - 在 `pre` 外部创建 `.pre-wrapper` 容器（与已有逻辑一致）
  - 在 `.pre-wrapper` 内创建 `.mermaid-rendered` 容器（初始为空，默认隐藏）
  - 添加 `.mermaid-toggle` 切换按钮（初始文字为"渲染"）
  - 在 `pre` 上用 `data-mermaid-code` 属性存储原始 Mermaid 源码文本
  - 使用 `data-mermaid-processed` 标记防止重复处理
  - 为切换按钮绑定点击事件处理
- `renderSingleMermaid(pre)` — **渲染单个 Mermaid 代码块**：
  - 从 `data-mermaid-code` 获取源码
  - 调用 `getMermaidTheme()` 获取当前主题
  - 调用 `mermaid.render(id, code)` 生成 SVG
  - 成功：将 SVG 插入 `.mermaid-rendered` 容器，隐藏 `pre`，显示 `.mermaid-rendered`，更新按钮文字为"源码"
  - 失败：在 `.mermaid-rendered` 中显示错误提示（红色文字），隐藏 `pre`，显示 `.mermaid-rendered`，更新按钮文字为"源码"
- `toggleMermaidView(btn)` — 在 `.mermaid-rendered` 和 `pre` 之间切换显示。如果当前是源码视图，调用 `renderSingleMermaid` 重新渲染（确保使用最新主题）。
- `renderMermaidBlocks(container)` — 遍历容器中所有未处理的 `pre code.language-mermaid`，为每个调用 `setupMermaidBlock`。

### 需求：Mermaid 在笔记预览中的集成

笔记预览通过 Worker 获取 HTML 后再在主线程做 DOM 后处理。Mermaid 的处理在这里不自动渲染，只设置交互结构。

- **WHEN** Worker onmessage 处理器调用 `_applyPreviewDOMHelpers()` 之后
- **THEN** 调用 `renderMermaidBlocks(els.mdRendered)`，为每个 Mermaid 代码块设置交互结构（默认显示源码，不自动渲染）

### 需求：Mermaid 在 AI 消息中的集成

- **WHEN** `ai-chat.js` 的 `renderMarkdown()` 执行到最后
- **THEN** 调用 `window.renderMermaidBlocks(el)`，为 AI 消息中的 Mermaid 代码块设置交互结构
- **注意**：流式渲染阶段不处理 Mermaid，仅在最终渲染时调用一次

### 需求：hljs 跳过 language-mermaid

`hljs.highlightElement()` 不能处理 `language-mermaid` 代码块。在 `_applyPreviewDOMHelpers()` 和 `renderMarkdown()` 中的代码高亮逻辑须跳过 `code.language-mermaid` 元素，避免 hljs 报错或产生异常输出。

## 已移除需求

### ~~主题切换时重新渲染所有 Mermaid 图表~~

不再需要全局 `reRenderMermaidCharts()` 函数和在 `applyTheme()` 中调用。用户每次点击渲染按钮时，`renderSingleMermaid()` 会根据当前最新主题重新渲染单个图表，按需更新即可。

## 场景流程汇总

```
用户打开笔记 → Markdown 渲染完成
  └─ renderMermaidBlocks() → 为每个 ```mermaid 代码块设置结构（默认显示源码）
      └─ 用户点击"渲染"按钮
          └─ renderSingleMermaid(pre)
              ├─ 成功 → 显示 SVG
              └─ 失败 → 显示错误提示

用户切换主题 → applyTheme() 更新 data-theme（不触发 Mermaid 重绘）

用户再次点击切换按钮（从源码切回渲染图）
  └─ renderSingleMermaid(pre) → 使用当前最新主题重新渲染
```
