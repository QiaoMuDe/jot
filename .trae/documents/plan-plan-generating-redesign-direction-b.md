# 计划：计划生成提示重设计（方向 B：SVG 描边清单 + 轮换文案）

## Summary

将 AI 助手 Plan 模式下「正在制定执行计划...」提示（border spinner + 静态文案）重设计为方向 B（已在 playground 预览确认）：行内 SVG 清单图标逐段描边 + 文案轮换 + 动态省略号 + 语义化重试副行。保持行内态，继续放在消息气泡内。

## Current State Analysis

* 提示逻辑：`frontend/src/js/ai-chat.js` L2831-2857（`ai:plan-generating` 事件监听），当前创建 `.ai-msg-plan-generating`（span 边框圈 spinner + `plan-gen-text` 静态文案「正在制定执行计划...」，重试时拼「（第 2 次尝试）」）。

* 样式：`frontend/src/css/components/ai-chat.css` L442-469（`ai-plan-gen-spin` / `ai-plan-gen-fadein` 两个 keyframes，无 reduced-motion 支持）。

* 生命周期：该元素位于气泡 `contentDiv` 内，会被多处 `contentDiv.innerHTML = ''` 清除（首 chunk L2533、plan-created L2877、ask-user L2910、agent-result L2974）。因此轮换定时器必须用 `wrap.isConnected` 自检停止，避免孤儿节点续写（项目教训：innerHTML 清空导致引用悬空）。

* 图标来源：`svgIcon('list', size)`（ai-chat.js L3308-3316，Feather 线条风格，随 currentColor 着色）。

* 进场曲线惯例：计划卡用 `0.2s cubic-bezier(0.22, 1, 0.36, 1)`（ai-chat.css L738）。

## Proposed Changes

### 1. `frontend/src/css/components/ai-chat.css`（L442-469 整块替换）

* `.ai-msg-plan-generating`：改为 `inline-flex; align-items: flex-start; gap: 9px; padding: 4px 0;`，进场动画沿用 `ai-plan-gen-fadein`，时长/曲线改为 `0.2s cubic-bezier(0.22, 1, 0.36, 1)`（与计划卡统一）。

* `.plan-gen-icon`：`color: var(--accent); display: flex; margin-top: 1px;`；其内 `svg { overflow: visible; }`，`svg line` 逐段描边：

  * `stroke-dasharray: 30; stroke-dashoffset: 30; animation: plan-gen-draw 0.5s ease forwards;`

  * 按 `nth-child` 递增 delay（0.05s / 0.35s / 0.65s / 0.95s / 1.15s / 1.35s，先三横线后三圆点）。

  * 描边完成后整体呼吸：`.plan-gen-icon` 加 `plan-gen-glow 1.6s ease-in-out 2.2s infinite`（opacity 1 → 0.55 → 1）。注：预览版的「蚂蚁线」在 13px 线段 + dasharray 30 下不可见，改为 opacity 呼吸，视觉反馈等价。

* `.plan-gen-main`：`display: flex; flex-direction: column; gap: 3px;`

* `.plan-gen-text`：`13px / weight 500 / var(--text-secondary)`，`position: relative; height: 18px; overflow: hidden;`；子 `span` 绝对定位，`.show`（opacity 1 / translateY 0）、`.hide`（opacity 0 / translateY -8px）、默认（opacity 0 / translateY 8px），transition `0.4s ease`。

* `.plan-gen-dots i`：3 个 3px accent 圆点，`plan-gen-dot 1.2s ease-in-out infinite` 依次点亮（delay 0 / 0.2s / 0.4s）。

* `.plan-gen-retry`：`11.5px / var(--text-muted)`。

* 新增 `@media (prefers-reduced-motion: reduce)` 块：描边直接到位（`animation: none; stroke-dashoffset: 0;`）、呼吸/圆点/文字位移全部停用（transition none）。

* 删除不再使用的 `@keyframes ai-plan-gen-spin`。

### 2. `frontend/src/js/ai-chat.js`（L2831-2857 重写 `ai:plan-generating` 监听）

* 顶部常量：`const PLAN_GEN_PHRASES = ['正在梳理任务目标…', '正在拆解执行步骤…', '正在规划最优路径…'];`

* DOM 结构（首次事件时创建，保留 `if (!hasReceivedChunk)` 门控）：

  ```
  wrap.ai-msg-plan-generating (aria-live="polite")
  ├─ span.plan-gen-icon  ← innerHTML = svgIcon('list', 16)
  └─ div.plan-gen-main
      ├─ div.plan-gen-text
      │   ├─ span（第一句，带 .show）
      │   └─ span.plan-gen-dots > i ×3
      └─ span.plan-gen-retry（默认隐藏）
  ```

* 文案轮换：`setInterval(2500)`；回调首行检查 `if (!wrap.isConnected) { clearInterval(...); return; }`（覆盖 chunk/plan-created/ask-user/done 等所有 `innerHTML=''` 清除路径）；轮换时创建新 span 加 `.show`、旧 span 加 `.hide` 并在 450ms 后移除。

* reduced-motion：`matchMedia('(prefers-reduced-motion: reduce)')` 命中时不启动轮换，仅显示静态第一句。

* 重试：`payload` 非空时 `plan-gen-retry.textContent = payload;` 并显示副行（payload 形如「第 2 次尝试」，语义化展示，不再拼进括号）。

* 保留既有门控/事件结构不变，不改其他监听器。

## Assumptions & Decisions

* 用户已选定方向 B，行内态放气泡内，无需容器化（方向 A 方案废弃）。

* 重试副行直接展示后端 payload 文本（如「第 2 次尝试」），不额外加工文案。

* 蚂蚁线改为 opacity 呼吸（dash 长度导致原方案不可见，见上）。

* 历史回放不涉及该状态（仅实时流显示），无需改动回放逻辑。

## Verification

1. `cd frontend && npm run build`，再 `wails build`（项目惯例：前端资源改动须重新构建）。
2. 运行应用，Plan 模式发送任务：观察描边逐段画出 → 呼吸循环；文案每 2.5s 上下位移轮换；省略号三点依次点亮。
3. 首个正文 chunk / 计划卡到达后提示被正常替换，无残留定时器（文案不再滚动即验证 isConnected 生效）。
4. 切换暗色主题（如暗夜 / Tokyo Night）检查 accent 图标与文字对比度。
5. 系统开启「减弱动态效果」后：描边静止到位、无轮换、圆点静止。

