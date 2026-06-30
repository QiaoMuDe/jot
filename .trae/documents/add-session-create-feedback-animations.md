# 为新会话创建交互添加反馈动画

## 摘要

目前三个新建会话的入口（`+` 按钮点击、双击"会话"标题、双击"AI 助手"标题）都没有交互动画反馈，点击/双击后感觉"没反应"。计划添加按钮按下反馈和新建条目入场动画。

## 当前状态分析

### 相关文件
- `frontend/src/css/components/ai-chat.css` — 侧栏和 AI 对话全部样式
- `frontend/src/css/animations.css` — 全局动画 keyframes 和工具类
- `frontend/src/js/ai-chat.js` — `createSession()` 和 `renderSessionList()` 函数

### 现有资源（可直接复用的动画）

`animations.css` 已有：
- `scalePulse` — scale 1 → 1.15 → 1，适合点击脉冲反馈
- `anim-slide-up` — opacity 0 + translateY(12px) → 正常，适合条目滑入
- `anim-scale-in` — opacity 0 + scale(0.9) → 正常（弹簧缓动），适合突出入场
- 工具类 `.anim-fade-in`、`.anim-slide-up`、`.anim-scale-in`

### 动画缺口

| 交互点 | 现有反馈 | 缺失 |
|--------|----------|------|
| `+` 按钮点击 | hover 变色 | 无 `:active` 按下缩放，无点击脉冲 |
| "会话" 标题双击 | 无任何 hover 或反馈 | 双击时无视觉响应 |
| "AI 助手" 标题双击 | 普通标题，无交互态 | 双击时无视觉响应 |
| 新建后条目出现 | 全量重渲染，无入场 | 新条目无滑入淡出效果 |

## 修改方案

### 修改 1: 按钮点击按下反馈 (`ai-chat.css`)

给 `.ai-session-new-btn` 添加 `:active` 按下状态：

```css
.ai-session-new-btn:active {
    transform: scale(0.92);
}
```

同时确保按钮有 `transform` 过渡支持。现有 `transition: var(--transition)` 通常覆盖 `all`，应该不需要额外属性。

### 修改 2: "会话" 标题 hover 和双击反馈 (`ai-chat.css`)

```css
.ai-session-sidebar-title {
    cursor: pointer;
    transition: color 0.15s ease;
}

.ai-session-sidebar-title:hover {
    color: var(--accent);
}
```

添加 `cursor: pointer` 告知用户可交互，hover 时变 accent 色暗示可双击。

双击反馈通过 JS 动态添加类实现（见修改 4）。

### 修改 3: "AI 助手" 标题 hover 反馈 (`ai-chat.css`)

`.view-title` 也需要 hover 提示：

```css
.view-title {
    cursor: default;
}
/* 在 AI 对话视图中，标题可双击 */
#viewAiChat .view-title {
    cursor: pointer;
    transition: color 0.15s ease;
}
#viewAiChat .view-title:hover {
    color: var(--accent);
}
```

### 修改 4: 双击脉冲动画 JS (通过临时类触发)

在 `ai-chat.js` 中，双击事件触发时给目标元素添加一个脉冲动画类：

```js
// 触发按钮脉冲反馈
function triggerPulseFeedback(el) {
    if (!el) return;
    el.classList.remove('anim-pulse');
    // 强制回流以重播动画
    void el.offsetWidth;
    el.classList.add('anim-pulse');
}
```

在 `animations.css` 中添加：

```css
@keyframes pulseClick {
    0% { transform: scale(1); }
    30% { transform: scale(1.12); }
    60% { transform: scale(0.95); }
    100% { transform: scale(1); }
}

.anim-pulse {
    animation: pulseClick 0.35s var(--anim-easing-spring) forwards;
}
```

双击事件处理器中调用：

```js
// 双击"会话"标题
sessionTitleEl.addEventListener('dblclick', () => {
    triggerPulseFeedback(sessionTitleEl);
    createSession();
});

// 双击"AI 助手"标题
aiChatTitleEl.addEventListener('dblclick', () => {
    triggerPulseFeedback(aiChatTitleEl);
    createSession();
});
```

### 修改 5: 新建会话条目入场动画 (`ai-chat.js` 中 `renderSessionList()`)

在 `renderSessionList()` 中，找出新创建的会话条目，添加滑入动画类。

方案：`createSession()` 末尾调用 `scrollToBottom()` 后，从列表中找出最新的 `.ai-session-item`，添加 `anim-slide-in` 动画。

```js
// 在 createSession() 末尾，loadSessionList() 之后
await loadSessionList();
// 为新条目添加入场动画
const items = sessionListEl.querySelectorAll('.ai-session-item');
if (items.length > 0) {
    const newest = items[0]; // 第一项是最新的（按 updated_at DESC）
    newest.classList.add('anim-slide-in');
}
```

在 `ai-chat.css` 中添加：

```css
@keyframes slideInNew {
    0% { opacity: 0; transform: translateY(-12px) scale(0.95); }
    100% { opacity: 1; transform: translateY(0) scale(1); }
}

.ai-session-item.anim-slide-in {
    animation: slideInNew 0.3s var(--anim-easing-out) forwards;
}
```

### 修改 6: `+` 按钮点击也触发脉冲

```js
sessionNewBtnEl.addEventListener('click', () => {
    triggerPulseFeedback(sessionNewBtnEl);
    createSession();
});
```

## 文件变更清单

| 文件 | 变更 |
|------|------|
| `frontend/src/css/animations.css` | 新增 `pulseClick` keyframe + `.anim-pulse` 类 |
| `frontend/src/css/components/ai-chat.css` | 新增 `:active` 按下、hover 交互、条目入场 keyframe |
| `frontend/src/js/ai-chat.js` | 新增 `triggerPulseFeedback()`，双击/点击事件中调用 |

## 验证方式

1. 点击 `+` 按钮：按钮瞬间缩小（scale 0.92），松开恢复
2. 双击"会话"标题：标题脉冲放大后恢复，新条目从上方滑入
3. 双击"AI 助手"标题：同上脉冲效果
4. 确认动画不影响功能：新建后会话正常创建、列表正常刷新
