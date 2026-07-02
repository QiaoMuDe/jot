# 修复 AI 空回复气泡布局错位

## Summary

当 AI 调用结束但 `finalContent` 为空（即后端既未推 `ai:stream-error` 也未推任何 `ai:stream-chunk`，仅发送了 `ai:stream-done` 事件）时，AI 气泡内的 `.msg-content` 高度坍缩为 0，仅剩 20px padding。叠加在气泡底部的操作栏（`position: absolute; bottom: -28px`）因为定位算法是相对**父容器底部**算偏移，会越过父容器、回到父容器内部。结果：

* 「⏱ 3.6 秒」耗时标签陷入气泡内部、容器宽度被压到 1 字符宽，文字竖排成 `3`/`.`/`6`

* 操作按钮虽然 z-index 在上，但仍可能与气泡内容视觉重叠

修复从三个角度解决：(1) 改操作栏定位方式，避免与极矮的父容器重叠；(2) 文字 nowrap + 气泡最小宽度，避免塌缩到不可读尺寸；(3) JS 侧检测空回复，注入明确的占位提示，让用户知道发生了什么，并方便点「重新生成」重试。

***

## Current State Analysis

### 关键文件

* `frontend/src/css/components/ai-chat.css` — 气泡、操作栏、耗时的样式

* `frontend/src/js/ai-chat.js` — 流式生命周期、DOM 创建、占位/错误消息

### 相关代码点

**操作栏定位** ([ai-chat.css:1336-1346](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1336-L1346))

```css
.ai-msg-actions {
    position: absolute;
    bottom: -28px;        /* ← 高度坍缩时与父容器重叠 */
    left: 0;
    right: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    z-index: 2;
}
```

**气泡基础** ([ai-chat.css:31-82](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L31-L82))：无 min-width / min-height，AI 回复 max-width: 82%。

**耗时 span** ([ai-chat.css:2534-2540](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L2534-L2540))：无 `white-space: nowrap`，窄容器下中文/字符会换行。

**空内容检测** ([ai-chat.js:1863-1891](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1863-L1891))：`ai:stream-done` 事件中 `finalContent` 为空时仍正常执行 `renderMarkdown(contentDiv, '')`，气泡正常显示，但没有任何提示。

**错误路径** ([ai-chat.js:1999-2014](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1999-L2014))：仅在收到 `ai:stream-error` 时调用 `addErrorMessage(err)`，流式正常结束但内容为空时不会触发。

### 根因总结

* `bottom: -28px` 是相对父容器底部定位

* 当父容器高度 < 28px 时，绝对定位元素会**穿过**父容器顶部边界，进入父容器内部

* AI 空内容时父容器仅 20px → 操作栏上沿掉进父容器 22px 处 → 耗时 span 出现在气泡内部

* span 容器宽 1 字符 → 「3.6」竖排

***

## Proposed Changes

### 1. 调整操作栏定位方式（CSS）

**文件**：`frontend/src/css/components/ai-chat.css` (line 1336-1346)

把 `bottom: -28px` 改成 `top: 100%` + `margin-top: 2px`，操作栏**始终紧贴父容器下沿外侧**，与父容器高度无关：

```css
.ai-msg-actions {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 2px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    z-index: 2;
}
```

这样无论气泡多矮，操作栏都不会进入气泡内部。

### 2. 防止耗时文字竖排（CSS）

**文件**：`frontend/src/css/components/ai-chat.css` (line 2534-2540)

给 `.ai-msg-time` 加 `white-space: nowrap`：

```css
.ai-msg-time {
    font-size: 0.75rem;
    color: var(--text-muted);
    user-select: none;
    line-height: 26px;
    white-space: nowrap;     /* 新增：窄容器下不换行 */
}
```

### 3. AI 气泡最小宽度（CSS）

**文件**：`frontend/src/css/components/ai-chat.css` (line 73-82)

给 `.ai-msg-assistant` 加 `min-width` 防止空内容气泡完全坍缩，便于显示占位文字：

```css
.ai-msg-assistant {
    align-self: flex-start;
    background: var(--card-bg);
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 16px 16px 16px 4px;
    max-width: 82%;
    min-width: 200px;        /* 新增：保证空内容时气泡有合理宽度 */
    box-shadow: 0 1px 3px var(--shadow);
}
```

200px 足够容纳「⚠️ AI 未返回内容，请重新生成」这种提示。

### 4. 空回复占位提示（CSS + JS）

#### 4.1 新增 CSS

**文件**：`frontend/src/css/components/ai-chat.css`

在 `.ai-msg-time` 样式块附近（约 line 2540 之后）添加：

```css
/* ── 空回复占位提示 ── */
.ai-msg-empty {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--text-muted);
    font-size: 0.8125rem;
    line-height: 1.4;
}
.ai-msg-empty-icon {
    flex-shrink: 0;
    color: #d97706;   /* 警示琥珀色 */
}
```

#### 4.2 新增 JS 逻辑

**文件**：`frontend/src/js/ai-chat.js` (在 `ai:stream-done` 回调内，约 line 1877-1890 之间)

在 `renderMarkdown(contentDiv, finalContent)` 之后，插入对空内容的检测和占位：

```js
const finalContent = streamingContent || fullContent;
renderMarkdown(contentDiv, finalContent);

// 空回复占位：模型未返回任何内容（流正常结束但无 chunk）
if (!finalContent || !finalContent.trim()) {
    contentDiv.innerHTML = '';
    const empty = document.createElement('div');
    empty.className = 'ai-msg-empty';
    const icon = document.createElement('span');
    icon.className = 'ai-msg-empty-icon';
    icon.innerHTML = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
    const text = document.createElement('span');
    text.textContent = 'AI 未返回内容，请尝试重新生成';
    empty.appendChild(icon);
    empty.appendChild(text);
    contentDiv.appendChild(empty);
}
```

占位元素使用现有 flex 布局渲染，宽度由 `.ai-msg-assistant { min-width: 200px }` 保证。

***

## Assumptions & Decisions

* **不动** **`ai:stream-error`** **路径**：该路径已经走 `addErrorMessage(err)`，行为合理。修复只针对**流正常结束但内容为空**的边缘情况。

* **不修改** **`addMessage`** **历史渲染路径**：历史消息如有空内容，加载时会以同样方式回显占位提示。

* **`min-width: 200px`** **是上限而非下限**：仅在内容为空时由 min-width 决定气泡宽度；正常消息仍按 max-width: 82% 渲染。

* **操作栏改用** **`top: 100%`** **而不是** **`bottom: -28px`**：前者语义更清晰（操作栏总是在气泡外），行为更稳健，与父容器高度解耦。

* **不修改后端**：纯前端展示层修复。

***

## Verification Steps

1. **构建前端**

   ```bash
   cd frontend && npm run build
   ```
2. **复现空回复**：临时在 `app.go` 的 `CallAIStream` 中模拟「只发 done 不发 chunk」的场景（如直接返回），发送一次 AI 消息。
3. **目视检查**：

   * [ ] 气泡内有清晰的「⚠️ AI 未返回内容，请尝试重新生成」提示

   * [ ] 气泡宽度约 200px 左右，不再是 30px 的窄列

   * [ ] 「⏱ X.X 秒」耗时标签在气泡**外**底部左侧横排，不再竖排

   * [ ] 操作按钮（复制/保存/重新生成/追问）位于气泡外底部右侧

   * [ ] 鼠标悬停时操作按钮正常显示
4. **回归检查**：

   * [ ] 正常 AI 回复：耗时在底部左侧，操作按钮悬浮在右侧

   * [ ] 用户消息：布局无变化

   * [ ] 联网搜索/卡片召回消息：折叠面板在耗时上方，位置合理
5. **构建并运行** `wails build` 确认无报错。

