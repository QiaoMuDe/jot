# 子 Agent 开发与维护指南

> 本指南面向 `internal/agent` 模块的后续维护者：如何**新增**、**维护**和**注册**子 Agent（委托工具），以及必须遵守的规范。
> 适用对象：`os_agent` 等以「委托工具」形式暴露给父层、内层运行独立 ReAct 循环（ChatModelAgent）的子 Agent。
> 配套文档：[TOOLS.md](internal/agent/TOOLS.md)（工具开发）、[EVENTS.md](internal/agent/EVENTS.md)（事件协议）。

---

## 1. 架构概览

```
internal/agent/
├── subagent.go         子 Agent 通用机制（一般无需改动）
│   ├── subAgentConfig           差异配置结构（一个域子 Agent = 一份配置 + 一次工厂调用）
│   ├── toolConstructors         全工具构造器注册表（白名单按工具名取构造器）
│   ├── delegatedAgentTool       委托工具基类（agent/innerTools/ctx/cfg）
│   ├── newDelegatedAgentTool    内层 ChatModelAgent 构造工厂
│   └── InvokableRun             内层运行 + 事件转发循环
├── subagent_os.go      os_agent 实例（提示词/白名单/配置/构造器，新增子 Agent 样板）
└── registry.go         buildTools：注册委托工具（WrapWithError 包装 + disabled/planMode 过滤）

前端对应渲染：
frontend/src/js/ai-chat.js               osAgentGroupOpen/Close/Active 分组栈 + isAgentGroup/substep 渲染
frontend/src/css/components/ai-chat.css  .is-os-agent / .is-substep 样式
```

### 运行机制（内层 ReAct 循环）

父层只注册**一个委托工具**（如 `os_agent`），模型每次调用它即触发一次内层 ChatModelAgent 的完整 ReAct 循环：

```
模型调用 os_agent(request=任务描述)
  → 父层 emit os_agent "tool_start"
  → 内层 ChatModelAgent 运行（复用同一会话 ChatModel，独立 MaxIterations）
      → 内层工具调用：emitToolStart/emitToolResult 写入父层同一 toolRecords 并发射 ai:tool-status
      → 内层流式正文/思考链：不转发（noopEmit）
  → 父层 emit os_agent "tool_result"（结果为内层最后一条非工具正文）
```

- 事件顺序 = `os_agent tool_start → 内层记录（start/result/error/partial 按序）→ os_agent tool_result`，构成前端分组边界。
- 审批（`ai:tool-approval`）与反问（`ai:ask-user`）沿用**同一会话** `Approver` / `AskWaiter`：内层危险操作照常弹审批面板，前端零改动。
- 事件协议详见 [EVENTS.md](internal/agent/EVENTS.md) §3.1；内层工具实现规范见 [TOOLS.md](internal/agent/TOOLS.md)。

---

## 2. 新增一个子 Agent（8 步）

### 第 1 步：新建 `subagent_<域>.go` 文件

**一个子 Agent 一个文件**（如 `subagent_os.go` / 未来 `subagent_note.go`），只放该域的差异部分，不碰 `subagent.go` 通用机制。文件头注释说明职责与装配关系（照抄 `subagent_os.go` 的风格）。

### 第 2 步：定义系统提示词常量

内层 Agent 的角色与行为约束，按 os_agent 样板分四段：

- **角色**：一句话说明是哪个域的执行子 Agent、负责什么。
- **边界**：只允许做什么（如 workspace 内文件/命令）、什么诉求**不要自行处理**要回告主 Agent。
- **审批说明**：危险操作会请求用户确认，被拒绝时改用其他方式或说明原因。
- **收尾要求**：任务完成时给出结构化摘要（做了什么 / 关键结果 / 遗留问题）。

### 第 3 步：定义工具白名单

```go
var osSubAgentToolNames = []string{
    "read_file", "write_file", "edit_file", "ls_dir", "glob", "grep_file",
    "copy_file", "move_file", "delete_file", "mkdir_dir", "run_command",
}
```

- 白名单按工具名从 `toolConstructors` 取构造器（[subagent.go](internal/agent/subagent.go)），**顺序即注册顺序**。
- **白名单约束（硬性）**：只放本域工具，禁止跨域装配（MCP / 网络 / 笔记 / 记忆等工具不得进内层白名单）。
- 若白名单里的工具尚未登记进 `toolConstructors`，需在 [subagent.go](internal/agent/subagent.go) 的注册表补一行（见第 6 步）。

### 第 4 步：定义 `subAgentConfig` 配置

```go
var osAgentConfig = subAgentConfig{
    name:          "os_agent",
    description:   "…内层 Agent 描述…",
    instruction:   osSubAgentInstruction,
    toolNames:     osSubAgentToolNames,
    maxIterations: osSubAgentMaxIterations,
    actionPrefix:  "执行操作系统任务：",
    infoDesc:      "…Info().Desc：何时调用 / 做什么…",
    requestDesc:   "…request 参数含义…",
}
```

字段语义见 [subagent.go](internal/agent/subagent.go) 的 `subAgentConfig` 注释；要点：

- `name` 同时是**委托工具名**与**内层 ChatModelAgent 名**（snake_case，全局唯一）。
- `infoDesc` 是模型选择委托工具的唯一依据：写清"何时调用 / 何时不要调用 / request 参数含义"，不要写实现细节。
- `maxIterations` 独立于父层，内层循环不消耗父层迭代次数（防死循环，os_agent 取 20）。

### 第 5 步：定义构造器

```go
// buildOSSubAgent 构造 os_agent 委托工具（registry.go 装配入口；chatModel 为 nil 或构造失败返回 nil）。
func buildOSSubAgent(runCtx context.Context, chatModel *openai.ChatModel, innerCtx *tools.Context, disabled map[string]bool) *delegatedAgentTool {
    return newDelegatedAgentTool(runCtx, chatModel, innerCtx, disabled, osAgentConfig)
}
```

构造器签名统一为 `(runCtx, chatModel, innerCtx, disabled) -> *delegatedAgentTool`；`disabled` 参数当前不改变内层白名单（旧禁用名静默忽略，保留供将来扩展）。

### 第 6 步：登记白名单工具构造器（如缺）

`toolConstructors` 是全工具构造器注册表的唯一入口。白名单引用未登记的工具名时，`newDelegatedAgentTool` 会跳过该工具并记 Warn（测试 `TestBuildOSSubAgentInnerTools` 会兜底发现数量不匹配）。

### 第 7 步：注册（registry.go `buildTools`）

参照 os_agent 的注册写法（[registry.go](internal/agent/registry.go#L207-L213)）：

```go
if oa := buildOSSubAgent(p.runCtx, p.chatModel, p.ctx, disabled); oa != nil {
    all = append(all, namedTool{"os_agent", tools.WrapWithError("os_agent", oa, p.ctx)})
}
```

- 构造返回 `nil`（chatModel 为 nil / 构造失败，已内部记 Warn）时**跳过注册**，不破坏其余工具装配。
- 必须用 `tools.WrapWithError` 包装（失败发射 `tool_error` 事件、记录并回填模型继续推理，含 panic 防护）。
- 委托工具在 `meta.go` 的 `BuiltinTools()` 中登记展示文案（名称须与注册名一致），受 `ai_agent_tools_disabled` 过滤、按需标记 `PlanOnly`（os_agent 非 PlanOnly）。

### 第 8 步：验证

```bash
gofmt -l internal/agent/
go build ./...
go vet ./internal/agent/...
go test ./internal/agent/...   # 含 subagent_test.go 的事件转发/装配用例
```

前端验证：`npm run build` 通过；手动触发一次该域多步任务，检查实时子步骤动画、审批弹窗、历史回放分组一致性。

---

## 3. 通用机制（subagent.go）要点

### 3.1 `subAgentConfig` 字段

| 字段 | 语义 |
|---|---|
| `name` | 委托工具名，也是内层 ChatModelAgent 名（snake_case，全局唯一） |
| `description` | 内层 ChatModelAgent Description |
| `instruction` | 内层系统提示词（角色 + 边界 + 审批说明 + 结果摘要要求） |
| `toolNames` | 内层白名单（按名从 `toolConstructors` 取构造器，顺序即注册顺序） |
| `maxIterations` | 内层 ReAct 循环最大迭代次数（防死循环，独立于父层） |
| `actionPrefix` | `ActionText` 动作文案前缀（`tool_start` 展示） |
| `infoDesc` | `Info().Desc`（何时调用 / 做什么） |
| `requestDesc` | `Info()` request 参数描述 |

### 3.2 `newDelegatedAgentTool` 工厂行为

- `chatModel == nil`（未配置 AI）→ 记 Warn 返回 `nil`（buildTools 过滤循环跳过该工具）。
- 白名单逐名取构造器 + `tools.WrapWithError` 包装（失败回填模型不中断内层循环）。
- `adk.NewChatModelAgent` 构造失败 → 记 Warn 返回 `nil`。
- 返回的 `delegatedAgentTool` 带 `innerTools` 字段，供测试断言内层白名单（`InnerTools()`）。

### 3.3 `InvokableRun` 事件转发细节

- **转发**：内层每步工具调用的 start/result/error/partial 实时写入父层同一 `toolRecords` 并发射 `ai:tool-status`（与父层工具完全一致）。
- **不转发**：内层流式正文与思考链（`noopEmit`），避免污染主消息气泡；只把**最后一条非工具正文**作为工具结果返回（全程无正文时兜底「（子 Agent 未返回文本结果）」）。
- **悬空收口**：内层执行中断时，对已发射 start 未收口的工具调用（`startedByCallID`）补发 `tool_result`（原因「（子 Agent 执行中止）」），避免子步骤在前端状态条/回放中悬空为「执行中」；`call_id` 为空时无精确配对依据，保持现状不臆断补发。
- **同轮多次调用**：父模型同轮多次调用委托工具（不同 `call_id`）时，各自保持「委托工具 start → 内层记录 → 委托工具 result」完整分组，按 `call_id` 配对不跨轮错配（对应前端 `osAgentGroupClose` 按 call_id 配对的输入契约）。
- **用户取消**：`ctx.Err()` 优先返回；内层失败包装为「{name} 子 Agent 执行失败: …」中文错误。

---

## 4. 编写规范与约束（红线）

- **通用机制免改**：新增子 Agent 只动「每 Agent 一个文件 + registry.go 注册 + meta.go 文案」，不修改 `subagent.go` 通用机制（如需新能力先在讨论后扩展基类）。
- **白名单只装本域工具**：禁止跨域装配（MCP / 网络 / 笔记 / 记忆等不进内层白名单）。
- **提示词必须写清边界**：明确只允许操作的范围（如 workspace），范围外诉求回告主 Agent，不自行处理。
- **审批/反问复用同一会话通道**：内层使用与父层同一 `Context.Approver` / `AskWaiter`，前端审批/反问面板零改动；不要在子 Agent 内另起审批通道。
- **内层流式正文/思考链不转发**：只回传最后一条非工具正文，保证主消息气泡不被内层过程文本污染。
- **独立迭代上限**：每个子 Agent 必须设置独立的 `maxIterations`（防死循环），不消耗父层迭代次数。
- **日志占位符**：`Warnw` 等结构化日志的 msg **原样输出不解析占位符**（fastlog 语义），变量一律放 `fastlog.String` / `fastlog.Error` 字段，msg 用纯文本。
- **命名规范**：文件 `subagent_<域>.go`、工具名 `name` snake_case 全局唯一、构造器 `build<Xxx>SubAgent`。

---

## 5. 维护子 Agent

### 5.1 修改既有子 Agent

- **改提示词**：改 `subagent_<域>.go` 的 instruction 常量（如 `osSubAgentInstruction`）。
- **改白名单**：改 toolNames 变量；同时确认 `toolConstructors` 已登记全部新名。
- **改配置**：改对应 `subAgentConfig` 字段（`infoDesc` / `requestDesc` / `actionPrefix` 等）。
- **改内层行为**：若需改通用机制（事件转发 / 悬空收口 / 工厂），动 [subagent.go](internal/agent/subagent.go) 并回归 `subagent_test.go`。

### 5.2 删除子 Agent

1. 删除 `subagent_<域>.go` 文件；
2. 从 `buildTools` 移除注册块；
3. 从 `meta.go` 的 `BuiltinTools()` 移除展示条目；
4. 同步 [tools/doc.go](internal/agent/tools/doc.go) 与父包 [doc.go](internal/agent/doc.go) 的结构说明。

### 5.3 新增子 Agent 的自查清单

- [ ] 新建了 `subagent_<域>.go` 且只含差异部分（提示词/白名单/配置/构造器）？
- [ ] 提示词含角色 / 边界（范围外回告主 Agent）/ 审批说明 / 收尾摘要四段？
- [ ] 白名单只含本域工具且全部已在 `toolConstructors` 登记？
- [ ] `subAgentConfig` 各字段齐备，`infoDesc` 说清"何时调用"？
- [ ] 构造器签名统一 `(runCtx, chatModel, innerCtx, disabled) -> *delegatedAgentTool`？
- [ ] `buildTools` 注册了且用 `WrapWithError` 包装、nil 跳过不破坏其余装配？
- [ ] `meta.go` 的 `BuiltinTools()` 登记了展示文案（含 PlanOnly/AlwaysOn 标记核对）？
- [ ] 未修改 `subagent.go` 通用机制（或已同步测试）？
- [ ] 子 Agent 文档（本文件 §2）与相关引用同步？
- [ ] `gofmt`、`go build ./...`、`go vet`、`go test ./internal/agent/...`、`npm run build` 通过？

---

## 6. 测试与前端渲染

### 6.1 单元测试（subagent_test.go）

覆盖基线（新增子 Agent 可参考扩展）：

- 内层白名单装配（数量与名称集合 = 白名单变量）；
- `chatModel` 为 nil / `disabled` 含该委托工具时的装配跳过（其余工具不受影响）；
- 内层 Context 与传入 ctx 为**同一指针**（审批共享）；
- `InvokableRun` 参数错误分支（request 为空 / 非法 JSON 返回中文错误）；
- **事件转发顺序**：`os_agent start → 内层 start/result → os_agent result`，toolRecords 与 `ai:tool-status` 事件均按序；
- **同轮多次调用**：按 `call_id` 分组配对不跨轮错配。

### 6.2 前端分组渲染（ai-chat.js / ai-chat.css）

- 模块级 `osAgentGroupOpen/Close/Active`：按 `call_id` 配对关组，无 `call_id` 按顺序关栈顶，无匹配不臆断关闭。
- 记录打 `isAgentGroup`（主行）/ `substep`（内层）标记；`buildToolStatusRows` 检测到分组时降级为按原始顺序逐条渲染（保证组内顺序不被打散），普通流维持原聚合行为。
- 样式：`.is-os-agent`（running 时左侧脉动点动画，`prefers-reduced-motion` 禁用）、`.is-substep`（缩进 + 左侧引导线 + `color-mix` 弱化背景 + `--text-muted`）。
- 修改渲染逻辑必须同步「实时 + 历史回放」双路径（见 AGENTS.md 渲染单一事实源约定）。
