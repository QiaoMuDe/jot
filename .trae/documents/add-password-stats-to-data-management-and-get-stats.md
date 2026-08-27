# 计划：数据管理概览 + AI 统计工具新增密码记录统计

## 变更概览

在两处新增密码记录统计：① 数据管理页概览面板新增第 5 个信笺区块；② AI 助手 `get_stats` 工具 overview 输出新增一行密码统计。

## 当前状态

### 数据管理概览（4 个区块）

| 区块         | 统计项                                  |
| ---------- | ------------------------------------ |
| 📝 笔记与存储   | 笔记总数 / 笔记本数 / 标签数 / 回收站 / 数据库大小      |
| ✓ 待办事项     | 总数 / 已完成 / 完成率                       |
| 🤖 AI 统计   | 会话数 / 消息数 / Token / 平均等待/思考/最长等待（星级） |
| 🧠 AI 量化索引 | 向量索引笔记数 / 片段数 / 占用大小                 |

### AI 助手 `get_stats` 工具 overview 输出

```
数据统计概览：
笔记：共 X 篇（回收站 X，置顶 X）
笔记本：X 个
标签：X 个
待办：X 项（已完成 X）
AI 对话：X 个会话，X 条消息，累计 X tokens
响应耗时：平均 X 秒，思考平均 X 秒，最长 X 秒
数据库大小：X MB
向量索引：已量化 X 篇笔记，X 个片段，占用 X MB
```

**现状**：密码记录统计在两处均完全缺失。

***

## 改动清单

### 改动 1：`PasswordService` 新增 `Count()` 方法

**文件**：`internal/services/password_service.go`

在现有方法末尾新增：

```go
// Count 统计密码记录总数（不含已软删除）
func (s *PasswordService) Count() (int64, error) {
    var count int64
    err := s.db.Model(&PasswordRecord{}).Count(&count).Error
    return count, err
}
```

遵循 `TodoService.Count()` / `TagService.Count()` 的同一模式。

***

### 改动 2：`DataStats` 结构体新增字段

**文件**：`internal/services/types.go`

在 `CompletedTodos` 字段之后新增：

```go
TotalPasswords  int64   `json:"total_passwords"`
```

***

### 改动 3：`StatsService` 注入密码服务 + 填充字段

**文件**：`internal/services/stats_service.go`

**3a. 结构体新增字段**：

```go
type StatsService struct {
    note   *NoteService
    tag    *TagService
    todo   *TodoService
    pw     *PasswordService    // 新增
    ai     *AIService
    dbPath func() (string, error)
}
```

**3b.** **`NewStatsService`** **构造函数新增参数**：

```go
func NewStatsService(note *NoteService, tag *TagService, todo *TodoService, pw *PasswordService, ai *AIService, dbPath func() (string, error)) *StatsService
```

**3c.** **`GetDataStats()`** **方法内新增调用**（在 `s.todo.CountCompleted()` 之后）：

```go
if pwCount, err := s.pw.Count(); err == nil {
    stats.TotalPasswords = pwCount
}
```

遵循现有 error 容错模式（`if err == nil` 才赋值）。

***

### 改动 4：`App` 层创建 StatsService 时传入 PasswordService

**文件**：`app.go`

搜索 `NewStatsService(` 调用处，将 `a.passwordService`（或创建 PasswordService 的实例）作为新参数传入。

需要确认 `App` 结构体中 PasswordService 实例的变量名（可能是 `a.passwordService` 或 `a.pwService`）。如果 `App` 中尚无独立的 PasswordService 实例变量，需在 `initApp()` 或初始化链中创建。

***

### 改动 5：数据管理概览前端新增第 5 区块

**文件**：`frontend/src/js/data-management.js`

在 `loadDataStats()` 函数中：

**5a. 解构新字段**（在 `completedTodos` 附近）：

```javascript
const totalPasswords = stats.total_passwords || 0;
```

**5b. 在** **`bodyEl.innerHTML`** **模板中新增区块**（建议插在"待办事项"区块之后、"AI 统计数据"区块之前）：

```javascript
${totalPasswords > 0 ? `
  <div class="letter-section">
    <div class="letter-section-title">🔐 密码管理</div>
    <div class="letter-item">你共安全保管了 <b>${totalPasswords.toLocaleString()}</b> 条密码记录。</div>
  </div>
` : ''}
```

**5c. 更新空数据判断条件**（第 99 行）：

```javascript
// 原：totalNotes > 0 || aiMessages > 0 || totalTodos > 0
// 新增 || totalPasswords > 0
```

***

### 改动 6：AI 助手 `get_stats` 工具 overview 输出新增一行

**文件**：`internal/agent/tools/get_stats.go`

在 `statsOverview()` 方法中，"待办"行之后、"AI 对话"行之前新增：

```go
fmt.Fprintf(&sb, "密码记录：%d 条\n", stats.TotalPasswords)
```

同步更新工具 `Info` 描述文案，在"待办"之后加入"密码记录"。

***

## 涉及文件总览

| 文件                                      | 改动类型                               | 改动量    |
| --------------------------------------- | ---------------------------------- | ------ |
| `internal/services/password_service.go` | 新增 Count() 方法                      | \~5 行  |
| `internal/services/types.go`            | DataStats 加 1 个字段                  | 1 行    |
| `internal/services/stats_service.go`    | 注入 pw + 构造函数加参 + GetDataStats 加调用  | \~5 行  |
| `app.go`                                | NewStatsService 传入 passwordService | 1-3 行  |
| `frontend/src/js/data-management.js`    | 解构字段 + 新增 HTML 区块 + 空数据条件          | \~15 行 |
| `internal/agent/tools/get_stats.go`     | overview 输出加一行 + 更新 Info 描述        | \~5 行  |

总计约 **30-35 行改动**，零新文件。

***

## 验证步骤

1. **后端编译**：`go build ./...` 确认无编译错误
2. **数据管理概览**：打开数据管理 → 概览面板，确认第 5 区块"🔐 密码管理"出现，显示正确的密码记录数
3. **空密码场景**：删除所有密码后，确认概览面板不显示密码区块
4. **AI 统计工具**：对 AI 助手说"查看数据统计概览"，确认输出包含"密码记录：X 条"
5. **AI 月度统计**：确认 `month` action 不受影响

