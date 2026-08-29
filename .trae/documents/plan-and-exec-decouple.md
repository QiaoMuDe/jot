# Plan-and-Exec 解耦实现计划

## 摘要

将当前的 Plan-within-ReAct 模式（计划生成与执行混在同一个 ReAct 循环中）解耦为 Plan-and-Exec 模式：

1. **阶段1（Plan Generation）**：单独调用 LLM 生成结构化执行计划（强制，通过 function calling）
2. **阶段2（Plan Execution）**：ReAct 循环基于预生成的计划执行

**影响范围**：仅修改 `internal/agent/agent.go` 一个文件。前端零改动，Agent 模式零影响。

***

## 当前状态分析

### 现有 Plan 模式流程

```
用户请求 → ReAct 循环开始
  → 每轮 GenModelInput 注入 genPlanHint("必须先用 create_plan")
  → 模型可能调用 create_plan → PlanState 创建 → ai:plan-created 事件
  → 模型执行工具 → 调用 update_plan → ai:plan-updated 事件
  → 模型想输出最终回答 → 检测未完成步骤 → 拒绝，强制继续
  → 兜底：模型跳过 create_plan 但执行了工具 → 自动补建单步计划
```

### 问题

* 模型可以跳过 `create_plan`（靠提示词引导 + 兜底补建）

* 计划生成和执行耦合，无法强制"先规划再执行"

### 目标模式

```
用户请求 → generatePlan()（单独 LLM 调用，function calling create_plan）
  → 成功：设置 PlanState + emit ai:plan-created
  → 失败：返回错误，通知用户重试/检查 API 配置
  → ReAct 循环开始（PlanState 已存在，genPlanHint 直接注入进度）
  → 模型按计划执行，调用 update_plan 更新进度
```

***

## 拟定变更

### 变更文件

| 文件                        | 改动类型 | 说明                                                          |
| ------------------------- | ---- | ----------------------------------------------------------- |
| `internal/agent/agent.go` | 修改   | 新增 `generatePlan()` 方法 + 修改 `Run()` 入口 + 修改 `genPlanHint()` |

**不需要改动的文件**：

* `types.go` — `Request.PlanMode` 和 `Result` 结构不变

* `registry.go` — `buildTools()` 不变，`create_plan`/`update_plan` 仍然注册（执行阶段可能调整计划）

* `tools/plan.go` — 工具实现不变

* `tools/context.go` — 数据结构不变

* `app.go` — Wails 绑定层不变

* 前端 — 事件协议不变（`ai:plan-created` / `ai:plan-updated`）

***

### 变更1：新增 `generatePlan()` 方法

**位置**：`internal/agent/agent.go`，在 `Run()` 方法之前或之后新增。

**功能**：Plan 模式下，在 ReAct 循环前单独调用 LLM 生成结构化执行计划。

**实现细节**：

```go
// generatePlan 在 Plan 模式下预生成执行计划：用精简提示词 + create_plan 工具的
// function calling 强制模型输出结构化计划，设置 PlanState 并发射 ai:plan-created 事件。
// 失败时返回错误，由 Run() 中断整个任务并通知用户。
func (s *AgentService) generatePlan(ctx context.Context, chatModel *openai.ChatModel,
    req Request, toolCtx *tools.Context) error {
    // 1. 构造精简的计划生成 system prompt
    // 2. 构造 messages：system + 历史 + 用户消息
    // 3. 调用 chatModel.Generate()，tools 仅包含 create_plan
    // 4. 从返回的 Message.ToolCalls[0].Function.Arguments 解析 plan
    // 5. 构造 tools.Plan 并设置 toolCtx.PlanState
    // 6. 发射 ai:plan-created 事件
    // 7. 任一步骤失败则返回 error
}
```

**计划生成 System Prompt（精简专用）**：

```
你是一个任务规划助手。请根据用户请求制定一个可执行的行动计划。

要求：
1. 将目标拆解为清晰的步骤列表（≤ 10 步）
2. 每步描述简洁明确，标注预计使用的工具名（如有）
3. 步骤应当具体可执行，避免过于笼统的描述

请直接调用 create_plan 工具制定计划，不要输出其他内容。
```

**构造消息**：

```go
msgs := []*schema.Message{
    schema.SystemMessage(planGenPrompt),
}
// 追加历史消息（user/assistant）
for _, h := range req.History {
    switch h.Role {
    case "user":
        msgs = append(msgs, schema.UserMessage(h.Content))
    case "assistant":
        msgs = append(msgs, schema.AssistantMessage(h.Content, nil))
    }
}
// 追加当前用户消息
if req.UserText != "" {
    msgs = append(msgs, schema.UserMessage(req.UserText))
}
```

**工具注册（仅 create\_plan）**：

使用 `chatModel.WithTools()` 创建一个**仅绑定** **`create_plan`** **工具的新模型实例**（不修改原缓存的 chatModel，避免影响 ReAct 循环）：

```go
// 1. 获取 create_plan 工具的 ToolInfo
cpTool := tools.NewCreatePlan(toolCtx)
cpToolInfo, err := cpTool.Info(ctx)
if err != nil {
    return fmt.Errorf("获取 create_plan 工具信息失败: %w", err)
}

// 2. 用 WithTools 创建仅绑定 create_plan 的新模型（不修改原 chatModel）
planModel, err := chatModel.WithTools([]*schema.ToolInfo{cpToolInfo})
if err != nil {
    return fmt.Errorf("绑定计划工具失败: %w", err)
}

// 3. 调用 Generate（非流式），模型必须调用 create_plan
msg, err := planModel.Generate(ctx, msgs)
```

> 关键 API（eino v0.9.13 + eino-ext/openai v0.1.13）：
>
> * `ChatModel.WithTools(tools []*schema.ToolInfo) (ToolCallingChatModel, error)` — 创建绑定工具的新实例
>
> * `ChatModel.BindForcedTools(tools []*schema.ToolInfo) error` — 强制绑定（修改原实例，**不使用**）
>
> * `ToolCallingChatModel` 继承 `BaseChatModel`，拥有 `Generate(ctx, msgs, ...opts)` 方法
>
> * 返回的 `*schema.Message.ToolCalls` 包含模型的工具调用决策

**解析返回值**：

```go
if len(msg.ToolCalls) == 0 {
    return fmt.Errorf("模型未调用 create_plan 工具，请检查模型是否支持 function calling")
}
tc := msg.ToolCalls[0]
if tc.Function.Name != "create_plan" {
    return fmt.Errorf("模型调用了意外的工具 %q，期望 create_plan", tc.Function.Name)
}

// 复用解析逻辑（Step 1 抽取的 ParseCreatePlanArgs）
plan, err := tools.ParseCreatePlanArgs(tc.Function.Arguments)
if err != nil {
    return fmt.Errorf("解析计划参数失败: %w", err)
}
```

**设置 PlanState + 发射事件**：

```go
plan := &tools.Plan{Goal: goal, Steps: steps, Current: 0}
toolCtx.PlanState = plan

// 发射 ai:plan-created 事件（与 create_plan 工具的 emitPlanCreated 逻辑一致）
payload := map[string]any{"goal": plan.Goal, "steps": plan.Steps}
if b, err := json.Marshal(payload); err == nil {
    toolCtx.Emit("ai:plan-created", string(b))
}
```

***

### 变更2：修改 `Run()` 方法 — 插入预规划阶段

**位置**：`internal/agent/agent.go` 第 354 行附近（工具注册完成后、Agent 创建前）。

**在** **`buildTools()`** **之后、`adk.NewChatModelAgent()`** **之前插入**：

```go
// Plan-and-Exec：Plan 模式下先单独调用 LLM 生成计划，再进入 ReAct 执行
if req.PlanMode {
    if err := s.generatePlan(runCtx, chatModel, req, toolCtx); err != nil {
        // 预规划失败（LLM 调用失败/解析失败），中断整个任务
        return result, fmt.Errorf("计划生成失败: %w", err)
    }
}
```

**失败处理**：用户要求失败时停止任务并通知用户。当前 `Run()` 返回 error 后，`app.go` 会通过 `stream-error` 事件通知前端显示错误提示，用户看到的是"计划生成失败：xxx"，可以检查 API 配置或重试。

***

### 变更3：修改 `genPlanHint()` — 适配预生成模式

**位置**：`internal/agent/agent.go` 第 993-1045 行。

**变更逻辑**：

```go
func genPlanHint(plan *tools.Plan, skippedPlanUpdate bool) string {
    if plan == nil {
        // ⚠️ 新增判断：如果 PlanMode 为 true 但 plan 为 nil，
        // 说明预规划失败后不应到达这里（Run() 会提前返回 error）。
        // 此处保持原有逻辑不变作为防御性兜底。
        return "\n\n【执行计划要求】收到用户请求后，必须先用 create_plan 工具制定执行计划..."
    }

    // ✅ plan 已存在（预生成成功）：直接注入进度信息
    var b strings.Builder
    fmt.Fprintf(&b, "\n\n【当前执行计划】\n目标：%s\n进度：%d/%d\n",
        plan.Goal, plan.Current+1, len(plan.Steps))

    // ... 已完成步骤、当前待执行步骤（保持不变）...

    b.WriteString("请按照计划继续执行，或调用 update_plan 调整计划。")

    // 催促提醒（保持不变）
    if skippedPlanUpdate {
        b.WriteString("\n【强制要求】上一轮你执行了工具但未调用 update_plan...")
    }

    // 未完成提醒（保持不变）
    if pendingCount := countPendingSteps(plan); pendingCount > 0 {
        fmt.Fprintf(&b, "\n【强制要求】当前计划还有 %d 个步骤未完成...", pendingCount)
    }

    return b.string()
}
```

**核心区别**：预生成模式下，`plan` 在进入 ReAct 循环时就已经存在，所以 `genPlanHint` 永远走 `plan != nil` 分支，不再输出"必须先用 create\_plan"的引导文本。模型直接看到进度信息，按计划执行即可。

***

### 变更4：`genPlanHint` 增加 `planMode` 参数（可选）

为了让 `genPlanHint` 在预生成模式下输出更精确的提示（比如不需要催促模型调用 `create_plan`），可以考虑传入一个标记。但分析后认为**不需要**，原因：

* 预生成成功后 `plan != nil`，自然不走"引导调用 create\_plan"的分支

* 催促提醒和未完成提醒在两种模式下行为一致

* `genPlanHint` 的签名不需要变更

***

## 关键设计决策

### 1. 为什么保留 `create_plan` / `update_plan` 工具在 ReAct 循环中？

* **`update_plan`**：执行阶段必须保留，模型需要在每步完成后更新进度

* **`create_plan`**：保留作为 fallback。如果模型在执行过程中发现预生成的计划严重不合理，可以通过 `create_plan` 重新制定（虽然当前实现中重新创建会覆盖 PlanState，但这是可接受的行为）

### 2. 为什么使用 `WithTools` 而不是 `BindForcedTools`？

* `BindForcedTools()` 会**修改原 chatModel 实例**，影响后续 ReAct 循环的工具列表

* `WithTools()` 创建一个**新的模型实例**，仅绑定 `create_plan`，不污染原缓存

* 预规划完成后，新实例自动被 GC 回收，原 chatModel 保持纯净

### 3. 为什么不直接调用 `createPlanTool.InvokableRun`？

* 复用参数解析逻辑是好的，但 `InvokableRun` 会修改 PlanState 和发射事件，与预规划流程耦合

* 更好的做法是：从 ToolCalls 的 Arguments 中提取 JSON，用相同的解析逻辑构造 Plan

* 抽取 `ParseCreatePlanArgs(argsJSON string) (*tools.Plan, error)` 函数供两处复用

### 4. 预生成的 Plan 结构是否需要变更？

* **不需要**。现有 `tools.Plan` 和 `tools.PlanStep` 结构完全适用

* `PlanStep.Status` 初始值为 `"pending"`，与 ReAct 循环中的行为一致

### 5. 前端是否需要改动？

* **不需要**。`ai:plan-created` 事件的数据格式不变（`{goal, steps[]}`）

* 前端通过 `ai:plan-updated` 事件跟踪步骤进度，逻辑不变

* 计划面板的显示/隐藏由 `stream-done` 事件控制，逻辑不变

***

## 实现步骤

### Step 1：抽取计划参数解析函数

从 `createPlanTool.InvokableRun` 中抽取参数解析+校验+Plan 构造的逻辑为独立函数，供预规划和工具执行两处复用。

**文件**：`internal/agent/tools/plan.go`
**改动**：新增 `ParseCreatePlanArgs(argumentsInJSON string) (*Plan, error)` 导出函数，从 `InvokableRun` 中提取。

### Step 2：实现 `generatePlan()` 方法

**文件**：`internal/agent/agent.go`
**改动**：

* 新增 `generatePlan()` 方法

* 在 system prompt 中描述 create\_plan 工具格式（精简提示词）

* 调用 `chatModel.Generate()` 获取响应

* 解析响应中的 ToolCalls 或 Content JSON

* 调用 Step 1 的解析函数构造 Plan

* 设置 PlanState + 发射 ai:plan-created 事件

* 失败返回 error

### Step 3：修改 `Run()` 插入预规划阶段

**文件**：`internal/agent/agent.go`
**改动**：在工具注册后、Agent 创建前，PlanMode=true 时调用 `generatePlan()`。失败直接 return error。

### Step 4：确认 `genPlanHint()` 行为

预生成后 `plan != nil`，`genPlanHint` 自动走正确的分支。无需修改代码，但需要验证提示词内容在预生成场景下是否合理。

***

## 验证方式

1. **正常功能测试**：开启 Plan 模式，发送请求，确认：

   * 前端先显示计划卡片（`ai:plan-created` 事件）

   * 然后模型按计划执行工具

   * 每步完成后计划卡片更新（`ai:plan-updated` 事件）

   * 所有步骤完成后输出最终回答

2. **失败测试**：模拟 LLM 调用失败（如 API Key 错误），确认：

   * `Run()` 返回错误

   * 前端显示错误提示（通过 `stream-error` 事件）

3. **Agent 模式不受影响**：关闭 Plan 模式，确认行为与改动前完全一致

4. **边界场景**：

   * 模型输出不包含 ToolCalls 的响应 → 解析失败 → 返回错误

   * 模型输出的 steps 为空 → 校验失败 → 返回错误

   * 超长历史消息导致 token 溢出 → Generate 调用失败 → 返回错误

