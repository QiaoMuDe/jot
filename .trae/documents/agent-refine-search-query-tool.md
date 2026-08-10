# Agent 搜索词精炼工具（方案 B）实施计划

## Summary

为 Agent 模式新增独立工具 `refine_search_query`（搜索词精炼），由模型在 ReAct 循环中自主判断：当用户输入模糊/口语化时，先调用精炼工具得到精炼词，再调用 `web_search` 用精炼词搜索。前端复用现有工具调用展示框架，精炼作为独立的工具调用条目展示与落库。

## Current State Analysis

- 工具注册：`internal/agent/agent.go` 的 `Run` 中构造 `webSearchTool` / `recallNotesTool` 并注册进 `adk.ToolsConfig`
- 精炼能力：`internal/services/query_refiner.go` 的 `RefineSearchQuery(ctx, query, *AIService) (string, error)` 可直接复用，webSearchTool 内部已有 `ai *services.AIService` 依赖
- 事件链路：agent.go 的 `emitToolStart` / `emitToolResult` 通用发射 `ai:tool-status`，前端 `showToolStatusStart`（[ai-chat.js L2463](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2463)）/ `showToolStatusDone`（[L2499](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2499)）按 `payload.name` 分支渲染；`TOOL_NAMES`（[L2437](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L2437)）与 `renderToolCalls` 的 `TOOL_LABELS`（[L3902](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L3902)）均有 `web_search` / `recall_notes` 映射
- 落库：`tool_calls` 字段（JSON 数组）由工具调用通用收集器写入，新工具自动被记录，历史渲染 `renderToolCalls` 只渲染 `tool_start` 条目

## Proposed Changes

### 1. 新增精炼工具 `internal/agent/tools.go`

新增 `refineSearchQueryTool` 结构体（参照 `webSearchTool` 模式）：

```go
type refineSearchQueryTool struct {
    tool.BaseTool
    ai *services.AIService
}

func (r *refineSearchQueryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
    return &schema.ToolInfo{
        Name:        "refine_search_query",
        Desc: "将用户口语化、模糊的搜索意图精炼为简洁搜索关键词。" +
            "当 web_search 的查询词是口语化表达、含义模糊、或包含多个话题时，" +
            "先调用本工具精炼，再用精炼后的关键词调用 web_search 进行搜索。",
        Params: schema.NewParamsOneOfByParams(map[string]*schema.Parameter{
            "query": {Type: schema.String, Desc: "用户原始搜索意图，可为口语化句子", Required: true},
        }),
    }, nil
}

func (r *refineSearchQueryTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
    // 解析 query → 检查 ctx.Err() → RefineSearchQuery(ctx, query, r.ai)
    // 精炼失败/无变化：返回原 query 文本（不报错，让模型继续搜索）
    // 成功：返回精炼词文本
}
```

说明：
- 精炼用**当前配置的 AI 模型**（与问答模式 RefineSearchQuery 一致）
- 用户停止（ctx.Err()）时直接返回错误终止
- 失败降级返回原 query，不中断 Agent 循环

### 2. 更新 `webSearchTool` 的 Desc（`internal/agent/tools.go`）

在 `webSearchTool.Info()` 的 Desc 末尾追加引导，促使模型先精炼再搜索：

> "如果查询词是口语化表达、含义模糊或包含多个话题，请先调用 refine_search_query 工具精炼，再使用精炼后的关键词作为本工具的 query。"

### 3. 注册新工具 `internal/agent/agent.go`

`Run` 中构造工具列表处追加：

```go
refineTool := &refineSearchQueryTool{ai: service.ai}
```

并加入 `adk.ToolsConfig` 的 `Tools` 数组（放在 webSearchTool 之前）。

### 4. 前端映射与展示 `frontend/src/js/ai-chat.js`

- `TOOL_NAMES`（L2437）加：`refine_search_query: '搜索词精炼'`
- `showToolStatusStart`（L2463）加分支：`name === 'refine_search_query'` → label `'正在精炼搜索词...'`（不展示 query 参数，精炼词结果在完成时展示）
- `showToolStatusDone`（L2499）加分支：`name === 'refine_search_query'` → label `'已精炼'`，展示精炼结果（`已精炼：原词 → 精炼词` 可省略，保持简洁为 `已精炼`）
- `renderToolCalls` 的 `TOOL_LABELS`（L3902）加：`refine_search_query: '搜索词精炼'`
- `renderToolCalls` 的 doneLabel 分支加：`name === 'refine_search_query'` → `'已完成精炼'`

### 5. 无需改动

- `agent.Result` / collector：refine 不产生来源与卡片，收集逻辑复用，无需改动
- 落库：`tool_calls` 通用收集自动包含 refine 记录
- `app.go` / 问答模式：完全不动

## Assumptions & Decisions

- **决策**：采用方案 B（独立工具，模型自主判断），放弃方案 A（硬编码到 web_search 内部）
- **假设**：deepseek-v4-flash / qwen3 等目标模型具备稳定的多工具编排能力，会按 Desc 引导先精炼再搜索；若实测模型跳过精炼，可再在 web_search 内部加兜底（后续任务）
- **决策**：refine 工具失败时降级返回原词而非报错——Agent 循环不中断，模型仍可继续搜索
- **决策**：精炼词不落库到 `search_sources`（搜索来源仍是 web_search 的结果），refine 过程随 `tool_calls` 落库回放

## Verification

1. `go build ./...`、`go vet ./...` 通过
2. `node --check ai-chat.js` 通过
3. `wails dev` 冒烟：
   - Agent 提问口语化搜索问题（如"新 SSR 框架和 Next.js 比怎么样"）→ 展示"正在精炼搜索词... → 已精炼"→"正在搜索：<精炼词> → 已完成搜索"，且最终回答基于精炼词搜索结果
   - Agent 提问精确搜索词 → 模型可能跳过精炼直接搜索（自主决策生效）
   - 重开会话：工具调用记录含"搜索词精炼 → 已完成精炼"条目，与实时一致
   - 问答模式无回归
