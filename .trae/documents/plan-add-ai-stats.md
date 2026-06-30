# 计划：在数据管理页面增加 AI 会话数 / AI 消息数统计

## 概要

在数据管理页面的统计卡片区增加 2 张新卡片，分别显示 AI 会话总数和 AI 消息总数，样式、动画和 UI 沿用现有的统计卡片体系。

---

## 现状分析

### 后端
- **DataStats 结构体**（[types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go#L12-L20)）：目前有 `TotalNotes`, `TrashedNotes`, `PinnedNotes`, `TotalTags`, `TotalNotebooks`, `DBSize`, `DBSizeStr` 7 个字段。
- **GetDataStats**（[app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L233-L261)）：从 `noteService.GetStats()` + `tagService.Count()` + `os.Stat` 组装统计数据。
- **AIService**（[ai_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go)）：`App` 结构体已有 `aiService` 字段，可直接调用。目前有 `GetAISessions()`、`CreateAISession()` 等方法，但无独立的计数方法。
- **AISession/AIMessage 模型**：详见 [ai_session.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_session.go)（软删除支持 `DeletedAt`）、[ai_message.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_message.go)。

### 前端
- **HTML**（[index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html#L586-L620)）：5 张 `.stat-card` 卡片，分别展示笔记总数、标签总数、回收站、笔记本数、数据库大小。
- **CSS**（[data-view.css](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/css/components/data-view.css#L43-L79)）：`.data-stats` 使用 `grid-template-columns: repeat(5, 1fr)`；`.stat-card` 使用 `cardEnter` 动画（`translateY(20px) scale(0.97)` → 原位）；hover 边框/阴影变化；窄屏 ≤640px 切为 `repeat(2, 1fr)`。
- **JS**（[data-management.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/data-management.js#L43-L93)）：`loadDataStats()` 从 `GetDataStats` 获取数据 → 交错卡片入场动画（`cardEnter`，每张延时 80ms）→ count-up 数字递增动画。共 5 张卡片，5 个 DOM 引用（`statTotalNotes` 等）。
- **DOM 引用**（[main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L357-L361)）：5 个 stat 元素的 `els` 引用。

---

## 变更清单

### 1. 后端：`internal/services/types.go` — DataStats 新增字段

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| `AISessions` | `int64` | `ai_sessions` | AI 会话总数（不计软删除） |
| `AIMessages` | `int64` | `ai_messages` | AI 消息总数（所有会话） |

```go
type DataStats struct {
    // ... 现有字段不变
    AISessions  int64  `json:"ai_sessions"`
    AIMessages  int64  `json:"ai_messages"`
}
```

### 2. 后端：`internal/services/ai_service.go` — 新增计数方法

新增两个方法（参照已有的 `GetAISessions` 中查消息数的模式）：

```go
// CountSessions 获取 AI 会话总数（不含软删除）
func (a *AIService) CountSessions() (int64, error) {
    var count int64
    err := a.db.Model(&models.AISession{}).Where("deleted_at IS NULL").Count(&count).Error
    return count, err
}

// CountMessages 获取 AI 消息总数
func (a *AIService) CountMessages() (int64, error) {
    var count int64
    err := a.db.Model(&models.AIMessage{}).Count(&count).Error
    return count, err
}
```

### 3. 后端：`app.go` — GetDataStats 接入 AI 统计

在 `GetDataStats` 函数中，`return stats` 之前插入：

```go
aiSessions, _ := a.aiService.CountSessions()
aiMessages, _ := a.aiService.CountMessages()
stats.AISessions = aiSessions
stats.AIMessages = aiMessages
```

使用 `_` 忽略错误，与现有风格一致（数据库不存在时 `Count` 返回 0）。

### 4. 前端：`frontend/index.html` — 新增 2 张统计卡片

在第 5 张卡片（数据库大小）之后，`.data-stats` 容器内末尾追加：

```html
<div class="stat-card">
    <div class="stat-body">
        <div class="stat-value" id="statAISessions">0</div>
        <div class="stat-label">AI 会话</div>
    </div>
</div>
<div class="stat-card">
    <div class="stat-body">
        <div class="stat-value" id="statAIMessages">0</div>
        <div class="stat-label">AI 消息</div>
    </div>
</div>
```

### 5. 前端：`frontend/src/main.js` — 新增 DOM 引用

在 `els` 对象中（`statDBSize` 之后）追加：

```javascript
statAISessions: $('statAISessions'),
statAIMessages: $('statAIMessages'),
```

### 6. 前端：`frontend/src/js/data-management.js` — 更新 loadDataStats

#### 6a. 变量声明区追加（~line 45）

```javascript
let aiSessions = 0, aiMessages = 0;
```

#### 6b. 从 stats 取值（~line 54）

```javascript
aiSessions = stats.ai_sessions || 0;
aiMessages = stats.ai_messages || 0;
```

#### 6c. 重置区追加（~line 80）

```javascript
els.statAISessions.textContent = '0';
els.statAIMessages.textContent = '0';
```

#### 6d. count-up 调用区追加（~line 88）

```javascript
animateCountUp(els.statAISessions, aiSessions);
animateCountUp(els.statAIMessages, aiMessages);
```

### 7. 前端：`frontend/src/css/components/data-view.css` — 调整网格列数

将 `.data-stats` 的 `grid-template-columns` 从 `repeat(5, 1fr)` 改为 `repeat(7, 1fr)`：

```css
.data-stats {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 12px;
}
```

保留窄屏 ≤640px 的 `repeat(2, 1fr)` 覆盖规则不变。

---

## 涉及文件汇总

| 文件 | 改动类型 | 说明 |
|------|----------|------|
| `internal/services/types.go` | 修改 | DataStats 新增 `AISessions`、`AIMessages` |
| `internal/services/ai_service.go` | 修改 | 新增 `CountSessions()`、`CountMessages()` |
| `app.go` | 修改 | `GetDataStats()` 调用 AI 计数 |
| `frontend/index.html` | 修改 | 新增 2 张 stat-card |
| `frontend/src/main.js` | 修改 | els 新增 2 个 DOM 引用 |
| `frontend/src/js/data-management.js` | 修改 | `loadDataStats()` 处理 AI 统计 |
| `frontend/src/css/components/data-view.css` | 修改 | grid 从 5 列改为 7 列 |

---

## 验证步骤

1. `go build .` 编译通过
2. 打开数据管理页面，统计卡片区出现 7 张卡片（原有 5 张 + AI 会话 + AI 消息）
3. 卡片入场动画正常工作（交错 `cardEnter` + count-up 递增）
4. 数值与数据库中实际数量一致
5. 窄屏 ≤640px 响应式布局正常（`repeat(2, 1fr)` 显示 7 张卡片）
6. 如果不存在 AI 会话/消息，显示 0
7. 数据库瘦身、导入数据后调用 `loadDataStats()`，AI 统计同步更新