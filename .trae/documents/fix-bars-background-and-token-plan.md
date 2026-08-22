# 修复：chip 项背景 + token 行穿透

## 问题分析

### 问题 1：chip 项与 AI 消息融合

三个 chip 类型的背景色都是 `color-mix(in srgb, var(--accent) 8%, transparent)`，即 92% 透明。AI 消息在透明 bars 区下方滚动时，透过 chip 显示出来，导致视觉融合。

### 问题 2：token 行穿透显示

`.ai-msg-actions` 是 `position: absolute; top: 100%`，`+28px` 的 padding 补偿边缘情况下不够，导致 token 行落入 bars/input 区。

## 方案

### 改动 1：chip 背景改为 opaque

**文件**：`frontend/src/css/components/ai-chat.css`

将三个 chip 类型（ref / skill / file）的 `background` 从 `color-mix(in srgb, var(--accent) 8%, transparent)` 改为 `color-mix(in srgb, var(--accent) 8%, var(--bg))`，hover 状态同理。

**效果**：chip 不再透明，不透明背景（8% accent + 92% 页面背景）使其与 AI 消息自然区分，且 bars 容器整体保持透明。

### 改动 2：增大 padding-bottom 补偿值

**文件**：`frontend/src/js/ai-chat.js`

`+28` → `+40`，提供 12px 额外安全缓冲。

## 文件修改清单

| 文件 | 改动 | 说明 |
|------|------|------|
| `frontend/src/css/components/ai-chat.css` | 3 处 chip 的 `background` 和 `:hover` 从 `transparent` 改为 `var(--bg)` | chip 项不透明，不与 AI 消息融合 |
| `frontend/src/js/ai-chat.js` | `padding-bottom` 补偿值 `+28` → `+40` | 增大安全缓冲 |

## 验证

1. 引用笔记/技能/文件后，chip 标签有实色背景，不透明，不与 AI 消息融合
2. bars 容器整体仍是透明，不影响布局
3. 最后一条消息的 token 行显示在 bars/input 区上方，不穿透