# 修复新建空会话重命名后标题不更新问题

## 问题描述

新建的 AI 会话（无对话消息），在侧边栏重命名后：
- 侧边栏会话条目标题 ✅ 已更新
- 页面顶部 `#aiChatTitle` 仍然显示 "AI 助手" ❌ 未同步更新

## 根因分析

`updateChatTitle()` 函数的判断条件过于严格：

**文件**: `frontend/src/js/ai-chat.js` 第 3114-3123 行

```javascript
function updateChatTitle() {
    const titleEl = document.getElementById('aiChatTitle');
    if (!titleEl) return;
    if (activeSessionId !== null && chatHistory.length > 0) {  // ← 问题在这里
        const s = sessions.find(s => s.id === activeSessionId);
        titleEl.textContent = s ? s.title : 'AI 助手';
    } else {
        titleEl.textContent = 'AI 助手';
    }
}
```

条件 `activeSessionId !== null && chatHistory.length > 0` 要求**同时**满足：
1. 存在活跃会话
2. 会话有历史消息（`chatHistory.length > 0`）

新建的空会话 `chatHistory.length === 0`，导致条件不满足，总是走 `else` 分支显示 "AI 助手"，即使会话已被重命名。

## 修改方案

仅修改 `frontend/src/js/ai-chat.js` 中 `updateChatTitle()` 的条件判断：

将：
```javascript
if (activeSessionId !== null && chatHistory.length > 0) {
```

改为：
```javascript
if (activeSessionId !== null) {
```

**理由**：
- 只要存在活跃会话，就应该显示该会话的标题
- 即便会话为空（无消息），会话对象在 `sessions` 数组中也有正确的 `title` 属性
- `sessions.find()` 若找不到匹配项，已有兜底逻辑显示 "AI 助手"

## 影响范围

该函数共被 7 处调用（rename、switchSession、createSession、showWelcome 等场景），修改后所有调用场景都会受惠：
- **空会话重命名后** → 顶部标题正确更新 ✅（本次修复的核心场景）
- **空会话切换** → 正确显示对应会话标题，而非 "AI 助手" ✅（附带改进）
- **已有消息的会话** → 行为不变 ✅

## 验证步骤

1. 新建 AI 会话（不发消息）
2. 在侧边栏通过双击/右键菜单/更多菜单重命名该会话
3. 确认页面顶部标题 `#aiChatTitle` 同步更新为新的名称
4. 切换回有消息的旧会话，确认标题仍正确显示
5. 创建新会话（不重命名），确认标题显示 "AI 助手"
