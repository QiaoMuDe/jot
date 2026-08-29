# 方案：AI 回复期间锁定模式切换与深度思考按钮（点击抖动+通知）

## Summary

AI 流式回复期间（`isStreaming === true`），锁定两个工具栏按钮的切换功能：
- **Agent/Plan 模式切换**（`#aiModeToggle`）
- **深度思考开关**（`#aiChatSearchToggle`）

锁定期间视觉置灰；若用户仍点击，按钮**抖动**并通过 `window.showNotification` 提示"回复进行中"。回复正常完成 / 停止 / 出错后解除锁定，恢复切换功能。

## Current State Analysis

- 模式切换按钮 DOM：[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1192-L1197)，两个 `<button class="ai-mode-btn">`。
- 深度思考开关 DOM：[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L1210-L1214)，`<div id="aiChatSearchToggle">`（非 button，无原生 disabled）。
- 两个点击处理当前**均不检查 `isStreaming`**：
  - 模式切换：[ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L502-L513)
  - 深度思考：[ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L877-L881)
- 流式生命周期：`startStreaming`（[L2278-L2280](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2278-L2280)）置 `isStreaming = true`；所有流式入口（发送/重新生成/重发/反问续答）均经由 `startStreaming`。
- `isStreaming = false` 重置点共 **4 处**（现有代码已在这些位置恢复发送/停止/润色按钮，模式一致）：
  1. 停止按钮 [L581](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L581)
  2. `ai:stream-error` [L2850](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2850)
  3. `ai:stream-done` [L2725](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2725)
  4. `startStreaming` catch [L2909](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2909)
- 样式位置：[ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1195-L1233)（模式切换）、[L1244-L1297](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css#L1244-L1297)（深度思考）。项目已有 shake 动画先例（如 [animations.css L204](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/animations.css#L204)、`ai-ask-shake`）。

**关键约束**：不能用原生 `disabled`（会阻断 click 事件，导致无法触发抖动+通知）。深度思考开关是 `div` 也原生不支持 disabled。因此采用"类锁定 + 点击守卫"方案：点击事件仍可达，由守卫判断 `isStreaming` 决定是否放行或抖动提示。

## Proposed Changes

### 1. [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js) — 新增两个工具函数

在 `syncModeToggle`（约 L5751）附近新增：

```js
/** AI 回复期间锁定模式/深度思考切换（视觉置灰，点击时抖动提示） */
function setToggleLocked(locked) {
    document.getElementById('aiModeToggle')?.classList.toggle('is-locked', locked);
    searchToggle?.classList.toggle('is-locked', locked);
}

/** 锁定状态下被点击：触发抖动动画（重启动画用 reflow） */
function shakeLockedToggle(el) {
    if (!el) return;
    el.classList.remove('is-shaking');
    void el.offsetWidth;
    el.classList.add('is-shaking');
    el.addEventListener('animationend', () => el.classList.remove('is-shaking'), { once: true });
}
```

### 2. [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js) — 点击守卫

- 模式切换处理（L503 开头）：
```js
btn.addEventListener('click', () => {
    if (isStreaming) {
        shakeLockedToggle(btn);
        window.showNotification?.('回复进行中，暂时无法切换模式', 'warning');
        return;
    }
    ...
```
- 深度思考处理（L877 开头）：
```js
searchToggle.addEventListener('click', async () => {
    if (isStreaming) {
        shakeLockedToggle(searchToggle);
        window.showNotification?.('回复进行中，暂时无法切换深度思考', 'warning');
        return;
    }
    ...
```

### 3. [ai-chat.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js) — 生命周期挂钩

- `startStreaming` 中 `isStreaming = true` 后：`setToggleLocked(true);`
- 上述 4 处 `isStreaming = false` 旁各加一行：`setToggleLocked(false);`

### 4. [ai-chat.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/ai-chat.css) — 锁定与抖动样式

在模式切换样式区（L1233 后）新增：

```css
/* AI 回复期间锁定：置灰（不使用 pointer-events，保留点击以触发抖动提示） */
#aiModeToggle.is-locked .ai-mode-btn,
.ai-chat-search-toggle.is-locked {
    opacity: 0.45;
    cursor: not-allowed;
}

/* 锁定态点击抖动 */
@keyframes ai-toggle-shake {
    0%, 100% { transform: translateX(0); }
    20% { transform: translateX(-3px); }
    40% { transform: translateX(3px); }
    60% { transform: translateX(-2px); }
    80% { transform: translateX(2px); }
}
.ai-mode-btn.is-shaking,
.ai-chat-search-toggle.is-shaking {
    animation: ai-toggle-shake 0.4s ease;
}
```

## Assumptions & Decisions

- 锁定范围：Agent/Plan 模式切换 + 深度思考两个按钮；模型选择、更多技能等其它工具栏项不在本次范围。
- 采用"类锁定 + 点击守卫"而非原生 `disabled`：`disabled` 会阻断 click，无法实现"点击抖动+通知"；深度思考是 `div` 也不支持 disabled。
- 锁定视觉用 `opacity: 0.45` 置灰（符合用户"禁用态静态灰化"偏好），**不加** `pointer-events: none`（会阻断抖动反馈）。
- 润色（`isPolishOptimizing`）走独立 `CallAI` 通道、不经过 `startStreaming`，不参与锁定逻辑。
- 通知文案区分：模式切换提示"无法切换模式"，深度思考提示"无法切换深度思考"。

## Verification

1. `node --check frontend/src/js/ai-chat.js` 语法校验。
2. 运行应用（vite HMR 热更新）：
   - 发送消息后：两个按钮置灰。
   - 回复期间点击任一按钮：按钮抖动 + 弹出提示通知，模式/思考状态不变。
   - 回复完成后：按钮恢复，可正常切换；active 态正确。
   - 中途点停止、出错场景：立即恢复。
   - Plan 模式 + 深度思考开启时分别验证。
