# 修复「恢复出厂设置」后 AI 助手页面标题和 Token 残留

## 问题描述

恢复出厂设置后，点击 AI 助手页面：

* 页面正确显示「尚未配置 AI 服务」的空状态提示

* 但顶部标题（`#aiChatTitle`）和右上角的会话总 Token 消耗（`#aiChatContextSize`）仍显示重置前的旧数据

## 根因分析

**`showEmptyState()`** **只隐藏了消息区和输入区，没有处理标题和 Token 显示：**

```
view-title (#aiChatTitle)         ← 未被 showEmptyState 管理，残留旧值
view-controls (#aiChatContextSize) ← 未被 showEmptyState 管理，残留旧 Token 数
─────────────────────────────────
#aiChatEmpty [显示]                ← showEmptyState 控制
#aiChatMessages [隐藏]             ← showEmptyState 控制 (messagesEl)
#aiChatInputArea [隐藏]            ← showEmptyState 控制 (inputAreaEl)
```

**调用链**：

1. 恢复出厂设置 → 清除数据库所有数据（`ResetDatabase`）
2. 用户点击 AI 助手标签 → `onAIChatViewActivated()`
3. `syncToolbarState()` → 从 DOM 同步变量（此时 `chatHistory`、`sessions`、`activeSessionId` 仍是老数据）
4. `GetAIConfig()` → 返回默认配置（无 API Key）→ `showEmptyState()`
5. **`showEmptyState()`** **没有：**

   * 重置标题文本（`#aiChatTitle` 仍显示旧会话名）

   * 清除 Token 数（`#aiChatContextSize` 仍显示旧的 Token 值）

   * 清空模块变量（`chatHistory`、`sessions`、`activeSessionId` 仍持有已失效的数据）

## 修改方案

### 修改 1：`showEmptyState()` — 重置标题、Token、模块变量

**文件**：`frontend/src/js/ai-chat.js`（第 2547-2555 行）

```javascript
function showEmptyState() {
    if (!emptyEl) return;
    emptyEl.style.display = '';
    if (messagesEl) messagesEl.style.display = 'none';
    if (inputAreaEl) inputAreaEl.style.display = 'none';
    if (clearBtnEl) clearBtnEl.style.display = 'none';
    // 侧栏仍可见但禁用操作
    if (sessionNewBtnEl) sessionNewBtnEl.style.display = 'none';
    // ★ 新增：重置标题和 Token 显示
    const titleEl = document.getElementById('aiChatTitle');
    if (titleEl) titleEl.textContent = 'AI 助手';
    if (contextSizeEl) {
        contextSizeEl.textContent = '';
        contextSizeEl.style.display = 'none';
    }
    // ★ 新增：清空内存中的旧会话数据（数据库已重置，这些数据已失效）
    chatHistory = [];
    sessions = [];
    activeSessionId = null;
}
```

#### 各新增行说明

| 行                                      | 作用                |
| -------------------------------------- | ----------------- |
| `titleEl.textContent = 'AI 助手'`        | 将标题恢复为默认值         |
| `contextSizeEl.textContent = ''`       | 清除 Token 数字文本     |
| `contextSizeEl.style.display = 'none'` | 隐藏 Token 区域       |
| `chatHistory = []`                     | 清空消息历史（内存中的已失效数据） |
| `sessions = []`                        | 清空会话列表（DB 中已无数据）  |
| `activeSessionId = null`               | 重置为无活跃会话          |

#### 为什么清空模块变量是安全的

`showEmptyState()` 仅在两种情况下被调用：

1. **AI 服务未配置**（`GetAIConfig()` 返回无 API Key）：DB 中即使有数据也无法使用，清空变量无影响
2. **`GetAIConfig()`** **抛出异常**（网络错误等）：此时用户正在**切换到 AI 聊天视图**，之前的数据已通过自动保存写入 DB。下次 AI 服务可用时 `onAIChatViewActivated()` 重新 `loadSessionList()`，会从 DB 重新加载

清空变量后，如果用户后续配置了 AI 服务并回到 AI 聊天视图：

1. `onAIChatViewActivated()` → `GetAIConfig()` → 配置完整
2. `loadSessionList()` → DB 为空（重置后无数据）→ `sessions = []`
3. `activeSessionId === null` → `sessions.length === 0` → `createSession()`

流程完整，不会出现副作用。

## 涉及文件

| 文件                           | 修改内容                                           |
| ---------------------------- | ---------------------------------------------- |
| `frontend/src/js/ai-chat.js` | `showEmptyState()` 新增：重置标题、清除 Token 显示、清空模块级变量 |

## 验证步骤

1. 先正常使用 AI 聊天，确保标题和 Token 有显示值
2. 执行恢复出厂设置
3. 点击 AI 助手标签
4. 确认：

   * 页面显示「尚未配置 AI 服务」的提示

   * 顶部标题显示「AI 助手」（不是旧会话名）

   * 右上角无 Token 消耗显示

