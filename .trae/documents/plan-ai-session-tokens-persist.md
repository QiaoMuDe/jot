# 计划：AI 会话 Token 数持久化到数据库

## 概要

将会话的上下文 Token 数保存到 `ais_sessions` 表中，切换会话时重新加载并显示。前端每次计算完 Token 后自动更新数据库。

## 现状分析

* **后端**：`AISession` 模型只有 ID、Title、时间戳字段，没有 Token 计数字段。

* **前端**：`estimateTokens()` 本地估算 Token 数，`updateContextSize()` 每次从 `chatHistory` 重新计算并更新到 DOM，计算结果不持久化。

* **切换会话**：`switchSession()` 从库加载消息 → 渲染 → 调用 `updateContextSize()` 重新计算并显示。

## 变更方案

### 1. 后端：会话模型加字段

**文件**：`internal/models/ai_session.go`

在 `AISession` 结构体中新增 `ContextTokens` 字段：

```go
type AISession struct {
    ID            uint           `gorm:"primaryKey" json:"id"`
    Title         string         `gorm:"size:100;default:新对话" json:"title"`
    ContextTokens int            `gorm:"default:0" json:"context_tokens"`  // ← 新增
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
```

GORM AutoMigrate 会自动新增列，已有的会话默认值为 0。

### 2. 后端：新增更新方法

**文件**：`internal/services/ai_service.go`

新增 `UpdateSessionContextTokens` 方法：

```go
func (a *AIService) UpdateSessionContextTokens(sessionID uint, tokens int) error {
    return a.db.Model(&models.AISession{}).Where("id = ?", sessionID).Update("context_tokens", tokens).Error
}
```

同时将 `ContextTokens` 字段加入 `AISessionSummary` 结构体，以便会话列表接口一并返回：

```go
type AISessionSummary struct {
    ID            uint   `json:"id"`
    Title         string `json:"title"`
    ContextTokens int    `json:"context_tokens"`
    LastMessage   string `json:"last_message"`
    MessageCount  int    `json:"message_count"`
    CreatedAt     string `json:"created_at"`
    UpdatedAt     string `json:"updated_at"`
}
```

`GetAISessions()` 中构造 summary 时填入 `ContextTokens: session.ContextTokens`。

### 3. 后端：暴露 API 绑定

**文件**：`app.go`

新增绑定方法：

```go
func (a *App) UpdateSessionContextTokens(sessionID uint, tokens int) error {
    return a.aiService.UpdateSessionContextTokens(sessionID, tokens)
}
```

### 4. 前端：切换会话时从数据库加载 Token 数

**文件**：`frontend/src/js/ai-chat.js`

需要做三处改动。

#### 4a. 存储会话的 Token 数到查找表

`renderSessionList()` 调用 `GetAISessions()` 拿到所有会话时，把每个会话的 `context_tokens` 存入一个全局对象 `window._sessionTokens`（key 为 session ID）：

```javascript
// 在 renderSessionList() 中，已有 const sessions = await window.go.main.App.GetAISessions();
// 在遍历 sessions 渲染之前添加：
window._sessionTokens = {};
sessions.forEach(s => { window._sessionTokens[s.id] = s.context_tokens || 0; });
```

#### 4b. `switchSession()` 加载 Token 并显示（不再重算）

`switchSession(id)` 中不再调用 `updateContextSize()`，改为直接从查找表加载数据库已保存的 Token 数并显示。新会话或 Token 为 0 时隐藏显示。

```javascript
async function switchSession(id) {
    // ... 现有清理逻辑（清空引用、笔记缓存等，约第 1141-1148 行）

    activeSessionId = id;
    const msgs = await window.go.main.App.LoadAISessionMessages(id);

    // ... 现有重建 chatHistory 逻辑

    // 渲染消息（现有）
    // ...

    // ★ 替换原 updateContextSize() 调用：从数据库加载 Token 数直接显示
    const savedTokens = window._sessionTokens?.[id] || 0;
    const el = document.getElementById('aiChatContextSize');
    if (el) {
        if (savedTokens > 0) {
            el.style.display = '';
            el.textContent = formatTokens(savedTokens) + ' tokens';
        } else {
            el.style.display = 'none';
            el.textContent = '';
        }
    }

    // 现有：刷新侧栏、滚动到底部、聚焦输入框
    renderSessionList();
    scrollToBottom();
    inputEl?.focus();
}
```

注意：`switchSession()` 不再调用 `updateContextSize()`，因为 DB 值就是上次保存的精确值，不需要重算。`updateContextSize()` 仅在消息变化时（发送、接收回复、清空、再生、新建会话）才被调用，调用时仍然会重新计算并更新 DB。

#### 4c. `updateContextSize()` 改为 async 并写入数据库

`updateContextSize()` 函数末尾追加数据库写入调用。修改后：

```javascript
async function updateContextSize() {
    let total = 0;
    for (const msg of chatHistory) {
        total += estimateTokens(msg.content || '');
    }
    const el = document.getElementById('aiChatContextSize');
    if (!el) return;
    if (total === 0) {
        el.style.display = 'none';
        el.textContent = '';
    } else {
        el.style.display = '';
        el.textContent = formatTokens(total) + ' tokens';
    }
    // 持久化到数据库
    if (activeSessionId && total > 0) {
        try {
            await window.go.main.App.UpdateSessionContextTokens(activeSessionId, total);
        } catch (e) {
            console.warn('Failed to save context tokens:', e);
        }
    }
}
```

注意：`updateContextSize()` 改为 `async`，所有调用处不需要调整（`await` 是可选的，调用方不关心返回值）。`switchSession()` 中调用 `updateContextSize()` 也不需要 await。

## 涉及文件

| 文件                                | 改动类型                                                                                                    |
| --------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `internal/models/ai_session.go`   | 新增 `ContextTokens` 字段                                                                                   |
| `internal/services/ai_service.go` | 新增 `UpdateSessionContextTokens()` 方法 + 更新 `AISessionSummary`                                            |
| `app.go`                          | 新增 `UpdateSessionContextTokens()` 绑定                                                                    |
| `frontend/src/js/ai-chat.js`      | `updateContextSize()` 改为 async 并写入数据库 + `switchSession()` 从查找表加载 Token 显示 + `renderSessionList()` 填充查找表 |

## 验证

1. 启动应用，打开 AI 对话页面，确认 Token 数正常显示
2. 发送几条消息，确认 Token 数更新
3. 切换会话，确认 Token 数显示正确（切换到有历史的会话时，Token 数立刻从数据库加载显示）
4. 重新打开应用，切换到之前的会话，确认 Token 数和上次离开时一致

