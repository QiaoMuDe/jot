# 修复 Mermaid 按钮对齐与 hover 问题

## 总结

修复 Mermaid 渲染按钮与复制按钮之间的三个 CSS/交互问题：垂直高度不对齐、"已复制"状态下宽度膨胀戳到相邻按钮、笔记预览中 hover 渲染按钮时复制按钮消失。

## 当前状态分析

### 问题 1：按钮垂直高度不齐

[editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css) 中两个按钮的 `top` 值不一致：

| 按钮 | CSS 选择器 | top 值 | 行号 |
|---|---|---|---|
| `.copy-code-btn` | `.copy-code-btn` | `6px` | L397 |
| `.copy-code-btn--single` | `.copy-code-btn--single` | `50%`（translateY(-50%)） | L442 |
| `.mermaid-toggle` | `.pre-wrapper.has-mermaid .mermaid-toggle` | `8px` | L1327 |

Mermaid 按钮比复制按钮低了 2px。

### 问题 2："已复制"后宽度膨胀

复制按钮 CSS（L395-414）：`padding: 2px 8px`，无 `min-width` 或固定宽度。

JS 中的内容切换：
- [main.js L3633](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)：`SVGS.copy + ' 复制'` → `SVGS.checkmark + ' 已复制'`（L3640）
- [ai-chat.js L2491](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)：14x14 SVG + `' 复制'` → 14x14 SVG + `' 已复制'`（L2497）

"已复制"比"复制"多一个字，宽度增加约 14-18px。复制按钮定位 `right: 6px`，Mermaid 按钮定位 `right: 72px`，间距仅 66px。膨胀后复制按钮向左延伸，戳到 Mermaid 按钮下方。

### 问题 3：笔记中 hover Mermaid 按钮时复制按钮消失

**DOM 结构差异**：

笔记预览（`_applyPreviewDOMHelpers`，[main.js L3627-3651](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js)）：
- 复制按钮通过 `pre.appendChild(btn)` 放在 `pre` 内部
- Mermaid 切换按钮通过 `wrapper.appendChild(toggleBtn)` 放在 `.pre-wrapper` 内部（`pre` 外部）

AI 消息（`renderMarkdown`，[ai-chat.js L2487-2509](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js)）：
- 复制按钮通过 `wrapper.appendChild(copyBtn)` 放在 `.pre-wrapper` 内部（`pre` 外部）
- Mermaid 切换按钮同样在 `.pre-wrapper` 内部

**hover CSS 规则**：

| 来源 | 规则 | 行号 |
|---|---|---|
| 复制按钮基本 hover | `:is(.md-rendered, .ai-msg-assistant) pre:hover .copy-code-btn` | L423-425 |
| AI 额外 hover（兜底） | `.ai-msg-assistant .pre-wrapper:hover .copy-code-btn` | L428-430 |
| AI 额外 hover（兜底） | `.ai-msg-assistant .copy-code-btn:hover` | L428-430 |
| Mermaid 按钮 hover | `.pre-wrapper.has-mermaid:hover .mermaid-toggle` | L1344-1347 |

**问题**：笔记（.md-rendered）缺少 `.pre-wrapper:hover .copy-code-btn` 兜底规则。当鼠标移到 `.mermaid-toggle`（在 `pre` 外部）时，`pre:hover` 不成立，复制按钮隐藏。AI 消息有兜底规则所以正常。

## 变更方案

仅在 [editor.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/editor.css) 中修改，不涉及 JS。

### 变更 1：统一按钮 top 值（修复问题 1）

将 `.copy-code-btn` 和 `.mermaid-toggle` 的 `top` 统一为一致的值。

- `.copy-code-btn`：`top: 6px` → `top: 8px`（与 Mermaid 按钮一致）
  - 理由：Mermaid 按钮 `padding: 4px 10px` 比复制按钮 `padding: 2px 8px` 更高，视觉上以 `top: 8px` 对齐更自然
- `.copy-code-btn--single`：`top: 50%` 保持不变（单行场景单独处理，不与 Mermaid 并排）

### 变更 2：修复"已复制"宽度膨胀（修复问题 2）

给 `.copy-code-btn.copied` 增加 `min-width`，或在非 copie d 状态也预留足够宽度。

方案：在 `.copy-code-btn` 上设置 `min-width`，使其在"复制"状态下就预留"已复制"的宽度。或者更简单——使用相同的 SVG + 文字组合，但通过 `text-align: center` 确保宽度不跳动。

推荐方案：给 `.copy-code-btn` 加 `min-width: 72px`，文字 `text-align: center`。这样"复制"和"已复制"状态宽度一致，不会戳到左侧的 Mermaid 按钮。

计算依据：Mermaid 按钮 `right: 72px`，复制按钮 `right: 6px`，可用空间为 66px。设置 `min-width: 66px` 或稍小值 `62px`（留安全边距）。

### 变更 3：补充笔记 hover 兜底规则（修复问题 3）

在 L428-431 的 AI hover 规则后面，增加等效的笔记规则：

```css
/* 笔记代码块：补齐 .pre-wrapper:hover 兜底（与 AI 消息保持一致） */
.md-rendered .pre-wrapper:hover .copy-code-btn,
.md-rendered .copy-code-btn:hover {
    opacity: 1;
}
```

同时由于 `.pre-wrapper:hover` 已经能兜底，还需要确保 Mermaid 代码块中复制按钮的 hover 触发条件完整。

注意：这条规则也可以写进已有的 `:is(.md-rendered, .ai-msg-assistant)` 选择器中，但为了可读性和维护性，统一成一条带 `:is()` 的合并规则更好。

建议将 L423-431 重构为：

```css
/* 代码块复制按钮 hover 显示：pre hover 直接触发 */
:is(.md-rendered, .ai-msg-assistant) pre:hover .copy-code-btn {
    opacity: 1;
}

/* 代码块复制按钮 hover 兜底：.pre-wrapper hover 或按钮自身 hover
   覆盖笔记 (复制按钮在 pre 内部) 和 AI (复制按钮在 pre-wrapper 内部) 两种 DOM 结构 */
:is(.md-rendered, .ai-msg-assistant) :is(.pre-wrapper:hover, .copy-code-btn:hover) .copy-code-btn {
    opacity: 1;
}
```

但这样 `.copy-code-btn:hover .copy-code-btn` 选择器不会匹配自身（父级 `.copy-code-btn:hover` 下的 `.copy-code-btn` 子元素不存在）。所以需要分开写：

```css
:is(.md-rendered, .ai-msg-assistant) .pre-wrapper:hover .copy-code-btn,
:is(.md-rendered, .ai-msg-assistant) .copy-code-btn:hover {
    opacity: 1;
}
```

这样一条规则同时覆盖笔记和 AI 两种场景。

## 涉及文件

仅一个文件变更：

- `d:\峡谷\Dev\本地项目\jot\frontend\src\css\components\editor.css`

无 JS 变更。

## 具体修改

### 修改 1 — 统一 top 值（L397）

```css
/* 修改前 */
.copy-code-btn {
    position: absolute;
    top: 6px;          /* ← 改这里 */

/* 修改后 */
.copy-code-btn {
    position: absolute;
    top: 8px;
```

### 修改 2 — 修复宽度膨胀（L395-414 加 min-width）

```css
/* 修改前 */
.copy-code-btn {
    /* ... */
    display: inline-flex;
    align-items: center;
    gap: 4px;
}

/* 修改后 */
.copy-code-btn {
    /* ... */
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    min-width: 62px;
    text-align: center;
}
```

### 修改 3 — 合并 hover 兜底规则（L423-431）

```css
/* 修改前（两条分离的规则） */
:is(.md-rendered, .ai-msg-assistant) pre:hover .copy-code-btn {
    opacity: 1;
}

/* AI 消息代码块：复制按钮在 .pre-wrapper 内、pre 外部，需额外 hover 规则 */
.ai-msg-assistant .pre-wrapper:hover .copy-code-btn,
.ai-msg-assistant .copy-code-btn:hover {
    opacity: 1;
}

/* 修改后（一条统一规则覆盖两种场景） */
:is(.md-rendered, .ai-msg-assistant) pre:hover .copy-code-btn {
    opacity: 1;
}

:is(.md-rendered, .ai-msg-assistant) .pre-wrapper:hover .copy-code-btn,
:is(.md-rendered, .ai-msg-assistant) .copy-code-btn:hover {
    opacity: 1;
}
```

## 验证步骤

1. `cd frontend && npx vite build` — 确保构建无错误
2. 运行应用，检查笔记预览中：
   - 复制按钮和 Mermaid 渲染按钮在同一水平线
   - 点击复制后"已复制"宽度不膨胀戳到渲染按钮
   - hover Mermaid 渲染按钮时复制按钮保持可见
3. 运行应用，检查 AI 消息中同样场景正常（回归测试）
