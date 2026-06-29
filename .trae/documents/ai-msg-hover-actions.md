# AI 消息气泡悬浮操作按钮

## 概述

为 AI 对话的消息气泡添加悬浮操作按钮：
1. **复制按钮** — 所有消息（用户 + AI）hover 时显示，点击复制内容到剪贴板
2. **重新生成按钮** — 仅 AI 回复消息 hover 时显示，点击从该处重新生成

## 设计决策

### 1. 复制按钮（所有消息）
- 跟随项目现有 hover 显隐模式：默认 `opacity: 0`，hover 时 `opacity: 1`（参考 [editor.css 的代码块复制按钮](file:///d%3A/资源池/下水道/Dev/本地项目/jot/frontend/src/css/components/editor.css#L372-L380)）
- 使用 `navigator.clipboard.writeText()` 复制纯文本内容
- 复制成功后短暂视觉反馈（0.5s），如变色 + 打勾图标

### 2. 重新生成按钮（仅 AI 回复消息）
- 点击后从 `chatHistory` 中移除该条 AI 回复及之后的所有消息
- 同步从 DOM 中移除对应的气泡
- 以截断后的历史重新调用 `CallAIStream`，生成新的回复
- 例如：chatHistory = `[user1, asst1, user2, asst2]`，点击 asst1 的重新生成 → 保留 `[user1]`，重新请求

### 3. 用户消息结构调整
- 当前用户消息直接用 `el.textContent = content`，没有子容器
- 需要包裹 `.msg-content` 子容器（与 AI 消息一致），以统一布局和按钮定位

### 4. 按钮布局
- 容器 `.ai-msg-actions` 绝对定位在气泡右上角（`top: 6px; right: 8px`）
- 按钮 26×26px，默认透明，hover 时 `var(--hover-bg)` + `var(--text-primary)`
- 使用 inline SVG 图标

## 修改文件

### 文件 1: `frontend/src/js/ai-chat.js`

| 位置 | 改动 |
|------|------|
| `addMessage()` | 用户消息改为包裹 `.msg-content`；尾部追加 `.ai-msg-actions`（复制 + 若为 assistant 则加重新生成） |
| `onSend()` 流式气泡 | 在 `streamingEl` 尾部追加 `.ai-msg-actions`（仅重新生成按钮，初始 disabled，流完成后启用） |
| 新增 `createMsgActions(text, role, msgEl)` | 返回按钮容器 DOM |
| 新增 `handleCopy(text, btn)` | `navigator.clipboard.writeText(text)` → 按钮显示勾 + "已复制" → 0.5s 恢复 |
| 新增 `handleRegenerate(msgEl)` | 找出 msgEl 在 messagesEl.children 中的索引 → `chatHistory.splice(idx)` → 移除后续 DOM → `CallAIStream(chatHistory)` |

### 文件 2: `frontend/src/css/components/ai-chat.css`

| 规则 | 说明 |
|------|------|
| `.ai-msg { position: relative; }` | 按钮定位锚点 |
| `.ai-msg-actions` | `position: absolute; top: 6px; right: 8px; display: flex; gap: 2px; opacity: 0; transition: opacity 0.15s;` |
| `.ai-msg:hover .ai-msg-actions` | `opacity: 1` |
| `.ai-msg-actions button` | `width: 26px; height: 26px; border-radius: 6px; border: none; display: flex; align-items: center; justify-content: center; cursor: pointer; background: transparent; color: var(--text-secondary);` |
| `.ai-msg-actions button:hover` | `background: var(--hover-bg); color: var(--text-primary);` |
| `.ai-msg-actions button.copied` | 复制成功态：绿色着色调 |

### SVG 图标
- **复制图标**：标准两层矩形（与项目代码块复制按钮一致）
- **重新生成图标**：循环箭头（↻）

## 验证

1. `npx vite build` — 零错误
2. `wails build -s` — 零错误
3. 功能验证：
   - 用户消息 hover 出现复制按钮，点击复制成功
   - AI 回复 hover 出现复制 + 重新生成按钮
   - 点击重新生成 → 从该处截断并生成新回复
