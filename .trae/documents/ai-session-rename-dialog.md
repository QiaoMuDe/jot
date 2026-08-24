# AI 会话侧边栏重命名改为对话框方案

## 摘要

将 AI 会话侧边栏的内联重命名（contentEditable）替换为弹出对话框，与笔记本重命名对话框方案统一。

## 当前状态

* `startInlineEdit(titleEl, sessionId)`（ai-chat.js 第1210行）用 contentEditable 编辑

* 三个触发入口：顶部标题栏双击（475行）、侧栏条目双击（1333行）、更多菜单"重命名"（3359行）

* CSS `.ai-session-item.editing`（ai-chat.css 第2184行）处理编辑态样式

## 变更方案

### 1. JS — `ai-chat.js`：替换 `startInlineEdit` 为 `showAISessionRenameDialog`

将 `startInlineEdit` 函数（第1210-1261行）替换为 `showAISessionRenameDialog(sessionId, currentTitle)`：

* 复用 `.new-notebook-overlay` / `.new-notebook-dialog` CSS 类（与笔记本重命名对话框一致）

* dialog 标题「重命名会话」，input 预填 `currentTitle`，自动聚焦+全选

* 按钮：取消 + 确认（居中显示）

* 确认逻辑：调用 `window.go.main.App.RenameAISession(sessionId, newTitle)` → 更新 `sessions` 数组 → `updateChatTitle()` + `renderSessionList()`

* 关闭：overlay 移除 + setTimeout 200ms 清理

* `_aiSessionRenameDialog` 变量跟踪实例，防止重复打开

### 2. JS — `ai-chat.js`：更新三个调用入口

* **侧栏双击**（第1333行）：`startInlineEdit(title, s.id)` → `showAISessionRenameDialog(s.id, s.title)`

* **更多菜单"重命名"**（第3359行）：`startInlineEdit(titleEl, s.id)` → `showAISessionRenameDialog(s.id, s.title)`

* **顶部标题栏双击**（第475行）：`startInlineEdit(aiChatTitleEl, activeSessionId)` → `showAISessionRenameDialog(activeSessionId, currentSessionTitle)`

### 3. JS — `ai-chat.js`：ESC 处理

AI 聊天的 ESC 处理在 `ai-chat.js` 第1011-1019行：

```javascript
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        if (sessionMoreMenu) closeSessionMenu();
        closeAiMsgContextMenu();
        if (refModal && refModal.style.display !== 'none') {
            closeNoteRefModal();
        }
    }
});
```

在此 ESC 链顶部新增：

```javascript
if (_aiSessionRenameDialog) { closeAISessionRenameDialog(); return; }
```

### 4. CSS — `ai-chat.css`：清理编辑态残留样式

删除不再需要的规则：

* `.ai-session-item-title:focus`（第2180行）— 无 contentEditable 了

* `.ai-session-item.editing .ai-session-item-title`（第2184行）— 不再用 editing 类

## 涉及文件

* `frontend/src/js/ai-chat.js` — 重写重命名函数 + 更新3处调用 + ESC分支

* `frontend/src/css/components/ai-chat.css` — 清理编辑态 CSS

## 验证

1. 侧栏双击会话 → 弹出对话框，input 预填标题
2. 更多菜单 → 点"重命名" → 弹出对话框
3. 顶部标题栏双击 → 弹出对话框
4. 确认 → 标题更新 + 侧栏重渲染
5. ESC / 点遮罩 / 点取消 → 对话框关闭
6. Enter → 确认重命名

