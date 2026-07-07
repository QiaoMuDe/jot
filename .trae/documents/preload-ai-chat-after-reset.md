# 重置/还原后预加载 AI 聊天页面

## 总结

当前 AI 聊天页面只在用户**主动点击** AI 助手选项卡后才执行初始化（`onAIChatViewActivated`），而 `switchView` 中有一个 50ms 的 `setTimeout`，导致视图可见后异步加载 `GetAIConfig()` 之前有一段空白时间，产生"一闪而过"现象。

**方案**：在重置出厂设置 / 一键还原完成后，**主动提前调用** `onAIChatViewActivated()`，使 AI 聊天页面的 DOM 状态、模块级变量（`chatHistory`、`sessions`、`activeSessionId`）在用户点击选项卡之前就已准备就绪。

## 当前状态分析

### 问题根因

```
用户点击 AI 助手选项卡
  → switchView('ai-chat')
  → #viewAiChat 立即 visible（虽然 innerHTML 已清空，但 display 状态未设置）
  → setTimeout 50ms
  → onAIChatViewActivated()
  → async GetAIConfig()         ← 异步等待
  → no API key → showEmptyState() → 隐藏消息区、显示空状态
```

在这 50ms + 异步等待期间，用户看到的是**未准备好的中间态**（空消息区 + 输入框可见），然后才跳转到正确的"尚未配置 AI 服务"空状态。

### 现有关键函数

| 函数                        | 位置                          | 作用                                                         |
| ------------------------- | --------------------------- | ---------------------------------------------------------- |
| `resetDatabase()`         | `data-management.js` \~L164 | 重置 DB → 清空 AI 消息/会话 DOM → `reloadSettings()` → 切换到 grid 视图 |
| `restoreFromDir()`        | `data-management.js` \~L349 | 一键还原 → 刷新笔记/标签/设置 → 切换到 grid 视图                            |
| `onAIChatViewActivated()` | `ai-chat.js` \~L2467        | 读取 AI 配置 → 未配置则 `showEmptyState()`，已配置则加载会话列表              |
| `showEmptyState()`        | `ai-chat.js` \~L2547        | 隐藏消息区/输入区，显示空状态，清空模块变量                                     |

## 变更方案

### 改动的文件和具体内容

#### 1. `frontend/src/main.js` — 暴露 `onAIChatViewActivated` 到 window

**原因**：`resetDatabase()` 和 `restoreFromDir()` 在 `data-management.js` 中，无法直接调用 `ai-chat.js` 的导出函数。通过 `window` 解耦是最小侵入的方式，无需修改模块导出结构。

**修改**：在 `import` 语句附近的 `window` 暴露区域新增一行：

```javascript
// 已有: window.initAIChat = initAIChat;
window.onAIChatViewActivated = onAIChatViewActivated;
```

#### 2. `frontend/src/js/data-management.js` — 重置/还原后预加载 AI 聊天

**`resetDatabase()`** **中的修改**：

在 `reloadSettings()` 调用之后、侧栏折叠之前，新增：

```javascript
// 提前预加载 AI 聊天页面状态，防止后续切换时闪烁
window.onAIChatViewActivated?.();
```

**`restoreFromDir()`** **中的修改**：

在 `reloadSettings()` 调用之后新增同样的调用：

```javascript
// 提前预加载 AI 聊天页面状态
window.onAIChatViewActivated?.();
```

### 执行流程对比

**修改前**（用户点击 AI 助手选项卡后）：

```
view visible → 50ms → onAIChatViewActivated → async GetAIConfig → showEmptyState
                    ↑ 这里有闪烁
```

**修改后**（重置/还原完成后）：

```
resetDatabase → ... → window.onAIChatViewActivated()
  → GetAIConfig → no API key → showEmptyState()
  → 设置 DOM display、清空模块变量
  → AI 聊天页面已处于正确状态

（用户稍后点击 AI 助手选项卡）
  → switchView('ai-chat')
  → view visible（已处于空状态，无变化）
  → 50ms → onAIChatViewActivated() 再次调用
  → GetAIConfig → 仍是 no API key → showEmptyState()（幂等，无闪烁）
```

### 不涉及的改动

* `ai-chat.js`：无需任何修改。`showEmptyState()` 已经是幂等的。

* `index.html`：无需修改。

* Go 后端：无需修改。

## 假设与决策

1. **重置后一定无 API Key**：重置会清空所有设置，`GetAIConfig()` 返回空 → `showEmptyState()`。这是轻量操作，无性能开销。

2. **还原后可能有 API Key**：备份数据可能包含配置。还原后调用 `onAIChatViewActivated()` 会加载会话列表，但还原本身已是重量操作，增加此开销可接受。

3. **双重调用安全性**：用户点击选项卡时 `onAIChatViewActivated()` 会再次执行。`showEmptyState()` 是幂等的（重复设为相同 display 值不会重绘），已配置状态重复加载会话列表也不影响显示结果。不存在副作用。

4. **不修改 data-management.js 的 import**：通过 `window` 调用是当前代码库中已有的跨模块通信模式（如 `window.loadNotebooks`、`window.loadSettings` 等），保持一致。

## 验证步骤

1. 编译前端：`cd frontend && npm run build`（或 `go build ./...` 触发 Wails 构建）
2. 点击设置 → 数据管理 → 重置出厂设置
3. 立即点击 AI 助手
4. **预期**：直接显示"尚未配置 AI 服务"空状态，无闪烁
5. 配置 API Key，发送几条消息
6. 点击设置 → 数据管理 → 一键还原（需要有一个有效备份）
7. 还原完成后点击 AI 助手
8. **预期**：直接显示正确的页面状态（空状态或有会话列表），无闪烁

