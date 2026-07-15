# 移动 AI 聊天滚动条至窗口右侧边框 Spec

## Why

当前 AI 助手消息列表的滚动条位于 `#aiChatMessages` 元素内部右侧。滚动条无法贴到窗口右侧边框的原因有两个：

1. `.ai-chat-content` 有 `padding: 0 16px 16px`，左右 16px padding 导致子元素内容区距父容器边框有 16px 间隙
2. `#aiChatMessages` 通过 `max-width: clamp(...)` + `margin: 0 auto` 居中，进一步将滚动条向内推

## What Changes

- **`.ai-chat-content`**：移除左右 padding（`padding: 0 0 16px`），让子元素能延伸到边框位置
- **`#aiChatMessages`**：移除 `max-width` + `margin: 0 auto`，改为全宽，滚动条自然出现在 `.ai-chat-content` 右侧边框位置
- **新增 `.ai-chat-messages-inner`**：内层容器承载居中布局，补偿父级去除的 padding
- **其他子元素**：`.ai-chat-input-area`、`.ai-chat-welcome`、`.ai-chat-empty` 补偿 padding 以保持视觉不变
- **JS**：新增 `messagesInnerEl` 引用，DOM 操作从 `messagesEl` 切到 `messagesInnerEl`

## Impact

- Affected specs: AI 聊天界面布局
- Affected code:
  - `frontend/index.html` — 在 `#aiChatMessages` 内插入 `.ai-chat-messages-inner`
  - `frontend/src/css/components/ai-chat.css` — 5 处样式调整
  - `frontend/src/js/ai-chat.js` — 新增 `messagesInnerEl` 引用，替换 DOM 操作目标

## ADDED Requirements

### Requirement: 滚动条位于窗口右侧边框

`#aiChatMessages` SHALL 为全宽布局，滚动条自然出现在 `.ai-chat-content` 右侧边框位置。

`.ai-chat-content` SHALL 移除 `padding-right`，使 `#aiChatMessages` 能延伸到边框处。

#### Scenario: 滚动条位置验证
- **GIVEN** AI 聊天页面已打开且有足够多的消息触发滚动
- **WHEN** 查看消息列表右侧边缘
- **THEN** 滚动条 SHALL 与 `.ai-chat-content` 右侧边框对齐
- **AND** 滚动条在视觉上紧贴窗口右侧

### Requirement: 各子元素保持原有视觉边距

`.ai-chat-content` 去除左右 padding 后，各子元素 SHALL 补偿相应的水平 padding 以保持视觉间距不变：

| 元素 | 原总边距 | 补偿后 padding |
|------|---------|---------------|
| `.ai-chat-messages-inner` | 父16px + 自20px = 36px | `16px 36px 72px` |
| `.ai-chat-input-area` | 父16px + 自32px = 48px | `8px 48px 16px` |
| `.ai-chat-welcome` | 父16px + 自20px = 36px | `16px 36px` |
| `.ai-chat-empty` | 父16px（无自padding） | `0 36px` |

### Requirement: 现有交互功能不受影响

以下交互功能在调整后 SHALL 保持正常工作：

| 功能 | 验证方式 |
|------|---------|
| `scrollToBottom()` | 新消息发送/接收后自动滚动到底部 |
| 滚动加载更多 | 滚动到顶部触发历史消息分页加载 |
| 滚动条自动显隐 | `scrolling` CSS 类切换，1 秒延迟淡出 |
| 代码块滚动事件 | `stopPropagation` 防止冒泡 |
| 消息入场动画 | `.ai-msg-enter-anim` 动画正常播放 |

## MODIFIED Requirements

### Requirement: `.ai-chat-content` 样式调整

```css
/* Before */
.ai-chat-content {
    padding: 0 16px 16px;  /* 左右 16px 阻碍滚动条贴边 */
}

/* After */
.ai-chat-content {
    padding: 0 0 16px;     /* 仅保留底部 padding */
}
```

### Requirement: 消息列表 CSS 样式拆分

**外层 `#aiChatMessages`（`.ai-chat-messages`）**：
- `flex: 1` — 保留
- `overflow-y: auto` — 保留（滚动条在此）
- `min-height: 0` — 保留
- `width: 100%` — 保留
- `box-sizing: border-box` — 保留
- `scroll-behavior: smooth` — 保留
- 移除：`padding`、`display: flex`、`flex-direction: column`、`gap`、`max-width`、`margin: 0 auto`

**新增 `.ai-chat-messages-inner`**：
- `max-width: clamp(800px, min(92vw, 100%), 1600px)` — 居中约束
- `margin: 0 auto` — 水平居中
- `padding: 16px 36px 72px` — 补偿父级去除的 padding
- `display: flex` + `flex-direction: column` + `gap: 40px` — 消息排列
- `width: 100%` + `box-sizing: border-box` + `min-height: 0`

### Requirement: JS DOM 操作目标迁移

JS 文件 SHALL 新增 `messagesInnerEl` 变量指向 `.ai-chat-messages-inner`，DOM 操作从 `messagesEl` 迁移至 `messagesInnerEl`。滚动相关属性和事件保持在 `messagesEl` 上不变。

## REMOVED Requirements

无
