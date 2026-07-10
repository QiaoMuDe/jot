# 修复 AI 代码块复制按钮随水平滚动条移动的问题

## 现状分析

### 问题描述

AI 消息中的代码块有一个"复制"按钮，位于代码块右上角。当代码内容宽度超出代码块容器、出现水平滚动条时，拖动滚动条会导致复制按钮跟随代码内容水平移动，而不是固定在右上角原位。

### 根因

DOM 结构上，复制按钮 `.code-copy-btn` 是 `<pre>` 的子元素（通过 `pre.appendChild(copyBtn)` 追加），而 `<pre>` 本身：

- 有 `overflow-x: auto` — 内容超宽时可横向滚动
- 有 `position: relative` — 作为绝对定位的复制按钮的包含块

当 `<pre>` 发生水平滚动时，其内部的**所有子元素**（包括绝对定位的复制按钮）都会随滚动偏移移动。因此 `right: 8px` 的定位是相对 `<pre>` 的**内容 + 滚动偏移量**计算的，而非相对可视区域固定。

### 关键代码位置

| 文件 | 行号 | 说明 |
|------|------|------|
| `frontend/src/js/ai-chat.js` | L2362 | `pre.appendChild(copyBtn)` — 按钮追加到 `<pre>` 内部 |
| `frontend/src/js/ai-chat.js` | L2367-2377 | 有语言标签时才创建 `.pre-wrapper` 包裹器 |
| `frontend/src/css/components/ai-chat.css` | L175-183 | `.ai-msg-assistant pre` — 定义了 `overflow-x: auto` + `position: relative` |
| `frontend/src/css/components/ai-chat.css` | L222-225 | `.ai-msg-assistant .pre-wrapper` — 已有 `position: relative` |
| `frontend/src/css/components/ai-chat.css` | L237-276 | `.ai-msg-assistant .code-copy-btn` — 绝对定位样式 |
| `frontend/src/css/components/ai-chat.css` | L254 | 悬浮显现选择器 `.ai-msg-assistant pre:hover .code-copy-btn` |

### 当前 DOM 结构

**有语言标签的代码块**（也受影响）：
```html
<div class="pre-wrapper">     <!-- position: relative -->
  <pre>                       <!-- position: relative; overflow-x: auto ← 滚动容器 -->
    <code>...</code>
    <button class="code-copy-btn">复制</button>  <!-- 在 pre 内部，随滚动移动 -->
  </pre>
  <span class="code-lang-badge">JavaScript</span>
</div>
```

**无语言标签的代码块**（受影响，且无包裹器）：
```html
<pre>                       <!-- position: relative; overflow-x: auto ← 滚动容器 -->
  <code>...</code>
  <button class="code-copy-btn">复制</button>  <!-- 在 pre 内部，随滚动移动 -->
</pre>
```

## 修复方案

### 核心思路

将复制按钮移出 `<pre>`（滚动容器），放到 `.pre-wrapper` 内，使按钮的 `position: absolute` 基于不滚动的 `.pre-wrapper` 定位。

**始终创建 `.pre-wrapper`**，无论代码块是否有语言标签，统一 DOM 结构。

### 修改步骤

#### 1. JS 修改 — `frontend/src/js/ai-chat.js`（renderMarkdown 函数内）

**调整点**：

1. 将复制按钮的追加目标从 `pre.appendChild(copyBtn)` 改为后续追加到 `.pre-wrapper` 上
2. 将包装逻辑提前：**始终**创建 `.pre-wrapper`，不再只在有语言标签时才创建
3. 复制按钮追加到 `.pre-wrapper`（在 `<pre>` 外部），语言标签追加到 `.pre-wrapper`（在复制按钮之后）

**新 DOM 结构**（统一后，有无语言标签一致）：
```html
<div class="pre-wrapper">     <!-- position: relative -->
  <button class="code-copy-btn">复制</button>  <!-- 在 pre 外部，不随滚动移动 -->
  <pre>                       <!-- overflow-x: auto 滚动容器 -->
    <code>...</code>
  </pre>
  <span class="code-lang-badge">JavaScript</span>  <!-- 仅有时存在 -->
</div>
```

#### 2. CSS 修改 — `frontend/src/css/components/ai-chat.css`

| 修改项 | 当前 | 改为 |
|--------|------|------|
| 悬浮显现选择器 | `.ai-msg-assistant pre:hover .code-copy-btn` | `.ai-msg-assistant .pre-wrapper:hover .code-copy-btn` |
| `pre` 的 `position` | `position: relative`（L183） | 移除 `position: relative`（不再需要作为定位容器） |
| `.pre-wrapper` 样式 | 仅 `position: relative`（已有 L223-225） | 保持不变 |

#### 3. 影响范围

| 影响 | 说明 |
|------|------|
| 有语言标签的代码块 | 按钮移出 `<pre>` 后修复滚动问题；其他行为不变 |
| 无语言标签的代码块 | 新增 `.pre-wrapper` 包裹器，按钮移出 `<pre>` 后修复滚动问题 |
| 悬浮显隐行为 | `.pre-wrapper:hover` 触发按钮显示，和原来 `.pre:hover` 效果一致 |
| 单行代码块 | 逻辑不变，仅调整 DOM 层级 |
| 笔记预览模式 | 不动（存在相同问题但不在本次范围内） |

### 验证方式

1. 在 AI 对话中发送/接收含多行代码块的消息（内容宽度超过代码块宽度）
2. 出现水平滚动条后，拖动滚动条，观察复制按钮是否固定在右上角
3. 测试有语言标签和无语言标签两种场景
4. 测试单行代码块（`code-copy-btn--single` 类）
5. 测试复制按钮点击功能是否正常
6. 测试悬浮显隐效果是否正常
