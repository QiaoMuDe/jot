# 笔记预览代码块样式优化

## 摘要

将笔记预览（`.md-rendered`）的代码块样式与 AI 消息代码块对齐，包括字体跟随、复制按钮图标化、hover 增强、间距控制等细节优化。

## 变更方案

### 文件：`frontend/src/css/components/editor.css`

#### 1. `pre code` — 字体改为 `var(--font-mono)`

第 1009 行（`code` 行内代码样式）和第 1030 行（`pre code` 块级代码样式）中，将硬编码的 `'Consolas', 'Monaco', 'Courier New', monospace` 改为 `var(--font-mono, ...)`。

```css
/* 行内 code */
.md-rendered code {
    font-family: var(--font-mono, 'Consolas', 'Monaco', 'Courier New', monospace);
    ...
}

/* pre code */
.md-rendered pre code {
    ... /* font-family 从缺少改为继承 */
    font-family: var(--font-mono, 'Consolas', 'Monaco', 'Courier New', monospace);
}
```

#### 2. 复制按钮 `.copy-code-btn` — 增强 hover 效果

```css
.copy-code-btn:hover {
    background: var(--hover-bg);
    color: var(--text-primary);
    border-color: var(--accent);  /* 新增 */
    transform: scale(1.08);
}
```

#### 3. `.pre-wrapper` — 间距控制（与 AI 一致）

```css
.pre-wrapper {
    position: relative;
    margin: 1.2em 0;               /* 将 margin 从 pre 移到 wrapper */
}
.pre-wrapper pre {
    margin: 0;                     /* pre 本身不再需要 margin */
}
.pre-wrapper + .pre-wrapper {
    margin-top: calc(1.2em - 1px); /* 相邻代码块间减少重复间距 */
}
```

同时将 `pre` 的 `margin: 1.2em 0` 移除。

#### 4. 复制按钮 — 添加毛玻璃背景

```css
.copy-code-btn {
    ...
    background: color-mix(in srgb, var(--card-bg) 85%, transparent);
    backdrop-filter: blur(4px);
    ...
}
```

### 文件：`frontend/src/main.js`

#### 5. 复制按钮默认态添加 SVG 图标

`_applyPreviewDOMHelpers()` 函数中，复制按钮初始创建时改为图标+文字：

```javascript
// 默认状态带 SVG 图标
btn.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> 复制`;
```

重置时也恢复带图标的 HTML。

## 不变的内容

- 不修改代码块背景色（保留现有的 `var(--bg-secondary)` 和深色/亮色主题覆盖）
- 不修改 `pre` 的 `padding`（12px 16px 保持现状）
- 不修改代码块的 `border-radius`（保留 `var(--radius-sm)`）
- 不修改语言标签（已较完善）
- 不修改复制按钮位置（保留在 pre 内部，不改为 AI 的 wrapper 模式——降低改动风险）

## 验证步骤

1. 检查代码块字体是否正确跟随用户设置的 `font_family`
2. 检查复制按钮默认态：显示剪贴板 SVG 图标 + "复制" 文字
3. 检查复制按钮 hover：边框变为 accent 色
4. 检查复制按钮毛玻璃效果：半透明背景 + backdrop-filter
5. 检查复制成功：图标切换为勾选 + "已复制"
6. 检查相邻代码块间距：不重叠
7. 检查深色/亮色主题：颜色正常
