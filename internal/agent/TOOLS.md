# Agent 工具开发与维护指南

> 本指南面向 `internal/agent` 模块的后续维护者：如何**新增**、**维护**、**注册**和**编写** Agent 工具，以及必须遵守的规范。
> 适用对象：`web_search` / `recall_notes` / `refine_search_query` / `get_current_time` 等 ReAct 循环中模型可调用的工具。

---

## 1. 架构概览

```
internal/agent/                    父包（Agent 对话链路）
├── agent.go                       AgentService.Run：装配 ChatModelAgent、消费事件流
├── registry.go                    buildTools：统一注册全部工具（新增工具的必经之地）
├── types.go                       Request / Result / EmitFn 对外契约
├── doc.go                         包级说明文档
└── tools/                         工具子包（每文件一个工具）
    ├── context.go                 共享上下文：Context / Record / Collector / WrapWithError
    ├── web_search.go              多源联网搜索工具（模型自选来源 + URL 去重）
    ├── recall_notes.go            本地笔记向量召回工具
    ├── refine_query.go            搜索词精炼工具
    ├── current_time.go            当前时间工具（最简无参示例）
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
- 所有事件发射、调用记录、失败回填都由共享上下文与父包统一处理，工具内部**不直接 emit 事件**。
- 前端通过 `ai:tool-status` 事件实时展示工具状态条，事件负载为 `tools.Record` 的 JSON。状态条与历史明细**直接展示英文工具名**（web_search 等），前端不维护中文映射。

---

## 2. 新增一个工具（6 步）

### 第 1 步：在 `tools/` 子包新建文件

文件名即工具名（snake_case），如 `your_tool.go`。文件头注释说明工具职责与实现要点（照抄现有文件的风格）。

### 第 2 步：实现工具

按下面"编写规范"实现：结构体 + `Info()` + `InvokableRun()` + `NewXxx()` 构造器。

### 第 3 步：注册

打开 [registry.go](internal/agent/registry.go#L19-L25) 的 `buildTools`，**追加一行**：

```go
tools.WrapWithError("your_tool", tools.NewYourTool(...), p.ctx),
```

### 第 4 步：更新工具清单文档

两处包说明文档各补一行：
- [tools/doc.go](internal/agent/tools/doc.go) 的工具列表与构造器名
- [doc.go](internal/agent/doc.go) 的结构说明（如涉及新依赖，同步更新 `Deps` 说明）

### 第 5 步：按需更新前端动作文案（可选）

工具状态条与历史明细**直接展示英文工具名**（web_search 等），无需维护中文名映射。仅当需要在开始调用时展示具体动作（如 web_search 的"搜索 {query}"、recall_notes 的"检索本地笔记"）时，在 [showToolStatusStart](frontend/src/js/ai-chat.js#L2530-L2547) 的 switch 中补一个 `name` 分支（见 §8）。此步可跳过。

### 第 6 步：验证

```bash
go build ./...
go vet ./internal/agent/...
```

然后重启应用，在 Agent 模式下触发一次调用验证。

---

## 3. 注册位置（核心）

**唯一注册入口：`internal/agent/registry.go` 的 `buildTools` 函数。**

```go
func buildTools(p BuildParams) []tool.BaseTool {
	return []tool.BaseTool{
		tools.WrapWithError("refine_search_query", tools.NewRefineSearchQuery(p.deps.AI), p.ctx),
		tools.WrapWithError("web_search", tools.NewWebSearch(p.deps.AI, p.deps.Setting, p.ctx), p.ctx),
		tools.WrapWithError("recall_notes", tools.NewRecallNotes(p.deps.Vector, p.deps.Setting,
			p.deps.GetEmbedConfig, p.req.RecallNotebookIDs, p.ctx), p.ctx),
		tools.WrapWithError("get_current_time", tools.NewGetCurrentTime(), p.ctx),
	}
}
```

要点：

- **每个工具都用 `tools.WrapWithError` 包装**。它保证工具执行失败时不中断 ReAct 循环：错误文本回填给模型继续推理、记 `tool_error` 记录、发射 `tool_error` 事件供前端展示失败态。
- 注册顺序即工具暴露给模型的顺序，无强依赖，但建议按用途分组排列。
- 工具依赖从 `BuildParams` 取：
  - `p.deps`：`Deps{A I, Vector, Setting, Logger, GetEmbedConfig}`（见 [agent.go](internal/agent/agent.go#L44-L51)）
  - `p.req`：本轮 `Request`（如需绑定会话级参数，如 `recall_notes` 的笔记本过滤 `RecallNotebookIDs`）
  - `p.ctx`：本轮共享的 `*tools.Context`
- **新增依赖**：若工具需要新的服务，先在 `Deps` 加字段，再在 [app.go](app.go) 构造 `NewAgentService` 处传入。

---

## 4. 编写规范

### 4.1 标准骨架（带依赖 + 结构化收集，参考 [recall_notes.go](internal/agent/tools/recall_notes.go)）

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

### 4.2 最简无参工具模板（参考 [current_time.go](internal/agent/tools/current_time.go)）

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
| 工具名 | snake_case，动词 + 对象，全局唯一 | `get_current_time` / `web_search` |
| 结构体 | 工具名 + `Tool` 后缀（小写，包内私有） | `currentTimeTool` |
| 文件 | 工具名 + `.go`，一文件一工具 | `current_time.go` |
| 构造器 | `New` + 工具名（导出的 CamelCase） | `NewGetCurrentTime` |
| 参数名 | 小写 snake_case | `query` / `sources` |
| 依赖字段 | 类型简写字段 | `ai` / `setting` / `vector` |

### 4.4 `Info()` 编写要求

- **`Desc` 是模型选择工具的唯一依据**：写清"何时调用 / 何时不要调用 / 参数含义"，可参考 [web_search.go 的 Desc](internal/agent/tools/web_search.go#L52)。
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
| 日志 | `ctx.Logger.Debugw(...)` / `Warnw(...)` | fastlog 结构化日志，见 [web_search.go](internal/agent/tools/web_search.go#L99-L104) |
| 部分失败登记 | `ctx.AddPartial(msg)` | 父包在 `tool_result` 后统一以 `tool_partial` 事件发射，前端显示 ⚠️ |
| 结构化收集 | `ctx.Collector.Sources` / `ctx.Collector.Cards` | 供 `Result.SearchSources` / `Result.RecallCards` 落库，格式与问答模式一致 |

**收集规则**：结构化数据（搜索来源 `services.SearchSource` / 召回卡片 `services.RecallCard`）追加到 `ctx.Collector`，不要塞进返回文本的 JSON。父包在 [agent.go 汇总段](internal/agent/agent.go#L242-L251) 统一序列化落库。

### 4.7 约束与红线

- **禁止 import 父包 `jot/internal/agent`**：会造成循环依赖（父包 import 了 tools）。工具只能依赖 `services` / `einocli` / 标准库 / 第三方库。
- **禁止工具内部直接 emit 事件**（`ctx.Emit` 仅供父包使用）。需要提示前端的行为用 `AddPartial`。
- **禁止在 `InvokableRun` 里启动长生命周期 goroutine**（工具是无状态、可并发执行的）。
- **不感知前端协议**：工具不知道 `runtime.EventsEmit` / 事件名，只依赖 `Context` 抽象。
- **依赖必须在构造器参数中显式声明**：不要在工具内部 new 服务或读全局变量（可测试性差）。

---

## 5. 维护工具

### 5.1 修改既有工具

- **改描述 / 参数**：只改 `Info()`，无需动注册与事件循环。
- **改执行逻辑**：只改 `InvokableRun()`。
- **改依赖**：同时改结构体字段、`NewXxx()` 签名、以及 [registry.go](internal/agent/registry.go#L19-L25) 中对应注册行。
- **改结构化收集**：确认收集类型与 `services` 包中数据结构一致（前端按该结构渲染）。
- **改动作文案**：如需调整开始调用时的动作提示，改前端 `showToolStatusStart` 对应分支（见 §8）；工具名直接展示英文名，无需映射。

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
- [ ] 两个 doc.go 清单更新了？
- [ ] （如需动作文案）`showToolStatusStart` 分支符合 §8 格式？工具名直接展示英文名，无需映射。
- [ ] `go build ./...`、`go vet ./internal/agent/...` 通过？

---

## 6. 现有工具清单

| 工具名 | 文件 | 依赖注入 | 功能 | 结构化收集 |
|---|---|---|---|---|
| `refine_search_query` | [refine_query.go](internal/agent/tools/refine_query.go) | `ai` | 口语化搜索词精炼 | 无 |
| `web_search` | [web_search.go](internal/agent/tools/web_search.go) | `ai`、`setting`、`ctx` | 多源联网搜索（模型自选来源、按 URL 去重分组） | `Collector.Sources` |
| `recall_notes` | [recall_notes.go](internal/agent/tools/recall_notes.go) | `vector`、`setting`、`getEmbedConfig`、`notebookIDs`、`ctx` | 本地笔记向量 + 关键词混合召回 | `Collector.Cards` |
| `get_current_time` | [current_time.go](internal/agent/tools/current_time.go) | 无 | 返回当前日期 / 时间 / 星期 / 年份 | 无 |

---

## 7. 事件协议速查（前端联调用）

`ai:tool-status` 事件负载为 `tools.Record` 的 JSON：

| Action | 含义 | Record 字段 |
|---|---|---|
| `tool_start` | 模型决定调用 | `name`、`args`（截断） |
| `tool_result` | 工具执行成功 | `name`、`result`（截断） |
| `tool_error` | 工具执行失败（回填模型） | `name`、`result`（错误文本截断） |
| `tool_partial` | 部分失败提示（前端 ⚠️） | `name`、`result`（失败说明） |

父包逻辑见 [agent.go](internal/agent/agent.go#L369-L401)：`emitToolStart` / `emitToolResult` / `DrainPartials`。注意 `emitToolResult` 会检查"最近一条同名记录是否为 tool_error"，失败态不会被 result 覆盖。

---

## 8. 前端工具状态提示（动作文案按需维护）

Agent 工具的前端状态提示集中在 `frontend/src/js/ai-chat.js`。**工具名直接展示后端下发的英文名**（web_search / recall_notes / ...），前端不维护中文名映射，新增工具无需改动前端展示名。

### 8.1 展示名规则

`getToolLabel(name)`（[ai-chat.js#L4045-L4047](frontend/src/js/ai-chat.js#L4045-L4047)）直接返回英文工具名（带 `「」` 展示）：

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
- 需要展示具体动作（如 web_search 的"搜索 {query}"、recall_notes 的"检索本地笔记"）时，在 [showToolStatusStart](frontend/src/js/ai-chat.js#L2530-L2547) 的 switch 中补一个 `name` 分支；无特殊展示需求走默认"执行"即可。
- 失败原因 / 部分失败说明来自 `payload.result`，截断 40 字符展示。

### 8.3 历史回放明细（折叠组件）

[renderToolCalls](frontend/src/js/ai-chat.js#L4055-L4115) 只渲染 `tool_start` 记录，每条 = 图标 + 加粗工具名 `「X」`（`.ai-tool-status-name`）+ 状态文本：

- 完成：`「web_search」：已完成`
- 失败：`「web_search」：失败：{原因前40字}`（原因取自 `tool_error` 记录 `result`）
- 部分失败：`「web_search」：部分来源失败：{说明前40字}`（取自 `tool_partial` 记录 `result`）

### 8.4 前端维护自查

- [ ] 无需维护中文名映射（工具名直接展示英文）。
- [ ] `showToolStatusStart` 按需补了动作分支（否则默认"执行"）？
- [ ] 实时状态条与历史明细均保持 `「工具名」：状态` 的格式风格？
- [ ] `npm run build` 通过？
