# 修复按钮对齐与"已复制"宽度膨胀

## 总结

两个问题：笔记中 Mermaid 渲染按钮比复制按钮高约 1px（AI 消息正常），以及"已复制"状态时文字变长导致按钮戳到渲染按钮下方。

## 当前状态分析

### 问题 1：笔记中按钮 1px 偏移

**根因：DOM 结构不一致。**

**笔记预览（`_applyPreviewDOMHelpers`，[main.js L3627-3668](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)）**：

工作流分两步：
1. L3627-3651：遍历 `pre` 创建复制按钮，用 `pre.appendChild(btn)` 放入 `pre` **内部**
2. L3653-3668：遍历 `pre` 创建 `.pre-wrapper`，将 `pre` 包裹进去

最终 DOM：
```
.pre-wrapper (position: relative)      ← containing block
  ├── pre
  │   └── .copy-code-btn (absolute)    ← 在 pre 内部
  ├── .code-lang-badge
  └── .mermaid-toggle (absolute)       ← 在 pre-wrapper 内部（pre 外部）
```

虽然两个按钮都以 `.pre-wrapper` 为定位参考，但**复制按钮在 `pre` 内部**、**渲染按钮在 `pre` 外部**。`backdrop-filter: blur(4px)` 会为元素创建独立的 GPU 合成层，当两个按钮处于不同子树的合成层时，浏览器可能产生 1px 的子像素渲染差异。

**AI 消息（`renderMarkdown`，[ai-chat.js L2490-2507](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)）**：
```
.pre-wrapper (position: relative)
  ├── pre
  ├── .copy-code-btn    ← 直接子元素（pre 外部）
  └── .mermaid-toggle   ← 直接子元素（pre 外部）
```
两个按钮在同一层级，无此问题。

**修复方案**：将笔记中复制按钮从 `pre` 内部移到 `.pre-wrapper` 内部，与 AI 消息结构完全一致。

### 问题 2："已复制"时宽度膨胀

复制按钮内容：

| 状态 | 内容 | 大约宽度（px） |
|---|---|---|
| 正常 | SVG(16) + gap(4) + " 复制"(~28) + padding(16) + border(2) | **~66px** |
| 已复制 | SVG(16) + gap(4) + " 已复制"(~40) + padding(16) + border(2) | **~78px** |

当前 `min-width: 62px` 不足以容纳任何状态。"已复制"至少需要 78px，超出与 Mermaid 按钮的间距（72-6=66px），所以戳到渲染按钮下方。

## 变更方案

### 变更 1：统一 DOM 结构 — 复制按钮移到 `.pre-wrapper` 内

**文件**：[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)

重构 `_applyPreviewDOMHelpers()` 中的代码块处理逻辑，将步骤顺序调换为：
1. 先做 `.pre-wrapper` 包裹（将 `pre` 移到 wrapper 内）
2. 再添加复制按钮到 **`wrapper`** 而非 `pre`

具体修改：移除 L3650 的 `pre.appendChild(btn)`，改为 `wrapper.appendChild(btn)`。但需先处理包裹逻辑，因此需将函数体重新组织。

**为什么是 JS 而非 CSS 方案**：因为问题根源是 DOM 结构不同，CSS 无法跨越父子层级修复合成层差异。

### 变更 2：复制按钮"已复制"时隐藏渲染按钮

**文件**：[editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css)

用户选择"已复制时隐藏渲染按钮"方案。

添加 CSS 规则：
```css
/* 复制按钮"已复制"状态时隐藏 Mermaid 切换按钮 */
.pre-wrapper:has(.copy-code-btn.copied) .mermaid-toggle {
    opacity: 0 !important;
    pointer-events: none;
}
```

`:has()` 伪类选择器检测 `.pre-wrapper` 内是否存在 `.copy-code-btn.copied`，若存在则隐藏 `.mermaid-toggle`。

同时保留 `.copy-code-btn` 的 `min-width: 62px` 不变，Mermaid 按钮 `right: 72px` 不变。复制按钮膨胀时渲染按钮隐藏，互不干扰。

## 涉及文件

| 文件 | 变更 | 风险 |
|---|---|---|
| `frontend/src/main.js` | 重构 `_applyPreviewDOMHelpers` | 低 — 仅改变 append 目标，行为不变 |
| `frontend/src/css/components/editor.css` | 新增 `:has()` CSS 规则 | 极低 — 纯新增规则，不影响现有样式 |

## 验证步骤

1. `cd frontend && npx vite build` — 构建无错误
2. 运行应用，检查笔记预览中复制按钮和渲染按钮是否在同一水平线
3. 点击复制按钮，检查"已复制"状态下是否戳到渲染按钮
4. 确认 AI 消息中两个按钮同样对齐正常
