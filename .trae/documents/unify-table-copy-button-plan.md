# 统一表格复制按钮与代码块复制按钮样式

## Summary

让笔记预览和 AI 消息中的表格复制按钮使用与代码块复制按钮相同的 UI 风格（SVG 图标 + 毛玻璃 `backdrop-filter` + `min-width` 定宽 + 统一状态反馈），并确保按钮锚定在 HTML 结构中的最后一个 `<th>` 上（而非表格可视右边缘）。

## Current State

### 两个表格复制按钮 + 一个代码块复制按钮

| | 笔记预览（main.js） | AI 消息（ai-chat.js） | 代码块（main.js） |
|---|---|---|---|
| **类名** | `copy-table-btn` | `table-copy-btn` | `copy-code-btn` |
| **DOM 位置** | `div.table-wrapper` 内 | `th:last-child` 内 | `div.pre-wrapper` 内 |
| **SVG 图标** | ✅ 成功/失败有 SVG，初始无 | ❌ Unicode 字符 | ✅ 全部三态有 SVG |
| **背景** | `var(--card-bg)` 纯色 | `var(--card-bg)` 纯色 | `color-mix + blur(4px)` 毛玻璃 |
| **min-width** | ❌ 无 | ❌ 无 | ✅ `min-width: 62px` |
| **定位基准** | `.table-wrapper` 右边缘 | 最后一个 `<th>` | `.pre-wrapper` 右上角 |

### 关键问题

1. **预览模式按钮不在 `<th>` 内** — 按钮放在 `.table-wrapper` 里，`right: 6px` 锚定到整个表格的右边缘。表格有水平滚动时，按钮可能位于可视区域之外（屏幕右侧滚出去了）。用户要求锚定到 HTML 最后一个 `<th>` 而不是屏幕显示中的最后一列。
2. **缺少 SVG 图标** — 预览模式初始状态无图标；AI 消息用 Unicode 字符替代 SVG。
3. **缺少毛玻璃背景** — 两个表格按钮都使用纯色 `var(--card-bg)`，无 `backdrop-filter`。
4. **缺少 `min-width`** — 按钮宽度随文字变化，复制/已复制切换时会抖动。

## Proposed Changes

### 修改概览

```
frontend/src/main.js              # 预览模式：按钮移到 lastTh，加 SVG 图标，去 ResizeObserver
frontend/src/js/ai-chat.js         # AI 消息：改为使用 SVGS 图标常量
frontend/src/css/components/editor.css   # 预览样式：加 backdrop-filter/min-width，改 hover 规则
frontend/src/css/components/ai-chat.css  # AI 样式：加 backdrop-filter/min-width
```

### 1. `frontend/src/main.js` — 预览模式表格按钮重构

**当前问题**：按钮放在 `.table-wrapper` 内，`right: 6px` 锚定整个表格右边缘。水平滚动时按钮可能不可见。

**修改**：

**a) DOM 结构调整**（[main.js#L3676-L3725](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/main.js#L3676-L3725)）：
- 保留 `table-wrapper` 作为 spacing 容器（`margin: 1.2em 0`），但**不再**在其中放置按钮
- 改用 `lastTh = table.querySelector('tr:first-child th:last-child')` 定位按钮
- 按钮通过 `lastTh.appendChild(btn)` 放入最后一个 `<th>` 内
- **删除**：`ResizeObserver`、JS 动态计算 `top`、整个 `updateBtnPosition` 函数

**b) 图标**：
- 初始状态：`btn.innerHTML = SVGS.copy + ' 复制'`
- 成功：`btn.innerHTML = SVGS.checkmark + ' 已复制'`（已有，不变）
- 失败：`btn.innerHTML = SVGS.xmark + ' 复制失败'`（已有，不变）

**c) Class**：
- 保留 `copy-table-btn`

### 2. `frontend/src/js/ai-chat.js` — AI 消息表格按钮图标统一

**当前问题**：使用 Unicode 字符 `\u2713` / `\u2717` 而非 SVG。

**修改**（[ai-chat.js#L2524-L2550](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/js/ai-chat.js#L2524-L2550)）：
- 初始：`copyBtn.innerHTML = SVGS.copy + ' 复制'`（替代 `copyBtn.textContent = '复制'`）
- 成功：`copyBtn.innerHTML = SVGS.checkmark + ' 已复制'`（替代 `copyBtn.textContent = '\u2713 已复制'`）
- 失败：`copyBtn.innerHTML = SVGS.xmark + ' 复制失败'`（替代 `copyBtn.textContent = '\u2717 复制失败'`）
- 恢复：`copyBtn.innerHTML = SVGS.copy + ' 复制'`（替代 `copyBtn.textContent = '复制'`）

需要确保 `SVGS` 在 ai-chat.js 中可用。查看 import 情况。

### 3. `frontend/src/css/components/editor.css` — 预览模式样式升级

**当前**：

```css
.copy-table-btn {
    position: absolute;
    top: 0;
    right: 6px;
    transform: none;
    padding: 2px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--card-bg);
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.6;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s, transform 0.15s;
    z-index: 1;
    display: inline-flex;
    align-items: center;
    gap: 4px;
}
```

**修改为**（[editor.css#L496-L545](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%A1%B9%E7%9B%AE/jot/frontend/src/css/components/editor.css#L496-L545)）：

```css
.copy-table-btn {
    position: absolute;
    top: 50%;
    right: 6px;
    transform: translateY(-50%);
    padding: 2px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--card-bg) 85%, transparent);
    backdrop-filter: blur(4px);
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.6;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s, transform 0.15s;
    z-index: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    min-width: 62px;
    text-align: center;
}
```

**hover 规则变更**：
- 当前：`.md-rendered .table-wrapper:hover .copy-table-btn`
- 改为：`.md-rendered th:has(.copy-table-btn):hover .copy-table-btn` 或更兼容的写法（`:has()` 兼容性已足够）

Since the button is now inside `<th>` (which is inside `<tr>`), the simplest hover trigger that works well: target the parent `<th>` using `:has()`, or alternatively the `<tr>`:

```css
.md-rendered tr:first-child th:last-child:hover .copy-table-btn,
.md-rendered .copy-table-btn:hover {
    opacity: 1;
}
```

**移除 `table-wrapper` 的 `position: relative`**（不再需要作为按钮定位容器），保留 `margin: 1.2em 0` 仅用于间距。

```css
.md-rendered .table-wrapper {
    margin: 1.2em 0;
    /* 移除 position: relative */
}
```

### 4. `frontend/src/css/components/ai-chat.css` — AI 消息样式升级

**当前**：

```css
.ai-msg-assistant .table-copy-btn {
    position: absolute;
    top: 50%;
    right: 6px;
    transform: translateY(-50%);
    padding: 2px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--card-bg);
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.6;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s, transform 0.15s;
    z-index: 1;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    white-space: nowrap;
}
```

**修改为**（[ai-chat.css#L200-L242](file:///d:/%E8%B5%84%E6%BA%90%E6%B1%A0/%E4%B8%8B%E6%B0%B4%E9%81%93/Dev/%E6%9C%AC%E5%9C%B0%E9%81%93/Dev/jot/frontend/src/css/components/ai-chat.css#L200-L242)）：

```css
.ai-msg-assistant .table-copy-btn {
    position: absolute;
    top: 50%;
    right: 6px;
    transform: translateY(-50%);
    padding: 2px 8px;
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--card-bg) 85%, transparent);
    backdrop-filter: blur(4px);
    color: var(--text-muted);
    font-size: 12px;
    line-height: 1.6;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, background 0.15s, transform 0.15s;
    z-index: 1;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    min-width: 62px;
    text-align: center;
    white-space: nowrap;
}
```

hover 和 copied 状态保持原有选择器：
```css
.ai-msg-assistant .table-copy-btn:hover {
    background: var(--hover-bg);
    color: var(--text-primary);
    transform: translateY(-50%) scale(1.08);
}

.ai-msg-assistant .table-copy-btn.copied {
    opacity: 1;
    color: #16a34a;
    border-color: #16a34a;
}
```

### 5. AI 消息 tr:hover 改为 th:hover（可选优化）

当前 hover 规则 `tr:hover .table-copy-btn` 意味着在**行**上悬浮才显示。改为仅在**表头单元格**上悬浮显示更精确（但 tr:hover 也 OK，不必改）。

### 6. `frontend/src/js/constants.js` — 检查 SVGS 导入路径

确认 ai-chat.js 中 `SVGS` 的导入方式。如果未导入，需要添加 import。

## 涉及的文件清单

| 文件 | 修改类型 | 说明 |
|------|----------|------|
| `frontend/src/main.js` | 修改 | 按钮定位从 table-wrapper 移到 lastTh；去 ResizeObserver；加 SVG 图标 |
| `frontend/src/js/ai-chat.js` | 修改 | Unicode 字符替换为 `SVGS.*` 图标常量 |
| `frontend/src/css/components/editor.css` | 修改 | backdrop-filter、min-width、justify-content；hover 规则；移除 table-wrapper position |
| `frontend/src/css/components/ai-chat.css` | 修改 | backdrop-filter、min-width、justify-content |

## Verification

1. 笔记预览中表格复制按钮显示在最后一个 `<th>` 内，悬停表头可见，带 SVG 图标 + 毛玻璃
2. AI 消息中表格复制按钮显示在最后一个 `<th>` 内，带 SVG 图标 + 毛玻璃
3. 表格水平滚动时，按钮跟随其 `<th>` 列，不会消失在右边缘外
4. 点击复制后正确切换为"已复制"（绿色勾 SVG），1.5s 后恢复
5. `npm run build` 构建无报错
