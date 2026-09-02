# AI 助手上下文摘要机制优化：Token 预算窗口 + 摘要边界持久化 + 轮次对齐

## Summary

将 AI 对话的上下文窗口与摘要触发机制从「按消息条数」改为「按 token 预算」：

* **A（Token 预算窗口）**：新增设置 `ai_context_token_budget`（默认 128K = 131072 token），tail 消息按累计估算 token 选取；移除条数窗口 `ai_context_window_size` 及其兜底逻辑。摘要触发条件改为 tail 估算 token 达到预算的 **80%**。

* **C（摘要边界持久化）**：`AISession` 新增 `SummaryUpToMsgID` 字段，摘要覆盖边界按消息 ID 持久化，不再用 `SummaryMsgCount - keepTail` 反推增量起点。存量旧数据（`SummaryUpToMsgID=0 && SummaryMsgCount>0`）按「旧摘要 + 全量历史」一次性重摘要（与正常增量路径天然统一，无需特殊分支）。

* **D（轮次对齐）**：tail 与压缩后保留区均以 user 消息开头，避免 LLM 看到半轮对话。

摘要仍为同步生成（保留 `ai:summary-status` 事件与前端提示，前端零改动）。

## Current State Analysis

现有实现：

| 位置                                                                                               | 内容                                                                                           |
| ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- |
| [ai\_service.go#L116-L129](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L116-L129) | `GetContextWindowSize`：读 `ai_context_window_size`（默认 20，clamp 2\~200），无前端 UI                 |
| [ai\_service.go#L131-L160](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L131-L160) | `TruncateMessagesForLLM`：保留 system + 最后 N 条（可能切断一轮）                                          |
| [ai\_service.go#L162-L225](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L162-L225) | `GenerateSessionSummary` / `buildSummaryPrompt`：递归摘要，单条截 500 rune，摘要上限 2000 rune             |
| [ai\_service.go#L226-L308](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go#L226-L308) | `UpdateSessionSummary`：条数阈值判断 + `offset = SummaryMsgCount - keepTail` 反推增量起点                 |
| [app.go#L1936-L1946](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1936-L1946)                              | `estimateTokens`：中文/1.5 + 其他/4 估算算法（前端有同款）                                                   |
| [app.go#L1965-L2047](file:///d:/峡谷/Dev/本地项目/jot/app.go#L1965-L2047)                              | `truncateAIMessages`：编排入口，Agent 流（L2068）与 Chat 流（L2356）共用；含 `needUpdate` 与 service 内部判断的重复逻辑 |
| [ai\_session.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_session.go)                       | `SummaryContent` / `SummaryMsgCount` 字段，AutoMigrate 管理表结构                                    |
| [db.go#L221](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go#L221)                            | `ai_context_window_size` 默认值种子                                                               |
| [EVENTS.md §7](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/EVENTS.md)                              | `ai:summary-status` 事件文档                                                                     |

关键事实：

* 每条消息落库的 `Tokens` 字段**不可直接用于窗口计算**：user 消息的 Tokens 事后会被覆写为整轮完整 prompt token 数（[app.go#L2226](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2226)），不是该消息自身内容体量。窗口计算必须用 `estimateTokens(Content)` 现算。

* 摘要边界若用条数反推，与可变设置强耦合，且新方案下窗口大小与消息条数无固定关系，反推逻辑必须废弃。

## Proposed Changes

### 核心设计：不变量与触发规则

* **不变量**：tail 之前的所有消息均已包含在会话摘要中（边界 = tail 起点）。

* **每轮发送前**：

  1. 全量加载消息，从尾部按轮次对齐选取 tail，使 tail 累计估算 token ≤ `budget`（128K）。
  2. 若 `tailTokens ≥ 0.8 × budget`（触发阈值 102.4K）→ 压缩：从 tail 头部向后保留最近 ≤ `0.5 × budget`（64K）的轮次，将 `[旧 tail 起点 .. 新 tail 起点)` 区间与旧摘要合并生成新摘要，边界推进到新 tail 起点。
  3. 摘要注入为「【历史对话摘要】」system 消息，位于 system 之后、tail 之前。

* **边界兼容**：`SummaryUpToMsgID=0` 且 `SummaryContent` 非空（存量会话）→ 等价于「边界为 0」，重摘要时自然走「旧摘要 + 全量历史」路径，无需特判。

* **防超限兜底**：待摘要区间 token 超过 40K 时，仅取区间末尾约 40K token 的消息参与摘要生成（prompt 中注明「更早对话已包含在现有摘要中」），防止单次摘要调用输入超模型上下文。

* **极端保护**：始终保留最后一条 user 消息及其后的消息；单条消息超预算时 tail 允许超出（不做切分），压缩跳过。

### 常量定义（services 包）

```go
const (
    DefaultContextTokenBudget = 131072 // 128K
    MinContextTokenBudget     = 4096
    MaxContextTokenBudget     = 1048576
    SummaryTriggerRatio       = 0.8  // tail 达预算 80% 触发压缩
    CompactKeepRatio          = 0.5  // 压缩后保留最近 50% 预算的 tail
    SummaryRegionTokenCap     = 40000 // 单次摘要输入的区间 token 上限
    MaxSummaryRunes           = 2000  // 摘要长度上限（沿用现状）
)
```

### 1. [internal/models/ai\_session.go](file:///d:/峡谷/Dev/本地项目/jot/internal/models/ai_session.go)

新增字段（AutoMigrate 自动加列，无需手工迁移）：

```go
SummaryUpToMsgID uint `gorm:"default:0" json:"summary_up_to_msg_id"` // 摘要已覆盖到的最后一条消息 ID（不含）；0 表示未摘要或存量旧数据
```

`SummaryMsgCount` 保留列但停止使用（不加删除迁移，旧数据无害残留）。

### 2. [internal/services/ai\_service.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_service.go)（核心，建议新文件 `ai_context.go` 承载）

**删除**：`GetContextWindowSize`、`TruncateMessagesForLLM`、`UpdateSessionSummary`。
**改造保留**：`GenerateSessionSummary` / `buildSummaryPrompt`（签名与内部逻辑沿用，输入消息来源变化）。

**新增**：

```go
// EstimateTokens 估算文本 token（算法与原 app.go.estimateTokens 一致：中文/1.5 + 其他/4）
func EstimateTokens(text string) int

// GetContextTokenBudget 读设置 ai_context_token_budget，默认 131072，clamp [4096, 1048576]
func (a *AIService) GetContextTokenBudget() int

// SelectTailByTokenBudget 从消息尾部按预算选取 tail：
//  - 从后往前累计 EstimateTokens(Content)，直到加入下一条会超预算
//  - 轮次对齐：tail 首条必须是 user 消息（边界落在 assistant 时继续丢弃直至 user）
//  - 始终保留最后一条消息（单条超预算时 tail 允许超出预算）
//  - 只处理 user/assistant 消息，system 消息原样保留在调用方处理
// 返回 (tail 消息, tail 在原切片中的起始下标)
func SelectTailByTokenBudget(messages []Message, budget int) ([]Message, int)

// CompactSessionSummary 压缩会话摘要：
//  - 加载 session（取 SummaryContent / SummaryUpToMsgID）
//  - 输入为待摘要区间消息 [startIdx..endIdx)（调用方已按边界与 tail 切好）
//  - 区间 token 超 SummaryRegionTokenCap 时只取末尾部分
//  - 调用 GenerateSessionSummary（旧摘要 + 区间消息）生成新摘要
//  - 成功后持久化 summary_content + summary_up_to_msg_id（单次 Updates）
// 返回是否更新成功
func (a *AIService) CompactSessionSummary(ctx context.Context, sessionID uint, msgs []Message, newBoundaryMsgID uint) bool
```

### 3. [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go)

**`estimateTokens`（L1936）**：改为委托 `services.EstimateTokens`，避免算法双份维护（本文件多处调用点不动）。

**`truncateAIMessages`（L1965-L2047）重写**：

```
1. messages := LoadAISessionMessages(sessionID)
2. budget := GetContextTokenBudget()
3. system 消息单独收集；其余消息 SelectTailByTokenBudget(messages, budget) → tail, tailStartIdx
4. tailTokens := Σ EstimateTokens(tail.Content)
5. if tailTokens >= SummaryTriggerRatio × budget：
     a. 从 tail 头部计算保留区（≤ CompactKeepRatio × budget，同样轮次对齐到 user，
        至少保留最后一轮），得到 newTailStartIdx（在全量消息中的下标）
     b. 待摘要区间 = 全量消息中 [boundaryPos .. newTailStartIdx)，
        boundaryPos = SummaryUpToMsgID==0 ? 区间起点0 : 按消息 ID 匹配到的下标
     c. 发送 ai:summary-status generating → 调 CompactSessionSummary(ctx, sessionID, 区间, 区间最后一条消息ID)
        → 按结果发送 done / skipped
     d. 压缩成功后重算 tail = 消息[newTailStartIdx..]（当前轮即用新边界）
6. 组装：system 消息 + 【历史对话摘要】system 消息（若 SummaryContent 非空）+ tail
7. Debugw 日志改为 token 口径：budget / tail_tokens / tail_msgs / summary_chars / has_summary / compacted
```

注意：`ai:summary-status` 事件负载与发射时机保持不变（generating → done/skipped），**前端** **[ai-chat.js#L231](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L231)** **零改动**。原 app.go 中 `needUpdate` 预判与 service 内部阈值判断的重复逻辑随重写消除（触发判断只存在于 truncateAIMessages 一处）。

### 4. [internal/database/db.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go)

* 默认值列表：`ai_context_window_size` 行删除（存量行残留无害），新增：

```go
{Key: "ai_context_token_budget", Value: "131072"},
```

### 5. [internal/agent/EVENTS.md](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/EVENTS.md) §7

更新触发语义描述：条数阈值 → tail token 达预算 80% 触发压缩；事件名、负载、前端行为不变。

### 6. 新增测试 [internal/services/ai\_context\_test.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/ai_context_test.go)

* `EstimateTokens`：中文 / 英文 / 混合样例断言

* `SelectTailByTokenBudget`：

  * 预算内全保留

  * 超预算截断且边界对齐 user 消息

  * 单条超预算仍保留最后一条

  * system 消息不参与累计

* `CompactSessionSummary`（sqlite 内存库）：首次摘要 / 增量压缩 / 边界持久化断言

## Assumptions & Decisions

| 决策点                     | 结论                                            | 依据                     |
| ----------------------- | --------------------------------------------- | ---------------------- |
| 条数兜底                    | 完全移除，不做双保险                                    | 用户明确选择                 |
| 预算默认值                   | 131072（128K），设置 key `ai_context_token_budget` | 用户明确指定                 |
| 触发阈值                    | 80% 为常量 `SummaryTriggerRatio`，不做成设置项          | 避免过度配置化；后续要开放成本极低      |
| 压缩保留量                   | 50% 预算（常量），给回复生成留 headroom                    | 衍生设计，计划内定死             |
| 摘要同步生成                  | 保留（B 方案未选），事件机制不变                             | 用户方案选择                 |
| 摘要消息 Tokens 覆写逻辑（L2226） | 不动，与窗口计算无关                                    | 现算 Content token 避开该问题 |
| 旧 SummaryMsgCount 列     | 保留不删，代码不再引用                                   | 无害残留，避免破坏性迁移           |
| 存量会话兼容                  | 边界=0 + 旧摘要作底 + 全量历史重摘要，与增量路径统一，无特判            | 用户选择「一次性全量重摘要」         |
| 超大摘要输入                  | 区间 >40K token 时取末尾 40K                        | 防止摘要调用自身超模型上下文         |

## Verification

1. `go build ./...`、`go vet ./...` 通过。
2. `go test ./internal/services/ -run TestEstimate -run TestSelectTail -run TestCompact` 新增测试通过。
3. 手动验证（运行应用）：

   * 新会话短对话：日志 `tail_tokens < 80% budget`，无摘要事件，消息原样发送。

   * 粘贴超长内容（估算 >102K token）后发消息：触发 `ai:summary-status generating → done`，日志 `compacted=true`，回复正常且上下文含摘要。

   * 打开一个有旧 `SummaryMsgCount` 的存量长会话发消息：确认走「旧摘要 + 全量历史」重摘要且 `summary_up_to_msg_id` 落库。

   * 中途把 `ai_context_token_budget` 调小后继续对话：边界不错位（按消息 ID 推进），无漏摘要/重复摘要。

   * 检查 tail 首条始终为 user 消息（日志或 DB 断点验证）。

