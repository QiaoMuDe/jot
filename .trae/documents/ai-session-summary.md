# AI 会话持久化对话摘要方案

## 概述

当前 `TruncateMessagesForLLM` 使用简单滑动窗口：超过窗口大小（默认 40 条）的消息被直接丢弃，模型完全看不到早期对话。本方案通过**持久化对话摘要**让模型拥有"真正记忆"——早期消息压缩为摘要持久化存储，每次对话时注入摘要 + 保留近期完整消息，使模型既了解历史脉络又掌握近期细节。

***

## 当前状态分析

### 消息截断流程

```
CallAIAgentStream
  → truncateAIMessages(sessionID)           [app.go#L1906]
    → LoadAISessionMessages(sessionID)       [ai_service.go#L519]  ← 从 DB 加载全部消息
    → GetContextWindowSize() → 40            [ai_service.go#L114]  ← 窗口大小
    → TruncateMessagesForLLM(messages, 40)   [ai_service.go#L129]  ← 仅保留最后 40 条
  → 组装 HistoryMessage 传给 Agent.Run()
```

### 关键文件

| 文件                                                                                             | 相关函数/结构                                                     | 说明                  |
| ---------------------------------------------------------------------------------------------- | ----------------------------------------------------------- | ------------------- |
| [internal/models/ai\_session.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_session.go)     | `AISession`                                                 | 会话模型，将新增摘要字段        |
| [internal/services/ai\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go) | `TruncateMessagesForLLM`, `LoadAISessionMessages`, `CallAI` | 截断核心 + 摘要生成 AI 调用   |
| [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1906)                                              | `truncateAIMessages`                                        | 截断调用入口，修改点          |
| [internal/database/models.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/models.go)          | `AllModels`                                                 | 模型注册表，AISession 已注册 |

***

## 设计决策

| 决策     | 选择                                | 理由                          |
| ------ | --------------------------------- | --------------------------- |
| 摘要存储位置 | `AISession` 新增字段                  | 与会话一对一，查询方便，无需新增表           |
| 摘要更新策略 | 增量更新（当前摘要 + 新增消息 → 新摘要）           | 每次传入内容 O(N) 而非 O(N²)        |
| 更新触发时机 | 消息数超过窗口后，每新增 `windowSize/2` 条触发一次 | 平衡频率与质量，约每 20 条更新一次         |
| 摘要格式   | 纯文本，结构化要点列表                       | 与现有 `summarize_text` 工具风格一致 |
| 摘要注入方式 | 作为 system 消息插入到 tail 消息之前         | 无需改动 agent.go 消息处理逻辑        |
| 摘要失败处理 | 静默失败，回退到纯滑动窗口                     | 不影响对话可用性                    |
| 前端改动   | 零改动                               | 摘要对用户透明，不影响 UI 渲染           |
| 摘要生成模型 | 复用用户配置的 AI 模型（`AIService.CallAI`） | 无需额外配置，与用户当前模型一致            |
| 摘要生成时机 | 异步 goroutine，不阻塞当前对话              | 摘要生成耗时几秒到十几秒，阻塞会拖慢用户体验      |
| 前端反馈   | 通过 `ai:summary-status` 事件展示状态     | 用户看到"正在生成摘要…"提示，避免感知延迟      |

***

## 变更清单

### 1. `internal/models/ai_session.go` — 新增字段

```go
type AISession struct {
    ID            uint           `gorm:"primaryKey" json:"id"`
    Title         string         `gorm:"size:100;default:新对话" json:"title"`
    ContextTokens int            `gorm:"default:0" json:"context_tokens"`
    IsPinned      bool           `gorm:"default:false" json:"is_pinned"`
    SummaryContent string        `gorm:"type:text" json:"summary_content"`       // 新增：会话摘要文本
    SummaryMsgCount int          `gorm:"default:0" json:"summary_msg_count"`     // 新增：已摘要的非 system 消息数
    CreatedAt     time.Time      `json:"created_at"`
    UpdatedAt     time.Time      `json:"updated_at"`
    DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
```

* `SummaryContent`：摘要文本，空字符串表示无摘要

* `SummaryMsgCount`：已摘要的消息条数，用于判断是否需要更新摘要

* `gorm:"type:text"` 确保无长度限制

* GORM AutoMigrate 会自动新增这两列，无需手动迁移

### 2. `internal/services/ai_service.go` — 新增方法

#### 2a. `GenerateSessionSummary` — 生成摘要

```go
// GenerateSessionSummary 基于旧摘要 + 新增消息生成新的会话摘要。
// 调用方传入旧摘要（可能为空）和新增的 user/assistant 消息列表，
// 调用 AI 模型生成新的结构化要点摘要。
// 失败时返回空字符串，调用方回退到纯滑动窗口。
func (a *AIService) GenerateSessionSummary(ctx context.Context, oldSummary string, newMessages []Message) string {
    // 1. 构建摘要提示词
    prompt := buildSummaryPrompt(oldSummary, newMessages)

    // 2. 调用 AI 模型生成摘要
    summary, err := a.CallAI(ctx, []Message{{Role: "user", Content: prompt}})
    if err != nil {
        a.logger.Warnw("会话摘要生成失败，回退到纯滑动窗口", fastlog.Error(err))
        return ""
    }

    // 3. 清理：去除多余空白，截断到合理长度（最多 2000 字）
    summary = strings.TrimSpace(summary)
    runes := []rune(summary)
    if len(runes) > 2000 {
        summary = string(runes[:2000])
    }
    return summary
}

// buildSummaryPrompt 构建摘要生成提示词
func buildSummaryPrompt(oldSummary string, newMessages []Message) string {
    var b strings.Builder
    b.WriteString("你是一个对话摘要专家。请将以下对话内容压缩为结构化要点摘要。\n\n")
    b.WriteString("规则：\n")
    b.WriteString("- 提取核心信息：用户意图、关键决定、重要事实、偏好设定、行动项\n")
    b.WriteString("- 数字、日期、人名、术语必须准确，不得编造\n")
    b.WriteString("- 保留用户明确表达的偏好和设置（如语言偏好、常用格式等）\n")
    b.WriteString("- 输出为结构化要点列表，用小节标题组织\n")
    b.WriteString("- 只输出摘要本身，不要任何解释、开场白或结尾语\n\n")

    if oldSummary != "" {
        b.WriteString("【现有摘要】\n")
        b.WriteString(oldSummary)
        b.WriteString("\n\n")
        b.WriteString("【新增对话】\n")
    }

    for _, msg := range newMessages {
        role := "用户"
        if msg.Role == "assistant" {
            role = "助手"
        }
        b.WriteString(role + "：")
        // 只取前 2000 字符，避免超出模型上下文
        runes := []rune(msg.Content)
        if len(runes) > 2000 {
            b.WriteString(string(runes[:2000]))
            b.WriteString("…（过长已截断）")
        } else {
            b.WriteString(msg.Content)
        }
        b.WriteString("\n\n")
    }

    b.WriteString("请基于现有摘要和新增对话，生成更新后的完整摘要：")
    return b.String()
}
```

#### 2b. `UpdateSessionSummary` — 持久化摘要

```go
// UpdateSessionSummary 检查是否需要更新会话摘要，需要时生成并持久化。
// 当消息总数超过窗口大小且新增消息数达到阈值时触发。
// 返回是否实际更新了摘要。
func (a *AIService) UpdateSessionSummary(sessionID uint, windowSize int) bool {
    // 1. 加载会话记录
    var session models.AISession
    if err := a.db.First(&session, sessionID).Error; err != nil {
        return false
    }

    // 2. 统计非 system 消息总数
    var totalNonSystem int64
    a.db.Model(&models.AIMessage{}).
        Where("session_id = ? AND role != ?", sessionID, "system").
        Count(&totalNonSystem)

    // 3. 判断是否需要更新摘要
    //    条件：消息总数超过窗口大小，且自上次摘要以来新增了 windowSize/2 条
    threshold := windowSize / 2
    if threshold < 5 {
        threshold = 5 // 最小阈值 5 条
    }

    if totalNonSystem <= windowSize {
        return false // 未超窗口，不需要摘要
    }

    if int(totalNonSystem)-session.SummaryMsgCount < threshold {
        return false // 新增消息不足阈值，暂不更新
    }

    // 4. 计算需要摘要的消息范围：从上次摘要位置到"尾部保留区"之前
    //    保留尾部 windowSize/2 条作为完整消息，剩余的拿去生成摘要
    keepTail := windowSize / 2
    summarizeUpTo := int(totalNonSystem) - keepTail

    // 取上次摘要之后、本次需要摘要的消息
    var msgsToSummarize []models.AIMessage
    a.db.Where("session_id = ? AND role != ?", sessionID, "system").
        Order("created_at ASC").
        Offset(session.SummaryMsgCount).
        Limit(summarizeUpTo - session.SummaryMsgCount).
        Find(&msgsToSummarize)

    if len(msgsToSummarize) == 0 {
        return false
    }

    // 5. 转换为 Message 列表
    newMsgs := make([]Message, len(msgsToSummarize))
    for i, m := range msgsToSummarize {
        newMsgs[i] = Message{Role: m.Role, Content: m.Content}
    }

    // 6. 生成新摘要（含超时控制，最长 30s）
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    newSummary := a.GenerateSessionSummary(ctx, session.SummaryContent, newMsgs)
    if newSummary == "" {
        return false // 生成失败，静默回退
    }

    // 7. 持久化
    session.SummaryContent = newSummary
    session.SummaryMsgCount = summarizeUpTo
    if err := a.db.Model(&session).Updates(map[string]interface{}{
        "summary_content":  session.SummaryContent,
        "summary_msg_count": session.SummaryMsgCount,
    }).Error; err != nil {
        a.logger.Warnw("会话摘要持久化失败", fastlog.Error(err))
        return false
    }

    a.logger.Infow("会话摘要已更新",
        fastlog.Uint("session_id", sessionID),
        fastlog.Int("total_msgs", int(totalNonSystem)),
        fastlog.Int("summary_msg_count", summarizeUpTo))
    return true
}
```

### 3. `app.go` — 修改 `truncateAIMessages` + 异步摘要生成

在 `truncateAIMessages` 中，将摘要更新改为**异步 goroutine**，不阻塞当前对话流程。同时通过 Wails `runtime.EventsEmit` 向前端发送状态事件。

```go
// truncateAIMessages 加载并截断会话消息，保留 system 消息 + 最后 N 条 user/assistant 消息，
// 同时注入持久化会话摘要（如有）。摘要更新通过异步 goroutine 执行，不阻塞当前对话。
func (a *App) truncateAIMessages(sessionID uint, logLabel string) []services.Message {
    messages := a.aiService.LoadAISessionMessages(sessionID)

    nonSystemBefore := 0
    for _, m := range messages {
        if m.Role != "system" {
            nonSystemBefore++
        }
    }

    windowSize := a.aiService.GetContextWindowSize()

    // 异步触发摘要更新（不阻塞当前对话）
    // 下一轮对话时新摘要已就绪
    if nonSystemBefore > windowSize {
        a.triggerSummaryUpdate(sessionID, windowSize)
    }

    // 滑动窗口截断：保留 system + 最后 N 条
    messages = services.TruncateMessagesForLLM(messages, windowSize)

    // 读取已有会话摘要，注入到 system 消息之后、tail 消息之前
    var session models.AISession
    if err := a.db.First(&session, sessionID).Error; err == nil && session.SummaryContent != "" {
        summaryMsg := services.Message{
            Role:    "system",
            Content: "【历史对话摘要】\n" + session.SummaryContent,
        }
        insertIdx := 0
        for i, m := range messages {
            if m.Role == "system" {
                insertIdx = i + 1
            } else {
                break
            }
        }
        messages = append(messages[:insertIdx], append([]services.Message{summaryMsg}, messages[insertIdx:]...)...)
    }

    // 日志记录
    nonSystemAfter := 0
    hasSummary := false
    for _, m := range messages {
        if m.Role != "system" {
            nonSystemAfter++
        } else if strings.Contains(m.Content, "【历史对话摘要】") {
            hasSummary = true
        }
    }
    a.LogSvc.Logger.Debugw(logLabel,
        fastlog.Int("window_size", windowSize),
        fastlog.Int("non_system_before", nonSystemBefore),
        fastlog.Int("non_system_after", nonSystemAfter),
        fastlog.Int("total_after", len(messages)),
        fastlog.Bool("has_summary", hasSummary))
    return messages
}

// triggerSummaryUpdate 异步触发会话摘要更新（goroutine，不阻塞调用方）。
// 需要更新时向前端发送 ai:summary-status 事件。
func (a *App) triggerSummaryUpdate(sessionID uint, windowSize int) {
    go func() {
        // 发送"开始生成"事件
        runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
            "status":     "generating",
            "session_id": sessionID,
        })

        updated := a.aiService.UpdateSessionSummary(sessionID, windowSize)

        status := "done"
        if !updated {
            status = "skipped"
        }
        runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
            "status":     status,
            "session_id": sessionID,
        })

        if updated {
            a.LogSvc.Logger.Infow("会话摘要已更新（异步）",
                fastlog.Uint("session_id", sessionID))
        }
    }()
}
```

### 4. `frontend/src/js/ai-chat.js` — 新增摘要状态提示

在输入框上方显示摘要生成状态。前端监听 `ai:summary-status` 事件，展示"正在生成对话摘要…"提示。

```js
// 在 ai-chat.js 的全局事件监听区域追加

/** 会话摘要生成状态标志 */
let summaryGenerating = false;

// 监听摘要状态事件
window.runtime?.EventsOn('ai:summary-status', function(data) {
    if (!data || data.session_id !== activeSessionId) return;

    if (data.status === 'generating') {
        summaryGenerating = true;
        showSummaryStatus('正在生成对话摘要…');
    } else {
        summaryGenerating = false;
        hideSummaryStatus();
    }
});

/**
 * 在输入框上方显示摘要生成状态条
 */
function showSummaryStatus(text) {
    let el = document.getElementById('aiSummaryStatus');
    if (!el) {
        el = document.createElement('div');
        el.id = 'aiSummaryStatus';
        el.className = 'ai-summary-status';
        // 插入到输入框上方
        const inputArea = document.querySelector('.ai-input-area');
        if (inputArea) {
            inputArea.parentNode.insertBefore(el, inputArea);
        }
    }
    el.textContent = text;
    el.classList.add('active');
}

/**
 * 隐藏摘要生成状态条
 */
function hideSummaryStatus() {
    const el = document.getElementById('aiSummaryStatus');
    if (el) {
        el.classList.remove('active');
        // 动画结束后移除 DOM
        setTimeout(function() {
            if (el && el.parentNode) {
                el.parentNode.removeChild(el);
            }
        }, 300);
    }
}
```

### 5. `frontend/src/css/components/ai-chat.css` — 新增状态条样式

```css
/* 会话摘要生成状态条 */
.ai-summary-status {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 14px;
    margin: 4px 12px;
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
    color: var(--text-secondary);
    font-size: 0.75rem;
    opacity: 0;
    max-height: 0;
    overflow: hidden;
    transition: opacity 0.2s ease, max-height 0.2s ease, margin 0.2s ease, padding 0.2s ease;
}

.ai-summary-status.active {
    opacity: 1;
    max-height: 40px;
}

.ai-summary-status::before {
    content: '';
    width: 12px;
    height: 12px;
    border: 2px solid var(--accent);
    border-top-color: transparent;
    border-radius: 50%;
    animation: summarySpin 0.8s linear infinite;
    flex-shrink: 0;
}

@keyframes summarySpin {
    to { transform: rotate(360deg); }
}
```

### 6. 无需改动的文件

| 文件                            | 理由                                   |
| ----------------------------- | ------------------------------------ |
| `internal/database/models.go` | `AISession` 已注册，AutoMigrate 自动处理新增字段 |
| `internal/agent/agent.go`     | 摘要作为 system 消息注入，agent 无感知，零改动       |
| `internal/database/db.go`     | 无需种子数据                               |

***

## 核心逻辑时序

### 正常对话（消息未超窗口）

```
用户发送消息 → 入库 → 对话完成
  → truncateAIMessages
    → 消息总数 30 < 40，不触发摘要更新
    → 无摘要，原样返回全部消息
  → 发给模型
```

### 消息超过窗口（触发摘要更新）

```
消息达到第 41 条 → 对话完成
  → truncateAIMessages
    → 消息总数 41 > 40，触发 UpdateSessionSummary
      → 无旧摘要，SummaryMsgCount=0
      → 取消息 0~20（41-20=21 之前的消息）
      → 生成摘要_1，存库，SummaryMsgCount=21
    → TruncateMessagesForLLM 保留最后 40 条（消息 1~40）
    → 读取摘要_1，注入到 system 之后
    → 最终：system + [摘要_1] + 消息 1~40
  → 发给模型
```

### 增量更新摘要

```
消息达到第 61 条 → 对话完成
  → truncateAIMessages
    → 消息总数 61 > 40，触发 UpdateSessionSummary
      → 已有摘要_1，SummaryMsgCount=21
      → 61 - 21 = 40 >= 20（阈值），需要更新
      → 保留尾部 20 条（消息 41~60），摘要到消息 40
      → 取消息 21~40（20 条）
      → 生成摘要_2：[摘要_1 + 消息 21~40] → 新摘要
      → 存库，SummaryMsgCount=40
    → TruncateMessagesForLLM 保留最后 40 条（消息 21~60）
    → 读取摘要_2，注入到 system 之后
    → 最终：system + [摘要_2] + 消息 21~60
  → 发给模型
```

### 摘要生成失败

```
生成摘要超时/报错 → GenerateSessionSummary 返回 ""
  → UpdateSessionSummary 返回 false
  → truncateAIMessages 查不到摘要 → 回退到纯滑动窗口
  → 行为与当前完全一致，不影响可用性
```

***

## 验证步骤

1. 新建会话，发送 30 条消息 → 检查摘要未被触发（`SummaryMsgCount` 仍为 0）
2. 继续发送到 41 条 → 检查摘要是否生成（`SummaryContent` 非空，`SummaryMsgCount` > 0）
3. 继续发送到 61 条 → 检查摘要是否增量更新（`SummaryContent` 变化，`SummaryMsgCount` 增加）
4. 检查 `truncateAIMessages` 日志中的 `has_summary` 字段
5. 检查对话中模型是否了解早期信息（如"你还记得我最早说过什么吗？"）
6. 断开网络后对话 → 摘要生成失败 → 回退到纯滑动窗口，对话正常进行

