# 为「深度思考」按钮新增悬停提示

## Summary

给 AI 助手输入框的「深度思考」开关按钮（`#aiChatSearchToggle`）添加一套与现有模式/审批按钮一致的悬停提示卡片，向用户说明：开启深度思考**不等于强制**模型深度思考，是否生效取决于当前模型是否支持深度思考/推理能力，避免误解。

## Current State Analysis

* 该按钮当前的唯一提示是原生浏览器 `title="深度思考"`（[index.html](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L1252)），信息量有限、样式不统一。

* 项目已有一套成熟的悬停提示系统 `initModeTips()`（[ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L520)）：

  * 在 `#aiModeTipPortal` 内预置 `.ai-mode-tip[data-tip="xxx"]` 卡片，由 JS 把「触发元素 → tip」注册进 `tipMap`，并绑定 `mouseenter`(300ms 延迟)/`mouseleave` 事件，控制 `.visible` 显隐、自动定位（含视口边缘防溢出、上下翻转）。

  * 现有绑定对象：模式按钮、上下文压缩圆环（`#aiChatContextUsage`）、审批下拉选项、消息/工具/召回统计卡。

* 深度思考的真实后端行为经代码核实：

  * 开启后向前端流式接口传递 `enable_thinking` 参数（[einocli/chat.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/einocli/chat.go#L113)）；Agent 模式还设 `reasoning_effort=high`（[agent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L729)）。

  * 是否真正产出思考链取决于服务端模型是否支持；不支持时接口可能忽略或返回 `model_not_support_thinking` 类错误（[aierrors/errors.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/aierrors/errors.go#L240)），前端会给出中文友好错误。

## Proposed Changes

### 1. `frontend/index.html`

* 在 `#aiModeTipPortal` 内、`data-tip="context-usage"` 悬停卡之后（约 [L2795](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/index.html#L2791-L2795)），新增悬停卡：

  ```html
  <!-- 深度思考开关的同款悬停提示（initModeTips 绑定显示，不再用原生 title） -->
  <div class="ai-mode-tip" data-tip="deep-think">
      <span class="ai-mode-tip-title">深度思考</span>
      <span class="ai-mode-tip-line">开启后会向接口发送深度思考请求（enable_thinking），恳请模型进入思考。</span>
      <span class="ai-mode-tip-line">是否真正深度思考取决于当前模型是否支持该能力，并非强制生效。</span>
      <span class="ai-mode-tip-line">模型不支持时该请求可能被忽略，或返回错误提示；关闭则始终不开启思考。</span>
  </div>
  ```

* 将 `#aiChatSearchToggle` 的原生 `title="深度思考"` 移除（避免与自定义卡双重弹出；自定义卡标题已含「深度思考」）。

### 2. `frontend/src/js/ai-chat.js` — `initModeTips()`

* 在「上下文压缩圆环」绑定之后（约 [L618-L620](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L613-L620)），新增对 `#aiChatSearchToggle` 的绑定，复用既有 `scheduleShow`/`hide`：

  ```js
  // 深度思考开关：同款悬停提示（说明开启不等于强制，取决于模型是否支持）
  const deepThinkToggle = document.getElementById('aiChatSearchToggle');
  const deepThinkTip = portal.querySelector('.ai-mode-tip[data-tip="deep-think"]');
  if (deepThinkToggle && deepThinkTip) {
      tipMap.set(deepThinkToggle, deepThinkTip);
      deepThinkToggle.addEventListener('mouseenter', () => scheduleShow(deepThinkToggle));
      deepThinkToggle.addEventListener('mouseleave', hide);
  }
  ```

* 说明：`#aiChatSearchToggle` 不属 `.ai-mode-btn`，不受 `is-locked` 禁弹逻辑影响，行为与「上下文压缩圆环」一致（仅悬停展示）。

## Assumptions & Decisions

* 提示文案只陈述「开启传递请求、是否生效取决于模型支持」这一事实，不做超出代码行为的承诺（如「一定进入思考」）。

* 复用既有 `initModeTips` 体系，不新建独立 tooltip 组件，保持样式/定位/交互与审批、上下文压缩等提示一致。

* 设置页的「深度思考」项（index.html L583-586）不在本次范围内，仅改输入框按钮。

## Verification

1. `npm run build` 前端编译通过。
2. 手工确认：鼠标悬停输入框「深度思考」按钮约 300ms 后弹出卡片，标题为「深度思考」，正文含「取决于模型是否支持」等说明；鼠标移开后卡片消失；原生 title 不再弹出。
3. 打开其他列表/按钮时该提示卡不受影响（仅悬停显示）；回复期间按钮 `is-locked` 状态下悬停提示仍可正常查看。
4. 如需桌面应用生效，再执行 `wails build` 生成二进制。

