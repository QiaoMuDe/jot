# Agent 前后端交互事件协议

> 记录 AI 助手模块后端 → 前端（Wails `runtime.EventsEmit` / `EventsOn`）的交互事件协议。
> 后端发射方：[app.go](app.go)（`CallAIAgentStream` / `UpdateSessionSummary`）、[agent.go](internal/agent/agent.go)（`Run`）、[tools/ask_user.go](internal/agent/tools/ask_user.go)、[tools/plan.go](internal/agent/tools/plan.go)；前端监听方：[ai-chat.js](frontend/src/js/ai-chat.js)。
> 事件名统一前缀 `ai:`；Agent 相关事件统一携带 `streamGen`（代际标识，前端按代过滤防串流）。

---

## 1. 事件总览

| 事件名 | 触发时机 | 负载要点 | 前端行为 |
|--------|---------|---------|---------|
| `ai:stream-chunk` | 流式正文分块 | `content`（增量文本） | 追加到气泡正文 |
| `ai:stream-thinking` | 深度思考分块 | `reasoning_content`（增量思考文本） | 追加到思考折叠区 |
| `ai:stream-done` | 流正常结束 / 取消 | `content`、耗时、token、`userMsgID`/`assistantMsgID` | 落库 + 收尾清理 |
| `ai:stream-error` | 流错误 | `error` JSON、token 估算 | 展示错误态 |
| `ai:tool-status` | 工具调用各阶段 | `tools.Record` JSON（`tool_start`/`tool_result`/`tool_error`/`tool_partial`） | 状态条 + 历史明细 |
| `ai:ask-user` | 模型发起反问 | `{question, options, selection}` JSON | 弹出反问面板并阻塞等待 |
| `ai:plan-generating` | Plan 模式预规划 LLM 调用期间 | 空字符串 | 显示计划生成状态文案（轮换文案，重试不额外通知） |
| `ai:plan-created` | `create_plan` 调用成功 / 预规划完成 | `{goal, steps}` JSON | 弹出计划面板 |
| `ai:plan-updated` | `update_plan` 调用成功 / 结果兜底 | `{step_id, status, result, steps}` JSON | 刷新计划面板 |
| `ai:agent-result` | Agent 结果汇总 | `RecallCards`、`ToolCalls`、`Plan`、`ReasoningContent` | 随 `stream-done` 落库渲染 |
| `ai:summary-status` | 会话摘要生成中/完成/跳过 | `{status, session_id}` | 状态条提示 |

---

## 2. 流式输出事件（`ai:stream-chunk` / `ai:stream-thinking` / `ai:stream-done` / `ai:stream-error`）

- `ai:stream-chunk`：参数 `(streamGen, content)`。Go 端逐块解析 SSE `delta.content`，经回调转事件推送（[agent.go](internal/agent/agent.go) 事件循环）。
- `ai:stream-thinking`：参数 `(streamGen, reasoning_content)`。仅深度思考模型返回，Go 端解析 `delta.reasoning_content`。
- `ai:stream-done`：参数 `(streamGen, content, elapsedThinking, elapsedTotal, totalTokens, userTokens, assistantTokens, userMsgID, assistantMsgID)`。
  - **正常完成**（[app.go](app.go) `CallAIAgentStream` 汇总处）：携带最终正文、耗时与 token 统计、落库后的 `userMsgID`/`assistantMsgID`。
  - **取消路径**（停止按钮）：参数为 `(streamGen, "", 0.0, 0.0, 0, 0, 0, 0, 0)`，`assistantMsgID == 0`；前端据此区分"取消"与"正常完成"，取消不写入 `chatHistory`（避免幽灵条目）。
- `ai:stream-error`：参数 `(streamGen, errorJSON, estimateUserTokens)`。前端展示错误并收尾。

---

## 3. 工具状态事件（`ai:tool-status`）

负载为 `tools.Record` 的 JSON（[context.go](internal/agent/tools/context.go)），Action 字段区分阶段：

| Action | 含义 | Record 字段 |
|---|---|---|
| `tool_start` | 模型决定调用 | `name`、`args`（截断）、`action_text`（动作文案，由工具实现 `ActionTextProvider` 提供） |
| `tool_result` | 工具执行成功 | `name`、`result`（截断） |
| `tool_error` | 工具执行失败（回填模型） | `name`、`result`（错误文本截断） |
| `tool_partial` | 部分失败提示（前端 ⚠️） | `name`、`result`（失败说明） |

父包逻辑见 [agent.go](internal/agent/agent.go)（`emitToolStart` / `emitToolResult`，注意 `emitToolResult` 会检查"最近一条同名记录是否为 `tool_error`"，失败态不会被 result 覆盖）与 [context.go](internal/agent/tools/context.go)（`DrainPartials` 统一以 `tool_partial` 发射）。

---

## 4. 反问交互事件（`ai:ask-user`）

`ask_user` 工具执行时发射，负载为 JSON 字符串 `{"question": "...", "options": ["...", ...], "selection": "single"|"multiple"}`（`options` 为空数组 `[]`，`selection` 缺省 `"single"`）。

- 前端收到后在输入区上方渲染**悬浮反问面板**（`#aiAskPanel`）：问句标题（右上角 × 关闭按钮 = 取消本轮）+ 选项区 + 自定义输入行。
- **同轮传输（AskWaiter）**：工具先 `ClaimAsk()` 原子抢占反问名额（模型并行发多条 ask_user 仅第一条成功），再发射事件并阻塞等待用户回答（ReAct 循环暂停、AI 消息不结束）；答案经 `AnswerAskUser(sessionID, answer)` 投递到会话等待通道，作为工具结果返回给模型继续完成原始请求——不落库为新 user 消息、不新开一轮。
- `selection` 语义：`single` 单选（点选项即提交）；`multiple` 多选（勾选多项后点"确认提交"）。
- 面板生命周期：`showAskPanel` / `hideAskPanel`；提交成功隐藏；× 关闭 = 取消本轮（复用停止逻辑）；切换会话/清空会话/`stream-done`/`stream-error`/停止时隐藏。
- 与计划面板互斥（方案 B）：`showAskPanel` 先收起计划面板，`hideAskPanel` 后若仍在流式中且有计划数据则恢复。
- 未注入 AskWaiter（非交互场景/测试）时不发射事件，直接返回引导文本。
- 落库保障：反问轮以本轮全部流式正文（问句 + 续答）作为 `result.Content` 落库，与前端同一气泡展示一致。

详见 [ask_user.go](internal/agent/tools/ask_user.go) 与 [context.go](internal/agent/tools/context.go)（`AskWaiter` / `ClaimAsk` / `WaitForAnswer`）。

---

## 5. 规划事件（`ai:plan-generating` / `ai:plan-created` / `ai:plan-updated`）

`create_plan` / `update_plan` 是允许工具内部直接 `ctx.Emit` 事件的例外（与 `ask_user` 并列），用于向前端展示执行计划卡片。这两个工具同时也会产生标准的 `ai:tool-status` 事件（`tool_start` / `tool_result`），规划事件是额外的独立通道。

### 5.0 `ai:plan-generating`（Plan-and-Exec 预规划状态）

Plan 模式下，[agent.go](internal/agent/agent.go) `Run()` 在调用 `generatePlan()`（单独 LLM 调用生成执行计划）前发射此事件，通知前端预规划阶段开始。首次负载为空字符串。

- 前端收到后将打字动画（`createTypingDots`）替换为"正在制定执行计划…"状态文案（`.ai-msg-plan-generating`），持续显示轮换文案。
- **重试机制**：`generatePlan()` 内部在解析/校验失败时自动重试（最多 3 次），重试期间不再额外发射此事件，前端始终保持轮换文案（重试进度仅记录后端日志）。
- `generatePlan()` 完成后由 `ai:plan-created` 事件接替渲染计划面板；所有重试均失败时由 `ai:stream-error` 接替展示错误。
- Agent 模式下不发射此事件。

### 5.1 `ai:plan-created`

`create_plan` 工具调用成功后发射；**Plan-and-Exec 预规划阶段**（`generatePlan()` 成功）也会发射同样的事件。负载为 JSON 字符串：

```json
{
  "goal": "分析用户关于 Rust 内存管理的提问",
  "steps": [
    {"id": 1, "description": "搜索本地笔记中关于 Rust 的内容", "status": "pending"},
    {"id": 2, "description": "搜索网络最新资料", "status": "pending"},
    {"id": 3, "description": "综合笔记和搜索结果回答用户", "status": "pending"}
  ]
}
```

- `goal`：计划目标描述（字符串）
- `steps`：步骤列表，每项含 `id`（1-based 编号）、`description`（步骤描述）、`status`（初始均为 `"pending"`）

### 5.2 `ai:plan-updated`

`update_plan` 工具调用成功后发射；**结果兜底**（模型漏调 `update_plan` 时 [agent.go](internal/agent/agent.go) 汇总处自动补标未完成步骤）也会发射。负载为 JSON 字符串：

```json
{
  "step_id": 1,
  "status": "done",
  "result": "找到 3 篇相关笔记",
  "steps": [
    {"id": 1, "description": "搜索本地笔记中关于 Rust 的内容", "status": "done", "result": "找到 3 篇相关笔记"},
    {"id": 2, "description": "搜索网络最新资料", "status": "in_progress"},
    {"id": 3, "description": "综合笔记和搜索结果回答用户", "status": "pending"}
  ]
}
```

- `step_id`：被更新的步骤编号（`null` 表示新增步骤）
- `status`：更新后的状态（`"pending"` / `"in_progress"` / `"done"` / `"skipped"`）
- `result`：步骤执行结果摘要（可为空串）
- `steps`：完整步骤列表快照（前端据此刷新计划卡片）

### 5.3 前端消费要点

- 负载**不含 `goal` 字段**：前端需将增量合并到已有数据（`Object.assign({}, streamPlanData, payload)`），直接覆盖会丢失标题。
- `ai:tool-status` 用于状态条展示（与其他工具一致），规划事件用于渲染计划面板，两类事件需同时处理。

---

## 6. 结果汇总事件（`ai:agent-result`）

Agent 最终结果汇总时由 [app.go](app.go) `CallAIAgentStream` 发射，参数 `(streamGen, RecallCards, ToolCalls, Plan, ReasoningContent)`，随后紧接正常路径的 `ai:stream-done`。

- `RecallCards`：`services.RecallCard` 数组（`recall_notes` 本地向量召回卡片，前端 `renderRecallCards` 展示，历史回放同）。
- `ToolCalls`：工具调用链（前端折叠明细）。
- `Plan`：本轮执行计划 JSON（`Result.Plan`，历史回放 `renderPlanCard` 渲染；为 `null` 表示无计划）。
- `ReasoningContent`：思考链。

---

## 7. 摘要状态事件（`ai:summary-status`）

会话摘要生成状态，由 [app.go](app.go) `UpdateSessionSummary` 路径发射，参数为 map：

```json
{"status": "generating" | "done" | "skipped", "session_id": 123}
```

- `generating`：开始生成（同步阻塞，当前轮对话即可用新摘要）。
- `done`：生成成功；`skipped`：本次无需更新。
- 前端 `summaryGenerating` 状态控制"正在生成对话摘要…"提示，取消流时重置。

---

## 8. 其他事件（非 Agent 链路）

- `ai:aiop-chunk` / `ai:aiop-done` / `ai:aiop-error`：AI 一站式处理（AIOP）链路的流式推送与终态通知（fire-and-forget，无返回值），参数 `(streamGen, content)` 等。
- 非事件类：工具元信息（`GetAgentTools`）、会话/消息 CRUD 等走 Wails 方法调用，不经事件通道。
