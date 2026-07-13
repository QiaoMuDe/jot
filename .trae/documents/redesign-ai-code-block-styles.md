# 重新设计 AI 消息代码块样式

## 摘要

对 AI 助手消息中的 Markdown 代码块进行全面的视觉重新设计，使其更精致、更具可读性，与 Jot 应用的温暖极简设计语言保持一致。

## 现状分析

### 当前样式（ai-chat.css 第174-296行）

| 元素 | 当前样式 | 问题 |
|------|---------|------|
| `pre` | `padding: 10px 12px; border-radius: 8px; bg: var(--input-bg);` | 内边距偏大，圆角硬编码 |
| `pre code` | `font-family: Consolas/Monaco/Courier New; font-size: inherit;` | 字体栈较传统 |
| `.code-copy-btn` | `top: 4px; right: 8px; opacity: 0 → 1 on hover;` | 纯文本按钮，视觉上简单 |
| `.code-lang-badge` | `top: 100%; right: 8px; mt: 4px; bg: var(--hover-bg);` | 位于 pre 外部下方，位置不理想 |

### 设计系统

Jot 主题为**温暖极简**风格（`:root` 默认主题）：
- 背景色: `#F7F5F0` 暖米白
- 强调色: `#D97706` 琥珀
- 边框: `#E5E0D8` 暖灰
- 字体: `system-ui, -apple-system, sans-serif`

## 设计方向

**方向选择：温暖工业风（Warm Industrial）**

在保留 Jot 温暖基调的基础上，为代码块注入工业感——清晰的边界、精致的细节、克制的装饰。灵感来自：印刷排版中的代码块呈现 + 现代代码编辑器（如 Zed、Warp）的干净视觉语言。

- 字体：跟随用户设置的 `font_family`，通过新增 `--font-mono` CSS 变量传递，代码块使用 `var(--font-mono)`，fallback 到 monospace
- 颜色：代码块背景略深于消息气泡，形成明确层次
- 圆角：统一使用设计系统的 `var(--radius-sm) = 6px`
- 动效：复制按钮 SVG 图标+文字，hover 状态增加精妙触感

## 变更方案

### 文件：`frontend/src/css/components/ai-chat.css`（第174-296行）

#### 1. `pre` 元素 — 更紧凑精致

```css
.ai-msg-assistant pre {
    background: var(--input-bg);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);  /* 6px, 与设计系统统一 */
    padding: 10px 14px;               /* 左右略增，上下不变 */
    margin: 0.5em 0;
    overflow-x: auto;
    overflow-y: hidden;
    font-size: 0.82em;
    line-height: 1.6;                 /* 略微增加行高，提高多行可读性 */
    scrollbar-width: thin;
    scrollbar-color: var(--scrollbar-thumb) transparent;
}
```

#### 2. `pre code` — 使用 `var(--font-mono)` 跟随用户字体设置

```css
.ai-msg-assistant pre code {
    background: none;
    padding: 0;
    border-radius: 0;
    font-family: var(--font-mono, 'Consolas', 'Monaco', 'Courier New', monospace);
    font-size: inherit;
    color: var(--text-primary);
}
```

#### 3. `.code-copy-btn` — 图标 + 更精致

- 位置微调：`top: 6px; right: 6px;`（与笔记预览对齐）
- 添加复制图标 SVG（使用内联 `<svg>`，通过 JS 注入或 CSS 伪元素）
- 背景：使用 `var(--card-bg)` 半透明，hover 时变为实色
- 添加微妙的 `backdrop-filter: blur(4px)` 效果
- 圆角使用 `var(--radius-sm)`
- 按钮 hover 时添加边框强调色变化

JS 侧（ai-chat.js）修改：在创建 `copyBtn` 时将纯文本按钮改为图标+文字。

```js
// 创建复制按钮 - 使用 SVG 图标
const copyBtn = document.createElement('button');
copyBtn.className = 'code-copy-btn' + (isSingleLine ? ' code-copy-btn--single' : '');
copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> 复制`;
copyBtn.title = '复制代码';
```

#### 4. `.code-copy-btn.copied` — 成功反馈优化

```css
.ai-msg-assistant .code-copy-btn.copied {
    color: var(--success);
    border-color: var(--success);
    opacity: 1;
}
.ai-msg-assistant .code-copy-btn.copied svg {
    color: var(--success);
}
```

并更新 JS：复制成功时替换图标为勾选图标，文字变为"已复制"。

```js
// 成功时替换 SVG 为勾选图标
copyBtn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg> 已复制`;
```

#### 5. `.code-lang-badge` — 位置优化，添加边框

```css
.ai-msg-assistant .code-lang-badge {
    position: absolute;
    top: 100%;
    right: 8px;
    margin-top: 4px;
    padding: 2px 8px;
    font-size: 0.7rem;
    font-family: var(--font-mono, 'Consolas', 'Monaco', 'Courier New', monospace);  /* 跟随用户字体 */
    line-height: 1.5;
    border-radius: 4px;
    background: var(--card-bg);
    border: 1px solid var(--border);
    color: var(--text-muted);
    opacity: 0;
    transition: opacity 0.15s ease;
    pointer-events: none;
    user-select: none;
}
```

#### 6. `.pre-wrapper` — 间距控制

```css
.ai-msg-assistant .pre-wrapper {
    position: relative;
    margin: 0.5em 0;              /* 将 margin 从 pre 移到 wrapper */
}
.ai-msg-assistant .pre-wrapper pre {
    margin: 0;                    /* pre 本身不再需要 margin */
}
.ai-msg-assistant .pre-wrapper + .pre-wrapper {
    margin-top: calc(0.5em - 1px); /* 相邻代码块间减少重复间距 */
}
```

#### 7. hljs 适配重置

```css
.ai-msg-assistant .hljs {
    background: transparent;
    padding: 0;
}
```

### 文件：`frontend/src/js/ai-chat.js`（第2566-2584行）

修改 `renderMarkdown` 中创建复制按钮的代码，从纯文本按钮改为图标+文字按钮，并增加成功/复位的图标切换。

### 文件：`frontend/src/css/variables.css`

在 `:root` 中添加 `--font-mono` 变量：

```css
/* 等宽字体 — 跟随用户 font_family 设置，通过 JS 动态注入 */
--font-mono: 'Consolas', 'Monaco', 'Courier New', monospace;
```

### 文件：`frontend/src/main.js`

修改 `applyFontFamily()`（第1306行），在设置 `--font-family` 的同时也设置 `--font-mono`：

```javascript
function applyFontFamily(fontFamily) {
    if (fontFamily) {
        document.documentElement.style.setProperty('--font-family', `${fontFamily}, system-ui, -apple-system, sans-serif`);
        document.documentElement.style.setProperty('--font-mono', `${fontFamily}, 'Consolas', 'Monaco', 'Courier New', monospace`);
        els.fontFamilyDisplay.textContent = fontFamily;
    } else {
        document.documentElement.style.removeProperty('--font-family');
        document.documentElement.style.removeProperty('--font-mono');
        els.fontFamilyDisplay.textContent = '系统默认';
    }
}
```

> 逻辑说明：用户选择的字体作为首选，代码块仍会尝试以该字体渲染。若用户选择的是等宽字体（如 Fira Code、JetBrains Mono），代码完美对齐；若不是（如 DM Sans），浏览器会用该字体渲染代码，用户既然选择"同一字体通吃"，那就是有意为之。

## 不变的内容

- 不修改笔记预览（`.md-rendered`）的代码块样式
- 不修改 highlight.js 主题文件
- 不修改代码块的 DOM 结构（pre-wrapper > pre > code 保持不变）
- 不修改行内代码（`.ai-msg-assistant code` 非 pre 内部的行内 code）样式

## 验证步骤

1. 检查多行代码块渲染：内边距、边框、圆角是否正确
2. 检查单行代码块渲染：复制按钮是否垂直居中
3. 检查复制功能：点击前后图标/文字切换，颜色变化
4. 检查语言标签：hover 时显示，位置正确
5. 检查深色主题：颜色对比度充足
6. 检查其他主题（light, nord 等）：样式一致
7. 检查相邻代码块间距：不重叠
8. 检查代码块水平滚动：复制按钮不随内容滚动


