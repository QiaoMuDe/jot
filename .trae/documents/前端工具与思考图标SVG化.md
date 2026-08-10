# 前端工具调用与思考链图标 SVG 化

## Summary

将 AI 聊天中两处使用 emoji/文本字符作为图标的 UI 替换为 SVG 线条图标：
1. **Agent 工具调用状态条**（`ai-tool-status-item` 的图标）：`🔍` / `✓` / `❌` / `⚠️`
2. **思考/思维链折叠区**（`thinking-summary` 的图标）：`💭`

SVG 采用项目现有 Feather 线条风格（`viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"`），图标颜色随现有 `is-done` / `is-error` / `is-warning` 状态类自动继承（`currentColor`），"进行中/思考中"状态加轻微脉冲动画（class-driven，支持 `prefers-reduced-motion`）。

仅改前端两个文件，后端零改动。

## Current State Analysis

### 工具状态条（实时流，`ai-chat.js`）
- 容器/条目样式：[ai-chat.css L429-478](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L429-L478)
- 图标当前为 emoji/字符，通过 `item.iconEl.textContent = ...` 写入：
  - [showToolStatusStart](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2472-L2507)：`🔍`（L2495 创建时、L2504 更新时）
  - [showToolStatusDone](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2510-L2529)：`✓`（L2526）
  - [showToolStatusError](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2532-L2561)：`❌`（L2558）
  - [showToolStatusPartial](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2564-L2577)：`⚠️`（L2574）

### 工具状态条（历史回放，`ai-chat.js`）
- [renderToolCalls](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3969-L4024)：`iconEl.textContent = '❌'`（L4000）/ `'⚠️'`（L4009）/ `'✓'`（L4014）

### 思考/思维链
- 实时创建：[appendThinkingChunk](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2270-L2302) L2288-2290：`summary.textContent = '💭 思考中'`
- 实时计时：[updateThinkingTimer](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2251-L2256) L2255：`summary.textContent = '💭 思考中 X 秒'`（**整段覆盖 textContent**）
- 结束态：[stopThinkingTimer](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2259-L2268) L2266：`summary.textContent = '💭 已思考 X 秒'`（**整段覆盖 textContent**）
- 历史渲染：[addMessage](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2949-L2973) L2966：`summary.textContent = '💭 已思考 X 秒'`
- 样式：[ai-chat.css L2199-2239](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2199-L2239)，`.thinking-summary` 已是 `display:flex; align-items:center; gap:4px`，可直接容纳 SVG。

### 项目现有 SVG 约定
`frontend/src` 中多处使用 Feather 风格内联 SVG（如 [ai-chat.js L2874](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2874)、[calendar.js L259](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/calendar.js#L259)），本次保持同一风格。

## Proposed Changes

### 1. `frontend/src/js/ai-chat.js`

**a. 新增模块级 SVG 图标函数**（放在 `addMessage` 之前、文件靠上位置，供实时流与历史回放共用）：

```js
// ── AI 消息内嵌 SVG 图标（Feather 线条风格，随 currentColor 着色） ──
function svgIcon(name, size = 14) {
    const paths = {
        search:   '<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>',
        check:    '<polyline points="20 6 9 17 4 12"/>',
        x:        '<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>',
        alert:    '<path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>',
        brain:    '<path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 4.44-2.04z"/><path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-4.44-2.04z"/>',
    };
    const p = paths[name];
    if (!p) return '';
    return '<svg class="ai-inline-icon' + (name === 'brain' ? ' ai-brain-icon' : '') + '" width="' + size + '" height="' + size + '" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' + p + '</svg>';
}
```

**b. 工具状态条实时流**（iconEl 从 `textContent` 改为 `innerHTML`）：
- `showToolStatusStart`：L2495 创建时与 L2504 更新时 → `item.iconEl.innerHTML = svgIcon('search');`（搜索图标）
- `showToolStatusDone`：L2526 → `item.iconEl.innerHTML = svgIcon('check');`
- `showToolStatusError`：L2558 → `item.iconEl.innerHTML = svgIcon('x');`
- `showToolStatusPartial`：L2574 → `item.iconEl.innerHTML = svgIcon('alert');`

**c. 工具状态条历史回放**（renderToolCalls）：
- L4000 `'❌'` → `svgIcon('x')`
- L4009 `'⚠️'` → `svgIcon('alert')`
- L4014 `'✓'` → `svgIcon('check')`

**d. 思考/思维链折叠区**（summary 由纯文本改为 `SVG + <span>` 结构，更新时只改文本节点，避免覆盖图标）：
- `appendThinkingChunk` L2288-2290：
  ```js
  const summary = document.createElement('summary');
  summary.className = 'thinking-summary is-thinking';
  summary.innerHTML = svgIcon('brain') + '<span class="thinking-text">思考中</span>';
  ```
- `updateThinkingTimer` L2255 → 只更新文本：
  ```js
  const text = summary?.querySelector('.thinking-text');
  if (text) text.textContent = '思考中 ' + elapsed.toFixed(1) + ' 秒';
  ```
- `stopThinkingTimer` L2266 → 更新文本并移除思考动画态：
  ```js
  const text = summary?.querySelector('.thinking-text');
  if (text && finalElapsed > 0) text.textContent = '已思考 ' + finalElapsed.toFixed(1) + ' 秒';
  const sum = thinkingDetails?.querySelector('.thinking-summary');
  if (sum) sum.classList.remove('is-thinking');
  ```
- `addMessage`（历史）L2966 → 同样结构：
  ```js
  summary.innerHTML = svgIcon('brain') + '<span class="thinking-text">' + (thinkingElapsed > 0 ? '已思考 ' + thinkingElapsed.toFixed(1) + ' 秒' : '已思考') + '</span>';
  ```

> 注：`summary` 的 `toggle` 事件监听（L2285/L2961）不受影响；`querySelector('.thinking-summary')` 用法不受影响。

### 2. `frontend/src/css/components/ai-chat.css`

**a. 通用内联图标**：
```css
.ai-inline-icon {
    flex-shrink: 0;
    display: block;
}
```

**b. 工具状态图标适配**（`.ai-tool-status-icon` 现有 `font-size/line-height` 对 SVG 无效，保留无妨；补充 SVG 尺寸）：
```css
.ai-tool-status-icon .ai-inline-icon {
    width: 14px;
    height: 14px;
}
```

**c. 思考区图标**：
```css
.thinking-summary .ai-inline-icon {
    width: 14px;
    height: 14px;
}
```

**d. 进行中/思考中脉冲动画**（class-driven + `prefers-reduced-motion` 降级）：
```css
.ai-tool-status-item.is-active .ai-tool-status-icon,
.thinking-summary.is-thinking .ai-inline-icon {
    animation: ai-icon-pulse 1.2s ease-in-out infinite;
}

@keyframes ai-icon-pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
}

@media (prefers-reduced-motion: reduce) {
    .ai-tool-status-item.is-active .ai-tool-status-icon,
    .thinking-summary.is-thinking .ai-inline-icon {
        animation: none;
    }
}
```

**e. 触发动画的状态类挂接**（JS 侧）：
- `showToolStatusStart`：`item.el.classList.add('is-active')`（完成/失败/部分失败时 `remove('is-active')`）
- `showToolStatusDone` / `showToolStatusError` / `showToolStatusPartial`：`item.el.classList.remove('is-active')`
- 历史回放 `renderToolCalls`：无进行中状态，不涉及。

## Assumptions & Decisions

1. **图标风格**：沿用项目现有 Feather 线条风格（stroke 1.5-2、currentColor），与日历/复制按钮等内联 SVG 视觉统一。
2. **颜色**：SVG 用 `stroke="currentColor"`，直接继承 `.is-done`（accent）/`.is-error`（danger）/`.is-warning`（warning）现有配色，无需新配色。
3. **尺寸**：工具图标与思考图标统一 14px（与现有按钮图标 14px 一致）。
4. **动画**：仅"进行中（🔍）"与"思考中（💭）"加轻脉冲动画，完成后静态；`prefers-reduced-motion` 下禁用。
5. **不改变布局结构**：仍为 `summary`（思考）+ `.ai-tool-status-item`（工具）结构，只替换图标载体与文本更新方式。
6. 不引入任何新依赖/图标库。

## Verification

1. 重新构建运行（`wails dev` 或前端构建）。
2. **Agent 模式工具调用**：提问触发 web_search/recall_notes → 观察"进行中"图标（放大镜脉冲）→ 完成后变为对勾；失败（key 错误）变交叉；部分来源失败变警告三角。颜色随状态类变化。
3. **深度思考**：开启深度思考提问 → 思考折叠区图标（大脑脉冲）+ "思考中 X 秒" → 完成后停止脉冲、显示"已思考 X 秒"。
4. **历史回放**：切换会话重新加载，历史消息中的工具状态条与思考区图标均为 SVG，样式与实时一致。
5. **无障碍**：系统开启"减少动态效果"后，脉冲动画消失、图标保持静态显示。
