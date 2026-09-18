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
| `ai:tool-approval` | 工作目录危险操作（write_file 覆盖 / edit_file 编辑 / run_command 执行 / transfer_file 下载到桌面或覆盖上传 / delete_file 删除）请求审批 | `{tool, summary, approval_id, critical}` JSON | 弹出审批面板并阻塞等待（回调 `ApproveToolCall`） |
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

### 3.1 子 Agent 内层步骤转发（os_agent）

文件/命令工具（read_file / write_file / edit_file / ls_dir / glob / grep_file / copy_file / move_file / delete_file / mkdir_dir / run_command）已封装为 `os_agent` 子 Agent（[subagent_os.go](internal/agent/subagent_os.go)，通用机制见 [subagent.go](internal/agent/subagent.go)），父层仅注册 os_agent 一个委托工具。内层每步工具调用以 `ai:tool-status` 事件**实时转发**（`tool_start` / `tool_result` / `tool_error` / `tool_partial` 语义与父层一致），写入父层同一 `toolRecords` 切片，顺序相邻：

```
os_agent tool_start → 内层工具 tool_start / tool_result / ... → os_agent tool_result
```

- 前端以「os_agent start → 内层记录 → os_agent result」为分组边界（状态条 / 历史明细按此顺序渲染）。
- 内层流式正文与思考链**不转发**：子 Agent 只把最后一条非工具正文作为结果，经 os_agent `tool_result` 透传给父层。
- 内层工具审批语义与父层一致：同一会话 `Approver`（`agentSession.RequestApproval`），危险操作照常发射 `ai:tool-approval`（见 §5）阻塞确认，前端审批面板零改动。

---

## 4. 反问交互事件（`ai:ask-user`）

`ask_user` 工具执行时发射，负载为 JSON 字符串 `{"questions": [{"question": "...", "options": ["...", ...], "selection": "single"|"multiple"}, ...], "question": "...", "options": [...], "selection": "..."}`。主格式为 `questions` 数组（1-3 条，每题可独立设置 `options` 与 `selection`，`options` 为空数组 `[]`、`selection` 缺省 `"single"`）；旧顶层字段（`question`/`options`/`selection`）取首条问题的值，仅作兼容保留。

- 前端收到后在输入区上方渲染**悬浮反问面板**（`#aiAskPanel`）：单问题时问句标题（右上角 × 关闭按钮 = 取消本轮）+ 选项区 + 自定义输入行；多问题时表单式渲染 N 个问题分组（编号题目标题 + 各题选项 + 各题自定义输入），底部唯一"确认提交"按钮逐题收集后一次性提交。
- **同轮传输（AskWaiter）**：工具先 `ClaimAsk()` 原子抢占反问名额（模型并行发多条 ask_user 仅第一条成功），再发射事件并阻塞等待用户回答（ReAct 循环暂停、AI 消息不结束）；答案经 `AnswerAskUser(sessionID, answer)` 投递到会话等待通道，作为工具结果返回给模型继续完成原始请求——不落库为新 user 消息、不新开一轮。
- **答案提交格式约定**：单问题 = 原始答案文本；多问题 = 每题一行（`答案1\n答案2...`，输入框单行、多选拼接为一行，行内无换行），后端按行数与问题数逐题映射；行数不匹配时整体作为单条答案兜底。
- `selection` 语义（每题独立）：`single` 单选（单问题时点选项即提交，多问题时选中后待全局提交）；`multiple` 多选（勾选多项后确认提交）。
- 面板生命周期：`showAskPanel` / `hideAskPanel`；提交成功隐藏；× 关闭 = 取消本轮（复用停止逻辑）；切换会话/清空会话/`stream-done`/`stream-error`/停止时隐藏。
- 与计划面板互斥（方案 B）：`showAskPanel` 先收起计划面板，`hideAskPanel` 后若仍在流式中且有计划数据则恢复。
- 未注入 AskWaiter（非交互场景/测试）时不发射事件，直接返回引导文本。
- 落库保障：反问轮以本轮全部流式正文（问句 + 续答）作为 `result.Content` 落库，与前端同一气泡展示一致。

详见 [ask_user.go](internal/agent/tools/ask_user.go) 与 [context.go](internal/agent/tools/context.go)（`AskWaiter` / `ClaimAsk` / `WaitForAnswer`）。

---

## 5. 审批交互事件（`ai:tool-approval`）

工作目录危险操作（`write_file` 覆盖已存在文件 / `edit_file` 编辑已存在文件 / `run_command` **每次执行**）真正执行前，经 `Context.Approver`（`tools.Approver`）请求用户审批。父包 `agentSession`（[agent.go](internal/agent/agent.go) 的 `RequestApproval`）在抢占审批名额后发射此事件并**阻塞等待**用户决定，工具暂停执行、ReAct 循环挂起；决定经 `ApproveToolCall` 投递后同轮恢复（不落库、不新开一轮）。

负载为 JSON 字符串：

```json
{"tool": "write_file", "summary": "覆盖文件：a.txt", "approval_id": 1, "critical": false}
```

- `tool`：请求审批的工具名（`write_file` / `edit_file` / `run_command` / `transfer_file` / `delete_file` 等，分级见下方「各工具审批分级」）。
- `summary`：操作的中文摘要（如"覆盖文件：xxx"/"执行命令：rm -rf …"），供前端审批面板展示。
- `approval_id`：本次审批的唯一自增编号，前端回调 `ApproveToolCall(sessionID, approvalID, approved)` **必须原样回传**，后端据此防串审（不一致报错）。
- `critical`：是否为不可绕过危险操作（破坏宿主系统的命令 / 高风险 net 类子命令命中 / transfer_file 下载到桌面等外部副作用写入时为 `true`）。`critical=true` 时前端审批面板**不应提供"忽略直接执行"语义**；`review` 模式下后端强制阻塞确认（不可绕过的最后防线），`auto` 模式自动放行但写 `tool_auto_approval` 审计留痕（门控实现见 [agent.go](internal/agent/agent.go) `RequestApproval`）。

**审批模式门控**（由后端依据会话配置 `approval_mode` 决定，事件仅在真正需要阻塞时才发射）：
- `confirm_every`：`critical` 任意 → 都需阻塞确认。
- `review`：`critical=true` → 强制阻塞确认（不可绕过）；`critical=false`（覆盖已存在文件等常规危险操作）→ 自动放行，不发射事件。
- `auto`：一律自动放行，不发射事件；`critical=true` 额外写 `tool_auto_approval` 审计留痕（`recordAutoApproval`）。

**各工具审批分级**（critical 取值以各工具文件头注释与 `requestApproval` 调用为权威）：
- `run_command`：命中高危命令黑名单 `critical=true`，否则 `false`。
- `delete_file`：删除操作一律 `critical=true`。
- `write_file` / `edit_file` / `copy_file` / `move_file`：覆盖已存在目标时请求审批 `critical=false`（纯新增免审批）。
- `transfer_file`：download（写用户桌面 = 工作区之外的外部副作用）一律请求审批 `critical=true`（覆盖时摘要附「（覆盖）」）；upload 对齐 copy_file——纯新增免审批、覆盖已存在目标请求审批 `critical=false`。
- `manage_note` / `manage_notebook` / `manage_tag` / `manage_todo`：写操作接入门控（create 免审批），删除类 action 一律 `critical=true`——`manage_note.delete`（软删进回收站，恢复由用户在回收站页面自行操作）、`manage_notebook.delete`（可选 `with_notes`，默认 false 其下笔记迁入默认笔记本、true 连同笔记移入回收站）、`manage_tag.delete`、`manage_todo.delete` / `clear`（硬删，审批摘要注明「不可恢复」）；其余写操作分级以各工具文件头注释为权威（如 `manage_note.edit` 恒 `critical=true`、批量 `move` / `add_tag` / `remove_tag` 为 `critical=true`、`update` / `pin` 等为 `critical=false`）。

**回调语义**（Wails 方法 `ApproveToolCall(sessionID uint, approvalID uint64, approved bool) error`）：
- `approved=true`：批准，工具返回 nil 继续执行，循环恢复。
- `approved=false`：拒绝，工具以中文错误文本（"用户拒绝了本次操作…"）返回，经 `WrapWithError` 落成 `tool_error` 记录并回填模型继续推理（不中断循环）。
- 无等待中的审批 / `approval_id` 不匹配 → 返回中文错误，前端应提示并刷新（不重复投递）。
- 会话在审批等待期间被停止/释放 → ctx 取消，工具以 `ctx.Err()` 返回，循环随终止。

事件在请求"真正阻塞确认"时发射；`review` 且 `critical=false` 自动放行不发射，`auto` 模式全部自动放行不发射（其中 `critical=true` 写 `tool_auto_approval` 审计痕）。并行危险操作（模型同轮多条）仅一条发射并阻塞，其余直接返回错误（防整轮挂起）。前端在切换会话 / 清空会话 / 停止 / `stream-done` / `stream-error` 时应隐藏审批面板并清理本会话的待回传 `approval_id`。

---

## 6. 规划事件（`ai:plan-generating` / `ai:plan-created` / `ai:plan-updated`）

`create_plan` / `update_plan` 是允许工具内部直接 `ctx.Emit` 事件的例外（与 `ask_user` 并列），用于向前端展示执行计划卡片。这两个工具同时也会产生标准的 `ai:tool-status` 事件（`tool_start` / `tool_result`），规划事件是额外的独立通道。

### 6.0 `ai:plan-generating`（Plan-and-Exec 预规划状态）

Plan 模式下，[agent.go](internal/agent/agent.go) `Run()` 在调用 `generatePlan()`（单独 LLM 调用生成执行计划）前发射此事件，通知前端预规划阶段开始。首次负载为空字符串。

- 前端收到后将打字动画（`createTypingDots`）替换为"正在制定执行计划…"状态文案（`.ai-msg-plan-generating`），持续显示轮换文案。
- **重试机制**：`generatePlan()` 内部在解析/校验失败时自动重试（最多 3 次），重试期间不再额外发射此事件，前端始终保持轮换文案（重试进度仅记录后端日志）。
- `generatePlan()` 完成后由 `ai:plan-created` 事件接替渲染计划面板；所有重试均失败时由 `ai:stream-error` 接替展示错误。
- Agent 模式下不发射此事件。

### 6.1 `ai:plan-created`

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

### 6.2 `ai:plan-updated`

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

### 6.3 前端消费要点

- 负载**不含 `goal` 字段**：前端需将增量合并到已有数据（`Object.assign({}, streamPlanData, payload)`），直接覆盖会丢失标题。
- `ai:tool-status` 用于状态条展示（与其他工具一致），规划事件用于渲染计划面板，两类事件需同时处理。

---

## 7. 结果汇总事件（`ai:agent-result`）

Agent 最终结果汇总时由 [app.go](app.go) `CallAIAgentStream` 发射，参数 `(streamGen, RecallCards, ToolCalls, Plan, ReasoningContent)`，随后紧接正常路径的 `ai:stream-done`。

- `RecallCards`：`services.RecallCard` 数组（`recall_notes` 本地向量召回卡片，前端 `renderRecallCards` 展示，历史回放同；`Content` 在序列化前截断为 `RecallPreviewMaxLen`=200 字符预览，截断卡片携带 `truncated=true`）。
- `ToolCalls`：工具调用链（前端折叠明细）。
- `Plan`：本轮执行计划 JSON（`Result.Plan`，历史回放 `renderPlanCard` 渲染；为 `null` 表示无计划）。
- `ReasoningContent`：思考链。

---

## 8. 摘要状态事件（`ai:summary-status`）

会话摘要压缩状态，由 [app.go](../../app.go) `truncateAIMessages` 路径发射，参数为 map：

```json
{"status": "generating" | "done" | "failed", "session_id": 123}
```

- 触发条件：tail 估算 token 达上下文预算（`ai_context_token_budget`，默认 128K）× 触发比例（`ai_context_summary_trigger_ratio`，默认 0.8）时压缩摘要；事件仅在触发轮发射。
- `generating`：开始生成（同步阻塞，当前轮对话即可用新摘要）。
- `done`：生成成功，正常继续本轮对话。
- `failed`：生成失败，**本轮对话被后端中止**（紧随其后会有 `ai:stream-error` 通知并解锁输入）；用户重新发起对话时会再次触发摘要。用户主动取消时不发 `failed`，由取消语义的 `ai:stream-done` 收尾。
- 前端 `summaryGenerating` 状态控制"正在生成对话摘要…"提示，取消流时重置。

---

## 9. 其他事件（非 Agent 链路）

- `ai:aiop-chunk` / `ai:aiop-done` / `ai:aiop-error`：AI 一站式处理（AIOP）链路的流式推送与终态通知（fire-and-forget，无返回值），参数 `(streamGen, content)` 等。
- 非事件类：工具元信息（`GetAgentTools`）、会话/消息 CRUD 等走 Wails 方法调用，不经事件通道。
