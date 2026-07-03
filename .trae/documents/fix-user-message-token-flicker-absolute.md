# 修复用户消息悬停时 token/操作按钮闪烁问题

## 问题

用户消息区域悬停时，tokens 和操作按钮疯狂闪烁。根因是**悬停触发了 CSS 布局变动**（token 零宽化 `width: 0; flex: 0 0 0`、按钮 `margin-left` 变化），在按钮右侧边缘区域 hover → unhover 反复循环。

## 方案

将用户消息的 `.user-tokens` 和 `.action-buttons` 改为 `position: absolute` 叠加布局，**悬停时只变 opacity，不触发布局变动**。AI 消息保持原有 flex 布局不变。

## 修改内容

### 文件：`frontend/src/css/components/ai-chat.css`

#### 1. 覆盖用户消息的 `.ai-msg-actions`（第 1292 行附近）
```css
/* 新增 */
.ai-msg-user .ai-msg-actions {
    display: block;          /* 覆盖 flex，让绝对定位的 children 叠加 */
    height: 26px;            /* 固定高度维持空间 */
}
```
- 为什么：`display: block` 替代 `flex`，`position: absolute` children 才能正确叠加
- 为什么：`height: 26px` 防止容器塌陷（absolute 元素脱离正常流）

#### 2. 用户消息 `.user-tokens` 改为绝对定位
```css
/* 替换原第 1269-1275 行 */
.ai-msg-user .user-tokens {
    position: absolute;
    left: 0;
    top: 0;
    font-size: 0.75rem;
    color: var(--text-muted);
    user-select: none;
    line-height: 26px;
    white-space: nowrap;
    transition: opacity 0.12s;
}

.ai-msg-user:hover .user-tokens {
    opacity: 0;
    pointer-events: none;
    /* 不改变 width/flex/overflow — 没有布局变动 */
}
```
- 为什么：绝对定位脱离 flex 流，`opacity` 变化不触发重排
- 为什么：保留 `transition: opacity 0.12s` 让切换顺滑，因无布局变动所以不会闪烁

#### 3. 用户消息 `.action-buttons` 改为绝对定位
```css
/* 替换原第 1304–1309 行 + 第 1286-1289 行 */
.ai-msg-user .action-buttons {
    position: absolute;
    left: 32px;
    top: 0;
    display: flex;
    gap: 2px;
    margin-left: 0;         /* 覆盖默认的 auto */
    opacity: 0;
    transition: opacity 0.12s;
}

.ai-msg-user:hover .action-buttons {
    opacity: 1;
}
```
- 为什么：`left: 32px` 与用户想要的偏移量一致
- 为什么：`margin-left: 0` 覆盖 `margin-left: auto`，否则 auto 会把绝对定位元素推到右边

#### 4. AI 消息恢复过渡动画
```css
/* 恢复第 1304-1309 行添加 transition */
.ai-msg-actions .action-buttons {
    display: flex;
    gap: 2px;
    margin-left: auto;
    opacity: 0;
    transition: opacity 0.12s;  /* 恢复此前被移除的过渡 */
}
```
- 为什么：AI 消息是 flex 布局，之前为了修闪烁把 transition 去掉了，现在用户消息已独立处理，AI 消息可以恢复平滑过渡

#### 5. 删除不再需要的规则
删除以下规则（已被上述绝对定位规则取代）：
- 第 1277-1284 行：`.ai-msg-user:hover .user-tokens { width: 0; flex:… ; overflow… }`（零宽化布局）
- 第 1286-1289 行：`.ai-msg-user:hover .action-buttons { margin-left: 32px }`（margin 覆盖）

### 文件：`frontend/src/js/ai-chat.js`

#### 6. 调整 `collapseActionsIfNeeded` 中的用户消息计算
```js
// 用户消息的可用宽度处理
if (msgEl.classList.contains('ai-msg-user')) {
    // 按钮从 left: 32px 开始，可用宽度需减去偏移
    const remaining = availableWidth - 32;
    isCollapsed = isCollapsed || (buttonsWidth > remaining && buttonsWidth > 60);
}
```
- 为什么：改为绝对定位后，`left: 32px` 占用固定偏移，不再需要测量 `tokensWidth`
- 注意：删除 `const tokensEl/tokensWidth` 相关代码

## 行为对照

| 场景 | 默认 | 悬停 | 布局变化 |
|------|------|------|---------|
| **用户消息** | tokens 可见（`left:0`），按钮隐藏 | tokens 渐隐（opacity→0），按钮渐显（opacity→1） | **无** |
| **AI 消息** | time+token 可见，按钮隐藏 | 按钮渐显（opacity→1） | 无布局变化（flex，保持原样） |

## 验证

1. 悬停在用户消息的操作按钮右侧边缘 → 不再闪烁
2. 用户消息默认状态显示 tokens
3. 用户消息悬停时 tokens 渐隐、操作按钮渐显
4. AI 消息悬停行为不受影响
5. 极窄场景下按钮正确折叠为"更多操作"
