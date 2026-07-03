# 用户消息 Token 显示及操作栏位置重新设计

## 需求概述

将用户消息的 tokens 显示位置从右侧（当前）移至左侧，与 AI 消息的 token 显示位置对齐。根据可用宽度自适应显示：宽度充足时悬停同时显示 tokens 和操作按钮，宽度不足时悬停切换显示。

## 当前状态分析

### 当前布局结构（用户消息）

```
.ai-msg.ai-msg-user (气泡容器)
  └── .msg-content (消息文本)
  └── .ai-msg-actions (position: absolute; top: 100%; display: flex)
        ├── .user-tokens <span>       ← margin-left: auto 右对齐
        └── .action-buttons <div>     ← margin-left: auto, opacity: 0 默认隐藏
```

### 当前悬停行为

* 默认：`.user-tokens` 显示（opacity: 0.75）

* 悬停：`.user-tokens` 隐藏（opacity: 0）、`.action-buttons` 显示（opacity: 1）

### 相关文件

* `frontend/src/js/ai-chat.js` — `createMsgActions()` 第 2722 行、`collapseActionsIfNeeded()` 第 2867 行

* `frontend/src/css/components/ai-chat.css` — `.user-tokens` 第 1268 行、`.ai-msg-actions` 第 1283 行、`.action-buttons` 第 1296 行

## 修改方案

### 1. CSS 修改 — `ai-chat.css`

#### 1.1 `.user-tokens` 左对齐

```css
/* 修改前 */
.ai-msg-user .user-tokens {
    font-size: 11px;
    color: var(--text-secondary);
    margin-left: auto;           /* ← 右对齐 */
    opacity: 0.75;
    transition: opacity 0.15s;
    white-space: nowrap;
}

/* 修改后 */
.ai-msg-user .user-tokens {
    font-size: 0.75rem;          /* 与 .ai-msg-time 一致 */
    color: var(--text-muted);    /* 与 .ai-msg-time 一致 */
    user-select: none;
    line-height: 26px;           /* 与 .ai-msg-time 一致，垂直居中 */
    white-space: nowrap;
    transition: opacity 0.15s;
}
```

移除 `margin-left: auto`，使 tokens 在 flex 容器中自然左对齐。样式与 `.ai-msg-time` 保持一致。

#### 1.2 新增宽度模式 CSS 类

```css
/* 悬停时隐藏 token（窄模式） */
.ai-msg-user.narrow-mode:hover .user-tokens {
    opacity: 0;
    pointer-events: none;
}

/* 悬停时 token 保持可见（宽模式） */
.ai-msg-user.wide-mode:hover .user-tokens {
    opacity: 0.75;
}
```

`.wide-mode`：宽度充足时，悬停保持 tokens 可见，操作按钮同时显示。
`.narrow-mode`：宽度不足时，悬停隐藏 tokens，仅显示操作按钮（当前行为）。

### 2. JS 修改 — `ai-chat.js`

#### 2.1 `createMsgActions()` 第 2722 行

**无需结构性修改。** `user-tokens` 元素本身位置正确（在 `.action-buttons` 之前创建），通过 CSS 左对齐即可。

#### 2.2 `collapseActionsIfNeeded()` 第 2867 行 — 新增宽度模式检测

```js
function collapseActionsIfNeeded(msgEl) {
    const actions = msgEl?.querySelector('.ai-msg-actions');
    const btnWrap = actions?.querySelector('.action-buttons');
    if (!actions || !btnWrap) return;

    btnWrap.classList.remove('collapsed');

    requestAnimationFrame(() => {
        const availableWidth = actions.clientWidth;
        const buttonsWidth = btnWrap.scrollWidth;
        if (buttonsWidth > availableWidth && buttonsWidth > 60) {
            btnWrap.classList.add('collapsed');
        }

        // 新增：检测用户消息的宽度模式
        if (msgEl.classList.contains('ai-msg-user')) {
            const tokensEl = actions.querySelector('.user-tokens');
            if (tokensEl) {
                const tokensWidth = tokensEl.scrollWidth;
                const totalWidth = tokensWidth + buttonsWidth;
                if (totalWidth > availableWidth) {
                    msgEl.classList.add('narrow-mode');
                    msgEl.classList.remove('wide-mode');
                } else {
                    msgEl.classList.add('wide-mode');
                    msgEl.classList.remove('narrow-mode');
                }
            }
        }
    });
}
```

逻辑：

* 如果是用户消息，测量 `.user-tokens` + `.action-buttons` 总宽度

* 总宽度 > 可用宽度 → `narrow-mode`（悬停切换显示）

* 总宽度 ≤ 可用宽度 → `wide-mode`（悬停同时显示）

#### 2.3 调用点验证

`collapseActionsIfNeeded()` 已在所有创建用户消息操作栏的地方被调用：

* `onSend()` 第 1854 行 ✓

* `loadHistoryMessages()` 第 1606 行 ✓

* `handleResend()` 第 3230 行 ✓

**无需额外调用。**

### 3. 文件变更清单

| 文件                                        | 修改内容                                                                                                   |
| ----------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `frontend/src/css/components/ai-chat.css` | ① `.user-tokens` 左对齐/样式一致性 ② 新增 `.wide-mode:hover .user-tokens` ③ 新增 `.narrow-mode:hover .user-tokens` |
| `frontend/src/js/ai-chat.js`              | `collapseActionsIfNeeded()` 中新增宽度模式检测逻辑                                                                |

## 实现后行为对照

| 场景           | 默认（无悬停）      | 悬停                         |
| ------------ | ------------ | -------------------------- |
| **宽模式-用户消息** | 左侧显示 tokens  | 左侧 tokens 保持可见 + 右侧操作按钮显示  |
| **窄模式-用户消息** | 左侧显示 tokens  | tokens 隐藏 + 右侧操作按钮显示（当前行为） |
| **AI 消息**    | 左侧显示时间+token | 右侧操作按钮显示（不变）               |

## 验证步骤

1. 发送用户消息 → 确认 tokens 显示在左侧
2. 悬停用户消息（窗口够宽）→ tokens 保持可见，操作按钮同时显示
3. 缩小窗口/增加按钮数 → 触发 narrow-mode → 悬停时 tokens 隐藏，仅显示按钮
4. 对比 AI 消息 → tokens 字体/颜色/大小一致
5. 加载历史消息 → tokens 显示位置正确
6. 重新发送/编辑消息 → tokens 位置正确

