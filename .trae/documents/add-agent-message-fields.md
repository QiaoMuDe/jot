# AI Agent 消息字段拆分：search\_sources / recall\_cards / 新增 tool\_calls

## Summary

解决 Agent 模式与问答模式共用 `ai_messages.search_sources` 字段导致的格式冲突（问答存 `SearchSource[]`、Agent 存 `toolCallRecord[]`，前端按来源结构渲染时错乱），并补齐 Agent 模式"召回卡片未落库"的问题。最终三个字段各司其职：

* `search_sources`：联网搜索来源（`[]SearchSource`），问答与 Agent 的 `web_search` 都写这里

* `recall_cards`：召回卡片（`[]RecallCard`），问答与 Agent 的 `recall_notes` 都写这里

* `tool_calls`（**新增**）：Agent 工具调用链（`[]toolCallRecord`），仅 Agent 写

## Current State Analysis

* `ai_messages` 表（`internal/models/ai_message.go`）仅有 `search_sources`、`recall_cards` 两个 text 字段，无工具调用专用字段

* Agent 模块（`internal/agent/`）当前把工具调用链 `toolCallRecord[]`（action/name/args/result）塞进 `Result.SearchSources` → app.go 落库 `search_sources`，与问答模式的 `SearchSource[]`（source\_label/url/title）结构冲突

* Agent 的 `recall_notes` 工具结果不落 `recall_cards`，历史消息无召回卡片

* 前端 `renderMultiSourcesPanel` 按 SearchSource 结构解析 → Agent 历史消息来源面板错乱

* 关键可用资源：`services.SearchWebResult.Sources []SearchSource` 与 `CardRecallResult.Cards []RecallCard` 均为结构化数据，Agent 工具执行时可直接拿到并收集

## Proposed Changes

### 后端

**1. 数据模型加字段**

* `internal/models/ai_message.go`：`AIMessage` 加 `ToolCalls string \`gorm:"type:text" json:"tool\_calls"\`\`（AutoMigrate 自动加列）

* `internal/services/ai_service.go` L21-31：`Message` 加 `ToolCalls string \`json:"tool\_calls"\`\`

**2. 字段透传（5 处转换）** — `internal/services/ai_service.go`

* `LoadAISessionMessages`（L650）：`ToolCalls: m.ToolCalls`

* `LoadAISessionMessagesPaginated`（L668）：`ToolCalls: m.ToolCalls`

* `SaveAIMessage`（L807）：`ToolCalls: msg.ToolCalls`

* `SaveAIMessages`（L853）：`ToolCalls: msg.ToolCalls`

* `ReplaceAISessionMessages`（L915）：`ToolCalls: msg.ToolCalls`

* 用 Grep 搜索 `AIMessage{` / `Message{` 在 services 包的其他转换处，逐一补齐

**3. Agent 模块收集结构化结果** — `internal/agent/`

* `types.go`：`Result` 改为三字段：

  ```go
  type Result struct {
      Content       string // 最终回答全文
      SearchSources string // 联网搜索来源 JSON（[]services.SearchSource，web_search 工具结果）
      RecallCards   string // 召回卡片 JSON（[]services.RecallCard，recall_notes 工具结果）
      ToolCalls     string // 工具调用链 JSON（[]toolCallRecord）
  }
  ```

* `tools.go`：

  * 新增内部收集器 `type resultCollector struct { Sources []services.SearchSource; Cards []services.RecallCard }`

  * `webSearchTool` 加字段 `collector *resultCollector`；`InvokableRun` 收集成功后 `w.collector.Sources = append(w.collector.Sources, r.result.Sources...)`

  * `recallNotesTool` 加字段 `collector *resultCollector`；`InvokableRun` 收集成功后 `r.collector.Cards = append(r.collector.Cards, result.Cards...)`

* `agent.go` `Run`：

  * 创建 `collector := &resultCollector{}`，构造两个工具时传入

  * 汇总：`result.SearchSources` = marshal(collector.Sources)（空则空串）；`result.RecallCards` = marshal(collector.Cards)；`result.ToolCalls` = marshal(toolRecords)（沿用现有 toolCallRecord 逻辑）

**4. app.go 落库** — `CallAIAgentStream`（约 L2590-2595）

```go
assistantMsg := services.Message{
    Role:          "assistant",
    Content:       result.Content,
    Tokens:        assistantTokens,
    SearchSources: result.SearchSources, // web_search 来源
    RecallCards:   result.RecallCards,   // 召回卡片（修复未存储问题）
    ToolCalls:     result.ToolCalls,     // 工具调用链
}
```

### 前端

**5. 消息对象透传** — `frontend/src/js/ai-chat.js`

* L1573 与 L1673 的 `chatHistory` map：加 `tool_calls: msg.tool_calls || null`

* `addMessage`（L2859）签名加 `toolCalls` 参数（放在 `recallCards` 之后）：解析 JSON（同 searchSources 模式），assistant 且有内容时调 `renderToolCalls(el, toolCalls)`

* 两处 `addMessage(...)` 调用（L1606 / L1657）在 `msg.recall_cards` 后透传 `msg.tool_calls`

**6. 工具调用链历史渲染（新函数）**

* 新增 `renderToolCalls(el, toolCalls)`：渲染折叠面板（复用现有 `ai-tool-status-list / ai-tool-status-item / ai-tool-status-icon / ai-tool-status-text` 样式与 `.is-done` 态），按 `tool_start`/`tool_result` 配对展示"🔍 正在搜索：xxx → ✓ 已获取 N 条结果"，可复用 `getToolName`、`parseField` 工具函数

**7. 流式完成时收集工具记录**

* `ai:tool-status` 监听（L2517-2529）：把 payload push 进新增数组 `streamToolRecords`（流开始置空）

* stream-done 完成分支 `chatHistory.push(...)`（L2584）加 `tool_calls: streamToolRecords.length > 0 ? JSON.stringify(streamToolRecords) : null`（与 search\_sources 落法一致），使当前气泡重新加载/后续操作不丢失工具链

* `search_sources`/`recall_cards` 在流式端不做额外收集（Agent 后端已落库，历史加载时从 `msg.search_sources` / `msg.recall_cards` 渲染；流式气泡内的来源/卡片如需即时展示，可延后——本期以"重开会话正确渲染"为验收）

### 数据兼容说明

* 存量问答模式消息的 `search_sources` 格式不变，渲染不受影响

* 存量 Agent 消息（极少，开发期产生）的 `search_sources` 内是旧 toolCallRecord 数据，新代码下来源面板可能显示异常——可接受，不迁移

## Assumptions & Decisions

* **决策**：新增 `tool_calls` 字段存储工具调用链，`search_sources` 回归纯搜索来源、`recall_cards` 回归纯召回卡片（用户已确认）

* **决策**：Agent 的 `web_search` / `recall_notes` 工具执行时收集结构化结果（`Sources` / `Cards`），由 `AgentService.Run` 汇总进 `Result`，实现与问答模式一致的来源/卡片落库

* 不做存量数据迁移；不修改问答模式 `CallAIStream` 行为

* 前端流式气泡内的工具状态展示（已有）保持不变；新增的是历史消息加载时的折叠面板

## Verification

1. `go build ./...`、`go vet ./...` 通过；`node --check frontend/src/js/ai-chat.js` 通过
2. AutoMigrate 后 `ai_messages` 表出现 `tool_calls` 列（`go run` 启动日志 / 数据库查看）
3. 静态检查：`services.Message`↔`models.AIMessage` 转换处均透传 `tool_calls`（Grep 复核无遗漏）
4. 手工冒烟（`wails dev`）：

   * Agent 模式提问触发 `web_search` → 回答完成后重开会话：来源面板按 SearchSource 正常渲染（无错乱）

   * Agent 模式提问触发 `recall_notes` → 重开会话：召回卡片面板正常渲染（recall\_cards 已落库）

   * Agent 模式历史消息显示工具调用链折叠面板

   * 问答模式提问（搜索+召回）→ 重开会话：来源/卡片渲染与改造前完全一致（无回归）

