# 自实现 AI 对话组件（替换 QuikChat）Spec

## Why
当前 AI 对话基于 QuikChat 第三方组件，样式覆盖 200+ 行 CSS 仍不够自然。需要替换为自实现组件，完全由 Jot CSS 变量控制外观，并增加 Markdown 渲染和代码高亮。

## What Changes
- **REMOVED**: 移除 QuikChat npm 依赖（`quikchat: ^1.2.7`）
- **NEW**: 自实现 AI 对话组件（`marked` + `highlight.js` 渲染，CSS 变量主题原生适配）
- **MODIFIED**: `index.html` AI 对话页面结构（移除 QuikChat 挂载点，替换为自定义消息列表 + 输入区）

## Impact
- Affected specs: `add-ai-assistant`（已完成，本 spec 替换其实现）
- Affected code:
  - `frontend/package.json`：移除 `quikchat`，保留 `marked` + `highlight.js`
  - `frontend/index.html`：`#viewAiChat` 内 QuikChat 容器 → 自定义消息列表 + 输入区
  - `frontend/src/js/ai-chat.js`：**重写** — 移除 QuikChat 初始化，实现自渲染消息引擎
  - `frontend/src/css/components/ai-chat.css`：**重写** — 高品质 UI 设计（见下方设计规范）

## UI/UX 设计规范

### 设计方向
- **风格**: 精致细腻的极简主义（Refined Minimalism），与 Jot 现有卡片式设计语言统一
- **基调**: 安静、专注、沉浸感 — AI 对话是辅助工具，不应喧宾夺主
- **差异化**: 气泡入场微动效 + 输入区聚焦光晕 + 代码块优雅高亮

### 布局结构
```
┌──────────────────────────────────┐
│ view-header: ← 返回  AI 助手  清空 │
├──────────────────────────────────┤
│                                  │
│    ┌─────────────────────┐       │
│    │  用户消息右对齐气泡   │       │
│    └─────────────────────┘       │
│                                  │
│   ┌──────────────────────────┐   │
│   │  AI 回复左对齐气泡       │   │
│   │  · Markdown 渲染正文     │   │
│   │  · 代码块 highlight.js   │   │
│   │  · 表格/列表/任务列表     │   │
│   └──────────────────────────┘   │
│                                  │
│          ● ● ● (打字指示器)      │
│                                  │
│  ┌────────────────────────────┐  │
│  │ 输入框 (placeholder)  发送 │  │
│  └────────────────────────────┘  │
├──────────────────────────────────┤
│   底部（无附加上下文栏）          │
└──────────────────────────────────┘
```

### 消息气泡设计
- **用户消息** (`.ai-msg-user`)
  - 右对齐
  - 背景色: `var(--accent)`，纯色填充
  - 文字色: `#fff`（或 `var(--accent)` 对应的前景色）
  - 圆角: 16px 16px 4px 16px（右下角扁平）
  - 最大宽度: 75%
  - 内边距: 10px 16px
  - 字体: 0.875rem, line-height: 1.5
  - 阴影: 无（扁平化风格）
  - 入场动画: 从右滑入 + 透明度 0→1, 200ms ease-out

- **AI 回复** (`.ai-msg-assistant`)
  - 左对齐
  - 背景色: `var(--card-bg)` 或 `var(--hover-bg)`
  - 文字色: `var(--text-primary)`
  - 边框: 1px solid `var(--border)`（极淡描边）
  - 圆角: 16px 16px 16px 4px（左下角扁平）
  - 最大宽度: 82%
  - 内边距: 12px 16px
  - 阴影: 0 1px 3px `var(--shadow)`（极浅阴影，增加层次感）
  - 入场动画: 从下滑入 + 透明度 0→1, 250ms ease-out

### Markdown 渲染样式
沿用 Jot 笔记内容已有的 Markdown 渲染样式（`.note-content` 体系），确保一致性：
- **标题** (h1-h4): 增大字重，加宽间距
- **代码块** (`<pre><code>`): `var(--input-bg)` 底色, 1px `var(--border)` 边框, 8px 圆角, 10px 内边距, 横向溢出滚动
- **行内代码** (`<code>`): `var(--input-bg)` 底色, 2px 6px 内边距, 4px 圆角
- **表格**: 完整边框, `var(--hover-bg)` 表头底色, 8px 单元格内边距
- **引用**: 左侧 3px `var(--accent)` 竖线, `var(--hover-bg)` 底色
- **任务列表**: checkbox 使用 `var(--accent)` 选中色
- **链接**: `var(--accent)` 色, hover 下划线

### 代码高亮 (highlight.js)
- 代码块内的 `<code>` 应用 `hljs.highlightElement()` 自动高亮
- 选用与当前 Jot 代码高亮主题匹配的 hljs 主题（使用 `.hljs` 选择器覆盖）
- 代码块右上角添加语言标签（`.code-lang-badge`）— 复用已有的 `add-code-lang-badge` 样式
- 代码块可横向滚动（`overflow-x: auto`）

### 输入区设计
```
┌─────────────────────────────────────┐
│ [textarea]                    [发送] │
│  placeholder: "输入消息..."          │
│  支持 Shift+Enter 换行              │
│  Enter 发送                         │
└─────────────────────────────────────┘
```
- 整体容器: 1px solid `var(--border)` 顶部边框, 背景 `var(--bg)`, 12px 内边距
- Textarea:
  - 背景: `var(--input-bg)`
  - 边框: 1px solid `var(--border)`, focus → `var(--accent)`
  - 圆角: `var(--radius-md)`
  - 最小高度: 40px, 最大高度: 140px, auto-resize
  - 字体: 0.875rem, color: `var(--text-primary)`
  - placeholder: `var(--text-muted)`
  - Focus 时: 外发光 `0 0 0 2px color-mix(in srgb, var(--accent) 20%, transparent)`
- 发送按钮:
  - 圆形, 36×36px, `var(--accent)` 底色
  - SVG 发送图标（箭头）白色
  - Hover: opacity 0.85
  - Active: scale(0.92)
  - Disabled: opacity 0.4（输入为空时禁用）

### 打字指示器 (Typing Indicator)
```
● ● ●
```
- 左对齐（与 AI 消息同侧）
- 三圆点 bounce 动画
- 圆点: 7px, `var(--text-muted)` 色, 圆角 50%
- 间距: 4px
- 动画: `transform: translateY(0→-4px→0)` + `opacity: 0.4→1→0.4`
- 每个圆点延迟 0.2s

### 入场过渡动画
- 每条消息入场: `opacity: 0 → 1`, 加上 `transform: translateY(8px) → translateY(0)`
- 动画时长 200-250ms
- 每条消息依次延迟 30ms 触发
- 消息列表自动滚动到底部（`scrollTo` 平滑滚动）
- 列表容器的滚动条复用项目统一风格（6px 宽, `var(--border)` 色 thumb）

### 空状态设计
```
       [⚠ 图标 48px]
   尚未配置 AI 服务
  请在设置中配置 API 后再使用
       [前往设置]
```
- 垂直居中 flex 布局
- 图标: 48px, `var(--text-muted)` 色, opacity 0.4
- 标题: 1rem, 600 weight, `var(--text-primary)`
- 描述: 0.85rem, `var(--text-secondary)`
- 按钮: 复用 `.btn-save` 样式
- 配置检测逻辑: 进入页面时调 `GetAIConfig()`，无 `api_key` 或空值则显示空态

### 尺寸与间距规范
- 对话区域 padding: 16px 20px
- 消息间垂直间隔: 12px
- 气泡与输入区间隔: 12px
- view-header 复用统一样式
- 消息列表区域占 flex:1，输入区在底部固定

### 无障碍与交互增强
- 发送按钮在输入为空时 `disabled`
- 输入框 focus 状态明亮可见
- 清空对话前 `confirm()` 确认，防止误操作
- 错误响应在气泡中显示（红色 tint），可再次发送重试
- 滚动条不占用消息宽度（`overflow: overlay` 或预留 padding）

## ADDED Requirements

### Requirement: 自实现 AI 对话组件
系统 SHALL 使用 `marked` + `highlight.js` + 原生 DOM 渲染替代 QuikChat，完全由 Jot CSS 变量控制外观。后端调用复用现有 `CallAI`（非流式）。

#### Scenario: 消息结构
- **WHEN** AI 对话页面加载且已配置
- **THEN** 页面显示自定义消息列表（`#aiChatMessages`）+ 输入区（`#aiChatInput` + `#aiChatSendBtn`）
- **AND** 用户消息以 `.ai-msg-user` 气泡右对齐显示（accent 色填充）
- **AND** AI 回复以 `.ai-msg-assistant` 气泡左对齐显示（surface 色 + 边框）

#### Scenario: Markdown 渲染 + 代码高亮
- **WHEN** AI 回复内容包含 Markdown
- **THEN** 使用 `marked.parse()` 渲染为 HTML
- **AND** 代码块使用 `highlight.js` 高亮（自动检测语言），右上角显示语言标签
- **AND** 支持 GFM 表格、列表、任务列表

#### Scenario: 发送与响应
- **WHEN** 用户在输入框中输入内容并点击发送或按 Enter
- **THEN** 用户消息添加到列表中（带入场动画：右滑入 + 淡入）
- **AND** 显示打字指示器（三圆点 bounce）
- **AND** 调用 `window.go.main.App.CallAI(messages)` 获取回复
- **AND** 收到回复后隐藏打字指示器，渲染 AI 消息（带入场动画：上滑入 + 淡入）
- **AND** 调用失败时在气泡中显示错误信息（红色 tint）

#### Scenario: 清空对话
- **WHEN** 用户点击「清空对话」按钮
- **THEN** 显示确认弹窗
- **AND** 确认后清空消息列表 DOM

#### Scenario: 未配置状态
- **WHEN** AI 对话页面加载但尚未配置 AI（缺少 API Key 或 Base URL）
- **THEN** 页面显示空状态提示 + 「前往设置」按钮

## REMOVED Requirements
### Requirement: QuikChat 组件
**Reason**: 被自实现组件替代
**Migration**: 移除 `quikchat` npm 包，`ai-chat.js` 重写为原生 DOM 渲染
