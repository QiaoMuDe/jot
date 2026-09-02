# AI 上下文窗口与会话摘要机制

本文档记录 AI 助手的上下文构建、token 预算窗口与会话摘要压缩的工作机制与算法。

核心实现：[ai_context.go](ai_context.go)（算法）+ [app.go `truncateAIMessages`](../../app.go)（编排入口）。

---

## 1. 总览

每轮对话（Agent 流 / Chat 流）发送给 LLM 前，`truncateAIMessages` 会重新构建上下文：

```
[system 消息...]                       ← 身份提示词、技能、引用上下文等，原样保留
[system] 【历史对话摘要】...            ← 老对话的压缩表示（存在摘要时）
[tail: user/assistant 消息]            ← 最近的原文对话，≤ token 预算
```

核心不变量：

> **tail 之前的所有消息，其内容一定已包含在会话摘要中。**

上下文体量由「token 预算」控制，而非消息条数——长消息提前触发压缩，短消息延迟触发，LLM 实际收到的上下文体量稳定可预期。

---

## 2. 设置与常量

### 设置项（settings 表）

| 键 | 默认值 | 说明 |
|---|---|---|
| `ai_context_token_budget` | `131072`（128K） | 上下文 token 预算，clamp 范围 [4096, 1048576]，无前端 UI |
| `ai_context_summary_trigger_ratio` | `0.8` | 摘要压缩触发比例（tail 达预算该比例时压缩），clamp 范围 [0.05, 1.0]，无前端 UI；调小便于测试压缩流程 |

读取入口：`GetContextTokenBudget()` 与 `GetContextSummaryTriggerRatio()`（[ai_context.go](ai_context.go)）。旧设置键 `ai_context_window_size` 已废弃，由 [db.go `cleanupOrphanedData`](../database/db.go) 在启动时从存量库删除。

### 算法常量（ai_context.go）

| 常量 | 值 | 说明 |
|---|---|---|
| `DefaultSummaryTriggerRatio` | `0.8` | 触发比例默认值（运行时读设置键，见上表） |
| `MinSummaryTriggerRatio` / `MaxSummaryTriggerRatio` | `0.05` / `1.0` | 触发比例 clamp 范围 |
| `CompactKeepRatio` | `0.5` | 压缩后保留最近 50% 预算的 tail |
| `SummaryRegionTokenCap` | `40000` | 单次摘要生成的输入区间 token 上限 |
| `MaxSummaryRunes` | `2000` | 摘要文本长度上限（rune） |

---

## 3. Token 估算算法

`EstimateTokens(text)`（与前端 `estimateTokens` 算法保持一致）：

```
token 数 = ceil(中文字符数 / 1.5 + 其他字符数 / 4)
```

中文字符判定：`unicode.Is(unicode.Han, r)`。

> **注意**：`AIMessage.Tokens` 落库字段**不可**用于窗口计算——user 消息的 Tokens 在回复完成后会被覆写为整轮完整 prompt 的 token 数（统计口径，见 app.go `UpdateAIMessageTokens` 调用处），并非该消息自身的体量。窗口计算一律使用 `EstimateTokens(Content)` 现算。

---

## 4. Tail 选取算法（窗口截断）

`SelectTailByTokenBudget(messages, budget)`，输入为**摘要边界之后**的 user/assistant 消息切片（调用方先用 `SummaryUpToMsgID` 定位边界，边界之前的内容已由摘要覆盖、不参与窗口计算——否则压缩后 raw tail 仍接近满预算，会导致每轮重复触发摘要）：

1. **从尾部向前累计**每条消息的 `EstimateTokens(Content)`；
2. 加入下一条会超出 `budget` 时停止（**最后一条消息无条件保留**，单条超预算时 tail 允许超出，不做切分）；
3. **轮次对齐**：若截断边界落在 assistant 消息上，向后丢弃直至 tail 首条为 user 消息（被丢弃的 assistant 回复由摘要覆盖）；
4. 返回 `(tail 切片, tail 在原切片中的起始下标)`。

效果：LLM 看到的 tail 永远从一轮完整对话的 user 消息开始，不会出现"半轮对话"。

---

## 5. 摘要压缩算法

### 5.1 触发条件

```
tailTokens = Σ EstimateTokens(tail[i].Content)
触发条件：tailTokens ≥ budget × 触发比例（ai_context_summary_trigger_ratio，默认 0.8）
```

每轮发送前判断一次；仅触发时才向前端发 `ai:summary-status` 事件（避免空事件）。

> **⚠️ 事件派发约束**：`truncateAIMessages`（含压缩与事件发射）必须运行在 goroutine 中
> （[app.go `CallAIAgentStream`](../../app.go) 已如此）。Wails 绑定方法返回前发出的
> `EventsEmit` 不会实时送达前端，会积压到方法返回后才派发——放在方法体内会导致
> `generating` 状态条延迟到压缩结束才出现，UI 全程无反馈。

### 5.2 压缩动作

```
tail（≥ 预算 × 触发比例）
┌─────────────────────┬──────────────────┐
│ 待摘要区间（送去压缩） │ 保留区（≤ 50% 预算）│
└─────────────────────┴──────────────────┘
```

1. `SelectKeepTailByTokenBudget(tail, budget, 0.5)`：从 tail 尾部向前累计，保留最近 ≤ 50% 预算的轮次（同样轮次对齐到 user，至少保留最后一条）；
2. **待摘要区间** = `[摘要边界之后 .. 新 tail 起点)`，即 tail 中被丢弃的旧部分（含边界与旧 tail 起点之间的任何未摘要消息）；
3. `CompactSessionSummary`：旧摘要 + 待摘要区间消息 → 调用 AI 生成新摘要（递归式增量摘要，见 §6）；
4. 成功后单次 `Updates` 持久化 `summary_content` + `summary_up_to_msg_id`（新边界的消息 ID）；
5. 当前轮立即用新 tail（保留区）+ 新摘要构建上下文，无需等下一轮。

压缩失败（AI 调用失败 / 超时）时**中止本轮对话**：不调用 LLM，向前端发 `ai:summary-status failed` + `ai:stream-error`（提示"对话摘要生成失败，请重新发送消息"）。用户主动取消（ctx 已取消）不发 `failed` 事件，由调用方按取消语义收尾（`ai:stream-done`）。用户重新发起对话时 tail 仍 ≥ 预算 × 触发比例，会再次触发摘要重试。

### 5.3 摘要边界持久化

- `AISession.SummaryUpToMsgID`：摘要已覆盖到的**最后一条消息 ID**（不含）。
- 增量起点按消息 ID 在消息列表中定位，而非条数反推——**中途修改预算设置不会造成边界错位**（旧条数方案 `SummaryMsgCount - keepTail` 的核心缺陷）。
- 不变量维持：压缩后 `SummaryUpToMsgID` = 新 tail 起点的前一条消息 ID，tail 之前的内容全部在摘要中。

### 5.4 存量数据兼容

`SummaryUpToMsgID == 0` 的会话（新会话或旧版本存量会话）等价于"边界在起点"：

- 新会话：`SummaryContent` 为空，首次压缩即生成首份摘要；
- 存量会话：`SummaryContent` 非空（旧条数口径生成），压缩时自然走「旧摘要作底 + 全量历史重摘要」路径，**与增量路径代码完全统一，无特殊分支**；成功后写入新边界，之后进入正常节奏。

### 5.5 摘要输入体量保护

待摘要区间 token 超过 `SummaryRegionTokenCap`（40K）时，`limitRegionByTokens` 只取区间**末尾**约 40K 的消息参与生成（更早内容已由旧摘要覆盖）。配合提示词中的固定说明：

> （更早的对话未逐条列出，其要点已包含在现有摘要中，请勿遗漏其中的关键信息。）

防止递归摘要调用自身的输入超出模型上下文。

---

## 6. 摘要生成 Prompt

`GenerateSessionSummary` / `buildSummaryPrompt`（ai_context.go）：

- **角色**：对话摘要专家，输出结构化要点列表（小节标题组织）；
- **提取规则**：用户意图、关键决定、重要事实、偏好设定、行动项；数字/日期/人名/术语必须准确；保留用户明确表达的偏好；
- **输入**：`【现有摘要】`（若有）+ `【新增对话】`（逐条"用户：/助手："）；
- **单条消息截断**：超过 500 rune 截断并标注"（过长已截断）"；
- **输出截断**：摘要超过 `MaxSummaryRunes`（2000 rune）裁剪；
- **失败语义**：返回空字符串，调用方沿用旧摘要（不阻塞对话，仅延迟压缩）。

已知取舍：递归式摘要在超长会话中会逐渐损失早期细节（"传话游戏"效应），通过"旧摘要作底 + 关键信息保留指令"缓解；如需更强保真可演进为分层摘要。

---

## 7. 数据模型

`AISession`（[ai_session.go](../models/ai_session.go)）相关字段：

| 字段 | 说明 |
|---|---|
| `SummaryContent` | 会话摘要文本（text） |
| `SummaryUpToMsgID` | 摘要覆盖边界的消息 ID；0 = 未摘要或存量旧数据 |
| `ContextTokens` | 会话累计 token 统计（展示口径，与窗口机制无关） |

旧列 `summary_msg_count` 已从模型移除，由 `cleanupOrphanedData` 幂等删列。

---

## 8. 前端事件

`ai:summary-status`（负载 `{"status": "generating" | "done" | "failed", "session_id"}`）：

- 仅在压缩触发的轮次发射：`generating` →（同步生成）→ `done` / `failed`；
- `failed` 表示生成失败，本轮对话被后端中止（不调用 LLM），紧随其后发射 `ai:stream-error`（前端通知"对话摘要生成失败，请重新发送消息"并解锁输入）；用户重新发起对话时再次触发摘要；
- 前端据此显示"正在生成对话摘要…"状态条，取消 AI 流时摘要生成一并取消（共享同一个 ctx）并重置状态；
- 详见 [EVENTS.md §7](../agent/EVENTS.md)。

---

## 9. 日志观测

`truncateAIMessages` 每轮输出 Debug 日志（Agent 流 / Chat 流各有 logLabel）：

| 字段 | 含义 |
|---|---|
| `budget` | 本次生效的 token 预算 |
| `tail_tokens` | tail 估算 token 总量 |
| `tail_start` | tail 在非 system 消息中的起始下标 |
| `tail_msgs` | tail 消息条数 |
| `has_summary` | 是否注入了摘要 |
| `compacted` | 本轮是否执行了压缩 |

压缩成功时另有 Info 日志：`session_id` / `summary_up_to_msg_id` / `region_msgs`。

---

## 10. 边界情况

| 场景 | 行为 |
|---|---|
| 单条消息超预算（如粘贴超长文章） | tail 仅含该消息并允许超预算，不做切分；可正常发送 |
| 压缩后保留区仍 ≥ 预算 × 触发比例（单轮极大） | 触发但区间为空（`newTailStart == boundaryPos`）时跳过压缩，事件不发 |
| 压缩失败（AI 失败 / 超时 / 用户取消） | 中止本轮对话：发 `failed` + `ai:stream-error`，不调用 LLM；用户重新发起时再次触发摘要 |
| 会话记录不存在（db 查询失败） | 无摘要注入，tail 照常选取 |
| 中途调小 `ai_context_token_budget` | 边界按消息 ID 推进，无错位；tail 立即按新预算变短，压缩在 tail ≥ 新预算 × 触发比例时触发 |
| 存量长会话首次触发 | 旧摘要 + 全量历史一次性重摘要（受 40K 区间上限保护），之后进入常规节奏 |

---

## 11. 相关测试

[ai_context_test.go](ai_context_test.go)：

- `TestEstimateTokens`：中/英/混合/空文本边界；
- `TestSelectTailByTokenBudget`：预算内全保留、超预算截断 + 轮次对齐、单条超预算；
- `TestSelectKeepTailByTokenBudget`：压缩保留区切分 + 对齐；
- `TestLimitRegionByTokens`：摘要输入区间上限；
- `TestCompactSessionSummary`：首次摘要 / 增量推进边界 / AI 失败沿用旧摘要（httptest 模拟 OpenAI 兼容端点走全链路）。
