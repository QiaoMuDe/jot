# Agent 工具开发与维护指南

> 本指南面向 `internal/agent` 模块的后续维护者：如何**新增**、**维护**、**注册**和**编写** Agent 工具，以及必须遵守的规范。
> 适用对象：`recall_notes` / `read_url` / `manage_note` / `get_stats` / `ask_user` 等 ReAct 循环中模型可调用的工具（完整清单见 §6）。

---

## 1. 架构概览

```
internal/agent/                    父包（Agent 对话链路）
├── agent.go                       AgentService.Run：装配 ChatModelAgent、消费事件流
├── registry.go                    buildTools：统一注册全部工具（新增工具的必经之地）
├── types.go                       Request / Result / EmitFn 对外契约
├── doc.go                         包级说明文档
└── tools/                         工具子包（每文件一个工具，完整清单见 §6）
    ├── context.go                 共享上下文：Context / Record / Collector / WrapWithError
    ├── manage_note.go             带依赖 + ActionText 的复杂工具参考（create/list/view/...）
    ├── plan.go                    规划工具实现（create_plan / update_plan）
    └── doc.go                     子包说明文档（工具清单，需同步维护）

前端对应展示（工具名直接展示英文，无需维护映射；如需动作文案见 §8）：
frontend/src/js/ai-chat.js                 实时状态条 / 历史回放明细渲染
frontend/src/css/components/ai-chat.css    状态条与折叠摘要样式
```

### 运行机制（ReAct 循环）

模型在循环中自行决策何时调用哪个工具，每次工具调用经历：

```
模型输出 ToolCall → agent.go emit "tool_start" 事件
                  → eino 执行工具的 InvokableRun
                  → 成功: emit "tool_result"；失败: WrapWithError 捕获并 emit "tool_error"
                  → 工具登记的部分失败经 DrainPartials 以 "tool_partial" 事件发射
                  → 工具返回文本回填给模型，模型继续推理或给出最终回答
```

- 工具只负责：**解析参数 → 执行 → 返回纯文本**（及可选的**结构化收集**）。
- 所有事件发射、调用记录、失败回填都由共享上下文与父包统一处理，工具内部**不直接 emit 事件**（**唯一例外**：`ask_user` 反向提问工具直接 `ctx.Emit("ai:ask-user", ...)` 发射问题卡片数据——这是向用户展示交互卡片的专用通道，事件协议见 [EVENTS.md](internal/agent/EVENTS.md)）。
- 前端通过 `ai:tool-status` 事件实时展示工具状态条，事件负载为 `tools.Record` 的 JSON。状态条与历史明细**直接展示英文工具名**（recall_notes 等），前端不维护中文映射。

---

## 2. 新增一个工具（6 步）

### 第 1 步：在 `tools/` 子包新建文件

文件名即工具名（snake_case），如 `your_tool.go`。文件头注释说明工具职责与实现要点（照抄现有文件的风格）。

### 第 2 步：实现工具

按下面"编写规范"实现：结构体 + `Info()` + `InvokableRun()` + `NewXxx()` 构造器。

### 第 3 步：注册

打开 [registry.go](internal/agent/registry.go#L19-L31) 的 `buildTools`，**追加一行**：

```go
tools.WrapWithError("your_tool", tools.NewYourTool(...), p.ctx),
```

### 第 4 步：更新工具清单文档

Go 包文档是工具清单的**唯一权威来源**，各补一行：
- [tools/doc.go](internal/agent/tools/doc.go) 的工具列表与构造器名（**必须**，权威清单）
- [doc.go](internal/agent/doc.go) 的结构说明（如涉及新依赖，同步更新 `Deps` 说明）

注意：本指南（TOOLS.md）不维护具体工具清单，新增工具**无需更新本文件**（见 §6）。

### 第 5 步：标记工具的模式约束（可选）

若工具**仅在 Plan 模式下可用**（如 `create_plan` / `update_plan`），需在 [tools/meta.go](internal/agent/tools/meta.go) 的 `BuiltinTools()` 中将该条目的 `PlanOnly` 设为 `true`：

```go
{Name: "your_plan_tool", Label: "说明", PlanOnly: true},
```

`PlanOnly` 的效果：
- **后端**：Agent 模式（`planMode=false`）下 `buildTools` 自动跳过该工具的注册，模型不可见、不会调用。
- **前端**：设置页工具列表中该工具显示为禁用样式（灰色 + checkbox disabled），点击触发抖动并通知"仅 Plan 模式可用"。

大多数工具不需要设置此字段（零值 = 两种模式都可用）。

### 第 5b 步：标记常驻不可禁用工具（可选）

若工具是**核心能力，不允许用户在前端禁用**（如 `manage_memory` / `ask_user`），在 [tools/meta.go](internal/agent/tools/meta.go) 的 `BuiltinTools()` 中将该条目的 `AlwaysOn` 设为 `true`：

```go
{Name: "your_core_tool", Label: "说明", AlwaysOn: true},
```

`AlwaysOn` 的效果：
- **后端**：读写 `ai_agent_tools_disabled` 的装配入口（[app.go](app.go) 的 `DisabledTools` 过滤处）会将该工具名从禁用集合中**强制剔除**（若配置残留则记 Warn 日志），保证任何配置/历史残留下模型都可见、可调用。
- **前端**：设置页该工具 checkbox 置灰不可勾选（复用仅 Plan 模式的禁用样式），强制勾选、不参与"全选/全不选"，可配置时点击触发抖动提示。

与 `PlanOnly` 的区别：
- `PlanOnly` 限制"**仅 Plan 模式**可用"，Agent 模式不注册、模型不可见。
- `AlwaysOn` 仅保证"**不可被禁用**"，工具本身在两种模式都正常注册、可用。

> 使用原则：`AlwaysOn` 只用于"禁用即导致注入与工具割裂 / 失能"的系统级能力，勿滥用为非系统能力加锁。当前强制启用集合见 [tools/doc.go](internal/agent/tools/doc.go)。

### 第 6 步：按需在工具实现内维护动作文案（可选）

工具状态条与历史明细直接展示英文工具名（recall_notes 等），无需维护中文名映射；若要在开始调用时展示具体动作（如"创建待办"），让工具实现可选接口 `ActionTextProvider`（`ActionText(argumentsInJSON string) string`），父包在 `tool_start` 时自动生成 `action_text` 下发前端，无需修改前端（见 §8）。此步可跳过。

### 第 7 步：验证

```bash
go build ./...
go vet ./internal/agent/...
```

然后重启应用，在 Agent 模式下触发一次调用验证。

---

## 3. 注册位置（核心）

**唯一注册入口：`internal/agent/registry.go` 的 `buildTools` 函数。** 注册形式如下（完整清单以 [registry.go](internal/agent/registry.go#L19-L31) 的 `buildTools` 为准）：

```go
// 在 buildTools 返回的切片中追加（每个工具一行，用 WrapWithError 包装）：
tools.WrapWithError("your_tool", tools.NewYourTool(...), p.ctx),
```

要点：

- **每个工具都用 `tools.WrapWithError` 包装**。它保证工具执行失败时不中断 ReAct 循环：错误文本回填给模型继续推理、记 `tool_error` 记录、发射 `tool_error` 事件供前端展示失败态。
- 注册顺序即工具暴露给模型的顺序，无强依赖，但建议按用途分组排列。
- 工具依赖从 `BuildParams` 取：
  - `p.deps`：`Deps{AI, Vector, Setting, Todo, Notebook, Tag, Note, Stats, Logger, GetEmbedConfig}`（见 [agent.go](internal/agent/agent.go#L44-L57)）
  - `p.req`：本轮 `Request`（如需绑定会话级参数，如 `recall_notes` 的笔记本过滤 `RecallNotebookIDs`）
  - `p.ctx`：本轮共享的 `*tools.Context`
- **新增依赖**：若工具需要新的服务，先在 `Deps` 加字段，再在 [app.go](app.go) 构造 `NewAgentService` 处传入。
- **动作文案（可选）**：由工具实现可选接口 `ActionTextProvider` 提供，父包在 `tool_start` 时自动调用其 `ActionText` 生成 `action_text` 下发前端；未实现则前端回退"执行"（见 §8）。

---

## 4. 编写规范

### 4.1 标准骨架（带依赖 + 结构化收集，参考 [recall_notes.go](internal/agent/tools/recall_notes.go)）

> 如需动作文案，工具可额外实现可选接口 `ActionTextProvider`（见 §8.3），骨架中无对应代码。

```go
package tools

// 文件头注释：工具职责、调用时机、实现要点。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"jot/internal/services"
)

// xxxTool 工具结构体：声明所需依赖。
type xxxTool struct {
	svc *services.XxxService // 依赖按需注入
	ctx *Context             // 需要结构化收集 / 部分失败登记时注入
}

// 编译期断言：确保实现 tool.InvokableTool。
var _ tool.InvokableTool = (*xxxTool)(nil)

// Info 返回工具元信息（模型据此决定是否调用）。
func (x *xxxTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "xxx",
		Desc: "一句话说明：何时调用、能做什么、参数含义。不要写实现细节。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "参数说明",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行工具：解析参数 → 调用服务 → 返回纯文本结果。
func (x *xxxTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("解析 xxx 参数失败: %w", err)
	}
	// 参数校验：必填项为空直接报错（错误会经 WrapWithError 回填模型）
	if args.Query == "" {
		return "", errors.New("xxx 参数缺少 query")
	}

	// 用户取消检查：工具内长耗时操作前可检查 ctx.Err()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 调用服务，返回格式化文本
	return "结果文本", nil
}

// NewXxx 创建工具。构造器签名 = 工具的全部依赖。
func NewXxx(svc *services.XxxService, ctx *Context) tool.InvokableTool {
	return &xxxTool{svc: svc, ctx: ctx}
}
```

### 4.2 最简无参工具模板

当前内置工具均带参数，此模板供未来新增无参工具时参考：

```go
func (c *xxxTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "xxx",
		Desc:        "…",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (c *xxxTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return "结果", nil // 无失败路径，直接返回
}
```

### 4.3 命名与命名空间

| 项目 | 规范 | 示例 |
|---|---|---|
| 工具名 | snake_case，动词 + 对象，全局唯一 | `read_url` / `recall_notes` |
| 结构体 | 工具名 + `Tool` 后缀（小写，包内私有） | `readURLTool` |
| 文件 | 工具名 + `.go`，一文件一工具 | `read_url.go` |
| 构造器 | `New` + 工具名（导出的 CamelCase） | `NewReadURL` |
| 参数名 | 小写 snake_case | `query` / `sources` |
| 依赖字段 | 类型简写字段 | `ai` / `setting` / `vector` |

### 4.4 `Info()` 编写要求

- **`Desc` 是模型选择工具的唯一依据**：写清"何时调用 / 何时不要调用 / 参数含义"，可参考 [read_url.go 的 Desc](internal/agent/tools/read_url.go#L63)。
- 参数 Schema 用 `schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{...})`，支持的字段：

```go
&schema.ParameterInfo{
	Type:     schema.String,             // schema.String / schema.Array / schema.Object / schema.Number / schema.Boolean
	ElemInfo: &schema.ParameterInfo{     // 仅数组：元素类型 + 枚举
		Type: schema.String,
		Enum: []string{"a", "b"},
	},
	SubParams: map[string]*schema.ParameterInfo{}, // 仅对象：子参数
	Desc:      "参数说明",
	Enum:      []string{"a", "b"},       // 仅字符串：限定合法值
	Required:  true,                     // 是否必填
}
```

### 4.5 执行与返回规范

- **返回纯文本**：`InvokableRun` 返回的字符串会回填给模型作为工具结果。结构化、分点、清晰的文本模型才读得懂。
- **错误处理**：
  - 参数解析失败、参数校验失败 → 返回 `error`（`WrapWithError` 会回填"工具执行失败：xxx，请调整策略或直接回答"）。
  - 用户停止 → 返回 `ctx.Err()`（父包不会误报失败，循环随 ctx 终止）。
  - 执行失败但仍有部分可用结果（如多来源搜索部分失败）→ **返回成功文本 + 失败说明后缀**，并通过 `ctx.AddPartial` 登记，而不是整体返回错误。
- **长文本**：返回前用 `TruncateRunes(s, MaxResultLen)` 截断**事件展示用**的摘要（父包 `emitToolResult` 已自动处理，工具一般无需关心）；`MaxResultLen = 500`。

### 4.6 共享上下文 `Context` 的用法

[context.go](internal/agent/tools/context.go) 定义 `Context`，工具通过它做三件父包统一处理的事：

| 能力 | 用法 | 说明 |
|---|---|---|
| 日志 | `ctx.Logger.Debugw(...)` / `Warnw(...)` | fastlog 结构化日志，见 [read_url.go](internal/agent/tools/read_url.go#L153-L158) |
| 部分失败登记 | `ctx.AddPartial(msg)` | 父包在 `tool_result` 后统一以 `tool_partial` 事件发射，前端显示 ⚠️ |
| 结构化收集 | `ctx.Collector.Cards`（召回卡片）；`ctx.Collector.Sources` 字段保留但内置搜索已移除、当前无工具写入（MCP 搜索以文本返回） | 供 `Result.RecallCards`（`Result.SearchSources` 恒空）落库 |

**收集规则**：结构化数据（召回卡片 `services.RecallCard`；搜索来源 `services.SearchSource` 类型保留但当前无工具收集）追加到 `ctx.Collector`，不要塞进返回文本的 JSON。父包在 [agent.go 汇总段](internal/agent/agent.go#L259-L264) 统一序列化落库。

### 4.7 约束与红线

- **禁止 import 父包 `jot/internal/agent`**：会造成循环依赖（父包 import 了 tools）。工具只能依赖 `services` / `einocli` / 标准库 / 第三方库。
- **禁止工具内部直接 emit 事件**（`ctx.Emit` 仅供父包使用）。需要提示前端的行为用 `AddPartial`。**例外**：以下工具为向用户展示交互卡片，允许直接 `ctx.Emit`：
  - `ask_user`：发射 `ai:ask-user` 事件（负载为 `{questions: [{question, options, selection}], question, options, selection}` JSON，`questions` 数组为主格式，旧顶层字段取首条问题值兼容旧前端）。
  - `create_plan`：发射 `ai:plan-created` 事件（负载为 `{goal, steps}` JSON）。
  - `update_plan`：发射 `ai:plan-updated` 事件（负载为 `{step_id, status, result, steps}` JSON）。
  - 其他工具一律不得仿照；各类事件的完整协议见 [EVENTS.md](internal/agent/EVENTS.md)。
- **禁止在 `InvokableRun` 里启动长生命周期 goroutine**（工具是无状态、可并发执行的）。
- **不感知前端协议**：工具不知道 `runtime.EventsEmit` / 事件名，只依赖 `Context` 抽象。
- **依赖必须在构造器参数中显式声明**：不要在工具内部 new 服务或读全局变量（可测试性差）。

---

## 5. 维护工具

### 5.1 修改既有工具

- **改描述 / 参数**：只改 `Info()`，无需动注册与事件循环。
- **改执行逻辑**：只改 `InvokableRun()`。
- **改依赖**：同时改结构体字段、`NewXxx()` 签名、以及 [registry.go](internal/agent/registry.go#L19-L31) 中对应注册行。
- **改结构化收集**：确认收集类型与 `services` 包中数据结构一致（前端按该结构渲染）。
- **改动作文案**：直接改对应工具文件里的 `ActionText` 方法实现（见 §8），无需改前端。

### 5.2 删除工具

1. 删除 `tools/` 下对应文件；
2. 从 `buildTools` 移除注册行；
3. 更新 `tools/doc.go` 与父包 `doc.go` 清单。

### 5.3 新增工具的典型自查清单

- [ ] 文件头注释写了职责？
- [ ] 有 `var _ tool.InvokableTool = (*xxxTool)(nil)` 断言？
- [ ] `Info().Desc` 说清了"何时调用"？
- [ ] 参数校验完备（必填项、非法枚举）？
- [ ] 需要结构化收集 / 部分失败时注入了 `ctx` 并使用 `Collector` / `AddPartial`？
- [ ] registry.go 注册了（且用 `WrapWithError` 包装）？
- [ ] 若为仅 Plan 模式可用的工具，tools/meta.go 的 `BuiltinTools()` 中 `PlanOnly` 设为 `true`？
- [ ] 两个 doc.go 清单更新了（含 get_stats 等只读工具与全部构造器名）？
- [ ] （如需动作文案）工具实现了 `ActionTextProvider` 接口（见 §8）？工具名直接展示英文名，无需映射。
- [ ] `go build ./...`、`go vet ./internal/agent/...` 通过？

---

## 6. 工具清单（权威来源）

本指南不维护具体工具清单。现有工具与构造器以以下代码真相为准：
- [tools/doc.go](internal/agent/tools/doc.go)：工具列表与导出构造器名（权威清单）
- [registry.go](internal/agent/registry.go#L19-L31)：注册顺序与依赖注入

新增/删除工具时仅需同步上述 Go 文档，**无需更新本文件**。

---

## 7. 事件协议

前后端交互事件（`ai:tool-status` / `ai:ask-user` / `ai:plan-created` / `ai:plan-updated` / `ai:agent-result` / `ai:stream-*` 等）的统一协议已迁移至 [EVENTS.md](internal/agent/EVENTS.md)。前端联调、事件消费与面板生命周期请查阅该文档。

---

## 8. 前端工具状态提示（动作文案在工具实现内维护）

Agent 工具的前端状态提示集中在 `frontend/src/js/ai-chat.js`。**工具名直接展示后端下发的英文名**（recall_notes / read_url / ...），前端不维护中文名映射，新增工具无需改动前端展示名。

### 8.1 展示名规则

`getToolLabel(name)`（[ai-chat.js#L4060](frontend/src/js/ai-chat.js#L4060)）直接返回英文工具名（带 `「」` 展示）：

```js
var getToolLabel = function(name) { return name || '工具'; };
```

如需本地化，可在此加映射兜底，但**默认不维护**。

### 8.2 实时状态条文案（统一格式）

| 阶段 | 事件 | 文案格式 | 位置 |
|---|---|---|---|
| 开始调用 | `tool_start` | `调用「{工具名}」工具：{动作}` | `showToolStatusStart` |
| 完成 | `tool_result` | `「{工具名}」：已完成` | `showToolStatusDone` |
| 失败 | `tool_error` | `「{工具名}」：失败：{原因前40字}` | `showToolStatusError` |
| 部分失败 | `tool_partial` | `「{工具名}」：部分来源失败：{说明}` | `showToolStatusPartial` |

- 工具名一律通过 `getToolLabel(name)` 获取并带 `「」`。
- **动作文案**：开始调用一行的"动作"来自后端 `tool_start` 记录的 `action_text`（由工具实现 `ActionTextProvider` 提供），缺失或为空时回退"执行"；前端不再维护任何工具名/动作分支。
- 失败原因 / 部分失败说明来自 `payload.result`，截断 40 字符展示。

### 8.3 在工具实现内维护动作文案

若要在开始调用时展示具体动作（如 manage_note 的"创建笔记"、recall_notes 的"检索本地笔记"），让工具在自己的 .go 文件里实现可选接口 `ActionTextProvider`：

```go
// ActionTextProvider 可选接口：提供开始调用时的中文动作文案。
// 实现后，父包在 tool_start 时自动调用 ActionText 生成 action_text 下发前端。
type ActionTextProvider interface {
	ActionText(argumentsInJSON string) string
}
```

实现要点：

- 工具在自己的 .go 文件实现 `ActionText(argumentsInJSON string) string`：解析 arguments JSON 中的 action / 关键参数，返回中文文案（如 manage_note 的 create → "创建笔记"、recall_notes → "检索本地笔记"）。
- **解析失败返回 ""**：前端回退显示"执行"。
- **action 未命中返回 "执行"**：给出明确的兜底文案。
- 父包在 `tool_start` 时按工具名自动调用 `ActionText`，把结果放进 `Record.ActionText`（json 字段 `action_text`）随 `ai:tool-status` 下发，**无需改动前端**。
- 未实现该接口的工具 → 前端回退显示"执行"。

### 8.4 历史回放明细（折叠组件）

[renderToolCalls](frontend/src/js/ai-chat.js#L4068-L4130) 只渲染 `tool_start` 记录，每条 = 图标 + 加粗工具名 `「X」`（`.ai-tool-status-name`）+ 状态文本：

- 完成：`「recall_notes」：已完成`
- 失败：`「recall_notes」：失败：{原因前40字}`（原因取自 `tool_error` 记录 `result`）
- 部分失败：`「recall_notes」：部分来源失败：{说明前40字}`（取自 `tool_partial` 记录 `result`）

### 8.5 前端维护自查

- [ ] 无需维护中文名映射（工具名直接展示英文）。
- [ ] 无需在 `showToolStatusStart` 维护工具名分支（已删除）。
- [ ] `npm run build` 通过？
