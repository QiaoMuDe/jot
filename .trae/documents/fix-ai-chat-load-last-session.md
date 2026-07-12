# 修复 AI 助手加载会话时优先加载置顶会话而非最后使用会话

## 根因分析

**后端排序** (`ai_service.go` 第 382 行):
```go
a.db.Order("CASE WHEN is_pinned = 1 THEN 0 ELSE 1 END, updated_at DESC").Find(&sessions)
```
置顶会话永远排在非置顶会话之前。

**前端选择** (`ai-chat.js` 第 2880 行):
```javascript
await switchSession(sessions[0].id);
```
直接取列表第一个，即置顶会话中最新的一个。

**结果**: 只要存在任意置顶会话，`sessions[0]` 永远是置顶会话。即使最近使用的是普通会话，打开 AI 助手时也会加载到置顶会话。

## 修改方案

### 修改文件: `frontend/src/js/ai-chat.js`

**位置**: 第 2878-2883 行，`onAIChatViewActivated()` 中

**修改前**:
```javascript
if (sessions.length > 0) {
    await switchSession(sessions[0].id);
}
```

**修改后**:
```javascript
if (sessions.length > 0) {
    // 在所有会话中选 updated_at 最新的（忽略置顶优先），确保加载最后使用的会话
    const mostRecent = sessions.reduce((a, b) =>
        new Date(a.updated_at) > new Date(b.updated_at) ? a : b
    );
    await switchSession(mostRecent.id);
}
```

**原理**: 无论排序优先级如何，`reduce` 遍历所有会话找出 `updated_at` 最大的那个，即真正最后使用过的会话。

### 不修改部分

- 后端排序不变：UI 列表仍然置顶在前、按时间排序
- 其他前端逻辑不变：`renderSessionList()`、`switchSession()`、`createSession()`
- `activeSessionId` 持久化逻辑不变：切换视图不重置 `activeSessionId`，只有冷启动（`activeSessionId === null`）才走这条路径

## 验证步骤

1. `npx vite build` 构建通过
2. 手动测试：
   - 有置顶会话 A，最近使用普通会话 B → 打开 AI 助手 → 加载 B ✅
   - 置顶会话 A 比普通会话 B 更新 → 打开 AI 助手 → 加载 A（因为它确实是最后使用的）✅
   - 无置顶会话，多个普通会话 → 加载最后使用的 ✅
   - 无任何会话 → 新建会话 ✅
