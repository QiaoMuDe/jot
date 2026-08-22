# AI 会话摘要方案调整：窗口 20 条，每 20 条更新一次

## 概述

将上下文窗口从 40 条改为 20 条，摘要更新策略从"每 10 条"改为"每 20 条"。模型永远看到 `[摘要]` + 最近 20 条完整消息。

---

## 参数变更

| 参数 | 当前值 | 改后值 | 含义 |
|------|--------|--------|------|
| `windowSize` | 40 | 20 | 保留最近 N 条完整消息 |
| `keepTail` | `windowSize/2` = 10 | `windowSize` = 20 | 尾部保留数 |
| `threshold` | `windowSize/2` = 10 | `windowSize` = 20 | 每新增 N 条触发一次摘要更新 |

---

## 变更清单

### 1. `internal/services/ai_service.go` — `UpdateSessionSummary`

改动两处：

**threshold 计算**：[L236](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L236)

```go
// 改前
threshold := windowSize / 2

// 改后
threshold := windowSize
```

**keepTail 计算**：[L250](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L250)

```go
// 改前
keepTail := windowSize / 2

// 改后
keepTail := windowSize
```

### 2. `app.go` — `truncateAIMessages`

当前每次 `nonSystemBefore > windowSize` 都发出 `generating` 事件，即使不需要实际更新（消息 22~40 轮）。需加守卫判断，只有真正需要更新时才发事件。

改动位置：[L1919-L1940](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1919)

```go
// 改前
if nonSystemBefore > windowSize {
    runtime.EventsEmit(a.ctx, "ai:summary-status", ...)
    updated := a.aiService.UpdateSessionSummary(ctx, sessionID, windowSize)
    ...
    runtime.EventsEmit(a.ctx, "ai:summary-status", ...)
}

// 改后
if nonSystemBefore > windowSize {
    // 先检查是否真的需要更新（diff >= windowSize），避免每次发空事件
    var checkSession models.AISession
    needUpdate := false
    if err := a.db.First(&checkSession, sessionID).Error; err == nil {
        needUpdate = nonSystemBefore-checkSession.SummaryMsgCount >= windowSize
    }

    if needUpdate {
        runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
            "status":     "generating",
            "session_id": sessionID,
        })
    }

    updated := a.aiService.UpdateSessionSummary(ctx, sessionID, windowSize)

    if needUpdate {
        status := "done"
        if !updated {
            status = "skipped"
        }
        runtime.EventsEmit(a.ctx, "ai:summary-status", map[string]interface{}{
            "status":     status,
            "session_id": sessionID,
        })
    }
}
```

---

## 时序推演

### 消息 1~20：无摘要

```
总消息 ≤ 20 → truncateAIMessages 条件不满足 → 全部消息发给模型
```

### 消息 21：首次触发

```
nonSystemBefore = 21 > 20 → 进入条件
  diff = 21 - 0 = 21 ≥ 20 → needUpdate = true
  → 发 "generating" 事件
  → UpdateSessionSummary:
    threshold = 20, keepTail = 20
    summarizeUpTo = 21 - 20 = 1
    offset = 0 - 20 = 0（归零）
    limit = 1 - 0 = 1
    取消息[0] 1 条 → 摘要_1
    SummaryMsgCount = 21
  → 发 "done" 事件
  → TruncateMessagesForLLM 保留最后 20 条（消息 1~20）
  → 注入摘要_1

模型看到：system + [摘要_1] + 消息 1~20 + 消息 21
```

### 消息 22~40：无事件

```
nonSystemBefore > 20 → 进入条件
  diff = N - 21 < 20 → needUpdate = false
  → 不发事件
  → UpdateSessionSummary 内部返回 false（不更新）
  → 不发事件
  → 注入已有摘要_1

模型看到：system + [摘要_1] + 最近 20 条
```

### 消息 41：增量更新

```
nonSystemBefore = 41 > 20 → 进入条件
  diff = 41 - 21 = 20 ≥ 20 → needUpdate = true
  → 发 "generating" 事件
  → UpdateSessionSummary:
    summarizeUpTo = 41 - 20 = 21
    offset = 21 - 20 = 1
    limit = 21 - 1 = 20
    取消息[1..20] 20 条 → [摘要_1 + 消息 1~20] → 摘要_2
    SummaryMsgCount = 41
  → 发 "done" 事件
  → 注入摘要_2

模型看到：system + [摘要_2(覆盖1~20)] + 消息 21~40 + 消息 41
```

### 消息 61：再次增量

```
diff = 61 - 41 = 20 ≥ 20 → needUpdate = true
  summarizeUpTo = 61 - 20 = 41
  offset = 41 - 20 = 21
  limit = 41 - 21 = 20
  取消息[21..40] 20 条 → [摘要_2 + 消息 21~40] → 摘要_3
  SummaryMsgCount = 61

模型看到：system + [摘要_3(覆盖1~40)] + 消息 41~60 + 消息 61
```

---

## 无需改动的文件

| 文件 | 理由 |
|------|------|
| `internal/database/db.go` | 种子值已改为 20 |
| `internal/models/ai_session.go` | 模型字段不变 |
| `frontend/` 全部 | 前端状态条逻辑不变 |
| `internal/agent/` 全部 | 摘要作为 system 消息注入，agent 无感知 |