# Tasks

- [x] Task 1: 创建 tools 子包共享上下文文件 `internal/agent/tools/context.go`
  - [x] 定义 `MaxResultLen` 常量（迁移自 agent.go 的 `maxToolResultLen`）
  - [x] 定义 `type EmitFn func(event string, data string)`（迁移自 types.go）
  - [x] 定义 `type Record struct { Action, Name, Args, Result string }`（json tag 与现 toolCallRecord 一致）
  - [x] 定义 `type Collector struct { Sources []services.SearchSource; Cards []services.RecallCard }`（去掉 SourceErrors 字段）
  - [x] 定义 `type Context struct { Emit EmitFn; Records *[]Record; Collector *Collector; Logger *fastlog.Logger }`，含 `AddPartial(msg string)` 与 `DrainPartials(name string)`（遍历登记的部分失败信息，构造 `Record{Action:"tool_partial"}` 追加 records 并 emit `ai:tool-status`，随后清空）
  - [x] 迁移 `wrapToolWithError` 为 `func WrapWithError(name string, t tool.InvokableTool, ctx *Context) tool.InvokableTool`（逻辑不变：ctx.Err 时不误报失败；记录 tool_error Record 并 emit；返回"工具执行失败：..."回填文本）
  - [x] 迁移 `truncateRunes` 为导出 `TruncateRunes(s string, maxLen int) string`

- [x] Task 2: 创建 `internal/agent/tools/web_search.go`（从 tools.go 迁移 webSearchTool）
  - [x] 迁移搜索源常量与 `defaultSearchSources`、`intSetting` 方法
  - [x] 结构体保持私有 `webSearchTool`，新增导出构造器 `NewWebSearch(ai *services.AIService, setting *services.SettingService, ctx *Context) tool.InvokableTool`
  - [x] `InvokableRun` 中删除对 `collector.SourceErrors` 的三处 append（未配置/运行失败），改为维护本地 `failedParts` 切片
  - [x] 部分失败返回前调用 `ctx.AddPartial(strings.Join(failedParts, "、"))`（保持与现 tool_partial 消息内容一致）
  - [x] 保留"全部来源未配置/全部失败"的错误路径（行为不变）

- [x] Task 3: 创建 `internal/agent/tools/recall_notes.go`（从 tools.go 迁移 recallNotesTool）
  - [x] 结构体保持私有 `recallNotesTool`，新增导出构造器 `NewRecallNotes(vector *services.VectorService, setting *services.SettingService, getEmbedConfig func() (baseURL, apiKey, model string, err error), notebookIDs []uint, ctx *Context) tool.InvokableTool`
  - [x] 保留向量召回逻辑、召回条数设置、embedding client 构建、Cards 收集逻辑

- [x] Task 4: 创建 `internal/agent/tools/refine_query.go`（从 tools.go 迁移 refineSearchQueryTool）
  - [x] 结构体保持私有 `refineSearchQueryTool`，新增导出构造器 `NewRefineSearchQuery(ai *services.AIService) tool.InvokableTool`
  - [x] 保留精炼逻辑（用户停止/失败降级返回原词）

- [x] Task 5: 创建父包注册表 `internal/agent/registry.go`
  - [x] 定义 `type BuildParams struct { deps Deps; req Request; ctx *tools.Context }`
  - [x] 定义 `buildTools(p BuildParams) []tool.BaseTool`，一行一个工具：`tools.WrapWithError("refine_search_query", tools.NewRefineSearchQuery(p.deps.AI), p.ctx)`、`tools.WrapWithError("web_search", tools.NewWebSearch(p.deps.AI, p.deps.Setting, p.ctx), p.ctx)`、`tools.WrapWithError("recall_notes", tools.NewRecallNotes(p.deps.Vector, p.deps.Setting, p.deps.GetEmbedConfig, p.req.RecallNotebookIDs, p.ctx), p.ctx)`
  - [x] 工具顺序与现实现一致（refine_search_query → web_search → recall_notes）

- [x] Task 6: 重构 `internal/agent/agent.go`
  - [x] 删除 `wrapToolWithError` / `truncateRunes` / `maxToolResultLen`（已迁移 tools 包），改用 `tools.WrapWithError` / `tools.TruncateRunes` / `tools.MaxResultLen`
  - [x] `toolRecords` 类型改为 `[]tools.Record`；`emitToolStart` / `emitToolResult` 签名同步改用 `[]tools.Record`
  - [x] Run() 内工具构建改为：创建 `collector := &tools.Collector{}` → 创建 `ctx := &tools.Context{Emit: emit, Records: &toolRecords, Collector: collector, Logger: s.deps.Logger}` → `tools := buildTools(BuildParams{deps: s.deps, req: req, ctx: ctx})`
  - [x] Tool 事件分支中删除 `name == "web_search"` 的 tool_partial 特判块，改为在 `emitToolResult` 后调用 `ctx.DrainPartials(name)`
  - [x] 汇总 Result 时 `collector.Sources` / `collector.Cards` 读取改为 `ctx.Collector.Sources` / `ctx.Collector.Cards`
  - [x] `Deps` 定义保持在 agent.go（或迁移至 deps.go，保持导出与字段不变）

- [x] Task 7: 更新 `internal/agent/types.go`
  - [x] 删除 `toolCallRecord` / `sourceError` / `resultCollector` 定义
  - [x] `EmitFn` 改为 `type EmitFn = tools.EmitFn`（保留对外契约别名）
  - [x] `Result.ToolCalls` 注释改为引用 `tools.Record`；`Result.SearchSources/RecallCards` 注释引用 `tools.Collector`

- [x] Task 8: 更新 `internal/agent/doc.go` 与 `internal/agent/tools/doc.go`（如新建）
  - [x] doc.go 说明新结构：工具在 `tools` 子包独立文件、父包 registry.go 统一注册、新增工具步骤

- [x] Task 9: 删除 `internal/agent/tools.go`

- [x] Task 10: 构建与静态校验
  - [x] `go build ./...` 通过
  - [x] `go vet ./internal/agent/...` 通过
  - [x] `go test ./internal/agent/...`（如有测试）通过
  - [x] 人工核对行为等价点：事件顺序 tool_start → tool_result → tool_partial；tool_error 路径；全部来源未配置的错误文本；`app.go` 编译不受影响

# Task Dependencies

- Task 1（context.go）是基础，Task 2/3/4（工具文件）依赖 Task 1 的类型。
- Task 5（registry.go）依赖 Task 2/3/4 的构造器。
- Task 6（agent.go）依赖 Task 5。
- Task 7（types.go）依赖 Task 1；Task 9（删除 tools.go）必须在 Task 2/3/4 完成之后。
- Task 8（doc）与 Task 10（构建校验）依赖全部前序任务。
- 可并行：Task 2、3、4 在 Task 1 完成后可并行实现。
