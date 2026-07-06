# AI 助手模块 — 会话恢复功能修改计划

## 概要

将 AI 助手模块的入口行为从"每次打开模块时自动创建新会话"改为"每次打开模块时自动恢复用户上一次使用的对话历史"。

## 现状分析

### 当前代码流程

视图激活入口在 [frontend/src/js/ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js) 的 `onAIChatViewActivated()` 函数（第 2395-2427 行）：

```
main.js (switchView → 'ai-chat' case)
  → setTimeout(onAIChatViewActivated, 50)
    → loadSessionList()
    → if (activeSessionId === null) { createSession() }
```

* `activeSessionId` 初始值为 `null`（第 24 行），未持久化

* 每次调用 `onAIChatViewActivated()` 时，只要 `activeSessionId === null`，就无条件调用 `createSession()` 创建新会话

* 结果是"每次打开 AI 助手模块，必定出现一个空白新会话"

### 现有可复用机制

* [GetAISessions 服务](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/services/ai_service.go#L352-L386) 已按 `updated_at DESC` 排序返回所有会话（置顶优先）

* [switchSession(id)](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1331-L1386) 已能完整加载历史会话的消息并渲染

* [createSession()](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L1401-L1436) 创建新会话逻辑保持不变

### 需要变更的位置

仅需修改 [frontend/src/js/ai-chat.js](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2415-L2420) 中 `onAIChatViewActivated()` 函数内约 6 行代码的决策逻辑。

## 修改方案

### 修改文件

**仅修改 1 个文件：** `frontend/src/js/ai-chat.js`

### 具体改动

在 `onAIChatViewActivated()` 函数中（第 2415-2420 行），将：

```javascript
// 没有激活会话时，始终创建新对话
if (activeSessionId === null) {
    await createSession();
}
```

改为：

```javascript
// 没有激活会话时，恢复上次使用的会话；无历史会话时才新建
if (activeSessionId === null) {
    if (sessions.length > 0) {
        await switchSession(sessions[0].id);
    } else {
        await createSession();
    }
}
```

### 逻辑说明

1. **`loadSessionList()`** 已在前一步（第 2413 行）执行完毕，`sessions` 数组已按 `updated_at DESC` 排序
2. `sessions[0]` 即为最近使用（或置顶）的会话
3. 调用 `switchSession(sessions[0].id)` 会：

   * 设置 `activeSessionId`

   * 从后端加载该会话的全部消息

   * 逐条渲染到消息列表

   * 更新侧栏高亮
4. 仅当没有任何历史会话时，才调用 `createSession()` 创建新会话

### 边界情况

| 场景                                | 行为                                         |
| --------------------------------- | ------------------------------------------ |
| 有历史会话（含消息）                        | 恢复最近使用的会话，显示历史消息                           |
| 所有历史会话已被删除                        | 同"无历史会话"→ 创建新对话                            |
| 已存在 `activeSessionId`（如切换页面后又切回来） | 保持当前会话不变（`activeSessionId !== null`，不走该分支） |

## 无需修改的部分

* **后端**: `GetAISessions()`/`CreateAISession()`/`switchSession()` 等已有 API 完全满足需求

* **前端其他模块**: 无需改动（`createSession()` 由用户主动点击"新建"按钮触发，不受影响）

## 验证步骤

1. 打开 AI 助手模块 → 应自动加载并显示最近使用的会话及其消息
2. 新建多个会话，切换并发送消息 → 关闭并重新打开 AI 助手模块 → 应自动恢复最后使用的会话
3. 删除所有会话 → 重新打开 AI 助手模块 → 应自动创建一个新的空白会话（显示欢迎语）
4. 在"笔记格子"视图中切换到其他页面再切回 AI 助手 → 当前会话应保持不变

