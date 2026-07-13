# 统一 AI 和预览代码块样式 + 定制代码块滚动条

## 摘要

解决 AI 消息代码块与笔记预览代码块视觉不一致的问题，统一背景色，并给所有代码块定制一条专属的细滚动条，使其靠近边框放置。

## 现状差异分析

### 背景色差异

| 主题 | AI 代码块 (`--input-bg`) | 预览代码块 (`--bg-secondary`) | 备注 |
|------|------------------------|---------------------------|------|
| 默认 | `#F3F1ED` | `#EDE9E0` | 预览略深 |
| dark | `#252525` | `#181818` | 预览更深 |
| light | `#F5F5F5` | `#f6f8fa` (硬编码) | 非常接近 |
| nord | `#E8ECF3` | `#E2E7EE` | 预览略深 |
| 其他主题 | 各主题独立值 | 各主题独立值 | 均有色差 |

### 滚动条差异

| 特性 | AI 代码块 | 预览代码块 |
|------|----------|-----------|
| 自定义 webkit | ✅ `.ai-msg-assistant pre::-webkit-scrollbar` | ❌ 无，继承全局 |
| 尺寸 | `6px` | 继承全局 `6px` |
| 受 `#mainContent` 影响 | ❌ 不受影响 | ✅ 受影响（thumb 被设为 transparent + 滚动淡出） |
| Firefox | `scrollbar-width: thin` | 无 |

### 设计方向

**选择：极简工业风（Minimal Industrial）**

统一后的代码块将采用：
- 背景统一为 `--bg-secondary`（比气泡/正文更 subdued，视觉层次更清晰）
- 专属 4px 细滚动条，紧贴 pre 底部/右侧边框放置
- 极简无按钮，hover 时才显现滑块

## 变更方案

### 文件：`frontend/src/css/components/editor.css`

#### 1. `pre` — 统一背景为 `--bg-secondary`，调整 padding-bottom 让滚动条靠边

```css
.md-rendered pre {
    position: relative;
    padding: 12px 16px 10px 16px;  /* 底部 padding 略减，让滚动条更靠近边框 */
    border-radius: var(--radius-sm);
    overflow-x: auto;
    overflow-y: hidden;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    min-height: 52px;
    scrollbar-width: thin;
    scrollbar-color: var(--scrollbar-thumb) transparent;
    font-size: 0.82em;
    line-height: 1.6;
}
```

> 额外增加 `scrollbar-width`、`scrollbar-color`、`font-size`、`line-height` 与 AI 对齐

#### 2. 移除 `[data-theme="light"] .md-rendered pre` 的硬编码 `#f6f8fa`

```css
[data-theme="light"] .md-rendered pre {
    background: var(--bg-secondary);  /* 保持变量而非硬编码 */
}
```

#### 3. 新增 `.md-rendered pre` 的 webkit 自定义滚动条（与 AI 一致）

```css
.md-rendered pre::-webkit-scrollbar {
    width: 4px;
    height: 4px;
}
.md-rendered pre::-webkit-scrollbar-track {
    background: transparent;
}
.md-rendered pre::-webkit-scrollbar-thumb {
    background: var(--scrollbar-thumb);
    border-radius: 2px;
}
.md-rendered pre::-webkit-scrollbar-thumb:hover {
    background: var(--scrollbar-thumb-hover);
}
.md-rendered pre::-webkit-scrollbar-corner {
    background: transparent;
}
.md-rendered pre::-webkit-scrollbar-button {
    display: none !important;
    width: 0 !important;
    height: 0 !important;
}
```

### 文件：`frontend/src/css/components/ai-chat.css`

#### 4. AI 代码块背景也统一为 `--bg-secondary`（与预览一致）

```css
.ai-msg-assistant pre {
    background: var(--bg-secondary);
    ...
}
```

#### 5. AI 代码块滚动条改 4px（与预览一致）

```css
.ai-msg-assistant pre::-webkit-scrollbar {
    width: 4px;
    height: 4px;
}
```

#### 6. AI 代码块底部 padding 略减，让滚动条靠近边框

```css
.ai-msg-assistant pre {
    ...
    padding: 10px 14px 8px 14px;  /* bottom 从 10px → 8px */
    ...
}
```

### 文件：`frontend/src/css/variables.css`

无需修改（`--bg-secondary` 已在所有主题中定义）。

## 不变的内容

- 不修改复制按钮、语言标签等其他样式（上次已对齐）
- 不修改代码块的字体 `var(--font-mono)`（上次已对齐）
- 不修改 `pre-wrapper` 间距控制
- 不修改 `.hljs` 重置样式

## 统一后效果

| 属性 | 统一后值 |
|------|---------|
| 背景 | `var(--bg-secondary)`（两者一致） |
| 滚动条尺寸 | `4px` × `4px`（两者一致，比全局 6px 更细） |
| 滚动条位置 | 通过减小 `padding-bottom` 让滑块更靠近边框 |
| 滚动条外观 | 半透明圆角滑块，hover 加深 |
| border-radius | `var(--radius-sm)`（两者一致） |
| line-height | `1.6`（两者一致） |

## 验证步骤

1. 检查 AI 代码块背景是否变为 `--bg-secondary`
2. 检查预览代码块背景是否统一使用 `--bg-secondary`（light 主题不再是 `#f6f8fa`）
3. 检查滚动条尺寸：4px，更细
4. 检查滚动条 hover：颜色加深
5. 检查滚动条是否更靠近边框位置
6. 检查多主题（dark、light、nord、tokyo-night）下背景和滚动条颜色正常
7. 检查水平滚动和垂直滚动（如果被触发）滚动条显示正常
