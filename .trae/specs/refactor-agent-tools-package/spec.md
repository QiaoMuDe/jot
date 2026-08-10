# Agent 工具子包重构 Spec

## Why

Agent 的工具数量会持续增长（当前 3 个，未来规划有笔记检索/写笔记/待办等）。现实现把全部工具堆在 `internal/agent/tools.go` 单文件（约 350 行），注册是 `agent.go Run()` 里手写的工具切片，且事件循环里硬编码了 `name == "web_search"` 的部分失败特判。工具增多后：文件膨胀、每加一个工具都要改 Run()、特判 if 分支蔓延。需要把工具抽成独立的 `tools` 子包 + 父包统一注册表，让"新增工具"变成标准化两步操作。

## What Changes

- 新增子包 `internal/agent/tools/`，每个工具独立文件：`context.go`、`web_search.go`、`recall_notes.go`、`refine_query.go`。
- 工具共享类型下沉到子包并导出：`EmitFn`、`Record`、`Collector`、`Context`、`WrapWithError`、`TruncateRunes`、`MaxResultLen`。
- 工具结构体保持私有（`webSearchTool` 等），仅导出构造器 `NewWebSearch` / `NewRecallNotes` / `NewRefineSearchQuery`。
- 父包新增 `registry.go` 统一注册表（`buildTools`，一行一个工具），`Run()` 不再编辑工具列表。
- 删除事件循环中 `name == "web_search"` 硬编码特判：改为工具通过 `Context.AddPartial` 登记部分失败信息，父包在 `tool_result` 之后统一 `DrainPartials` 发射 `tool_partial`（事件顺序不变）。
- 删除 `toolCallRecord` / `sourceError` / `resultCollector.SourceErrors`（部分失败信息改为工具本地维护并通过 Context 登记）。
- `internal/agent/types.go` 保留对外契约：`Deps` / `Request` / `Result` / `NewAgentService` 签名不变；`EmitFn` 改为 `type EmitFn = tools.EmitFn` 别名。
- **BREAKING（仅包内部）**：`internal/agent/tools.go` 删除，内容拆分至子包。

## Impact

- Affected code: `internal/agent/`（agent.go / types.go / doc.go / tools.go 删除 / 新增 registry.go）
- Affected code: `internal/agent/tools/`（新增子包）
- 外部无影响：`app.go` 仅引用 `agent.NewAgentService(agent.Deps{...})` 与 `agent.Request`，均不变。
- Affected specs: add-ai-agent-mode（Agent 功能基线，行为不变）

## ADDED Requirements

### Requirement: tools 子包独立文件结构

系统 SHALL 提供 `internal/agent/tools/` 子包，每个工具一个文件（web_search / recall_notes / refine_query），共享类型在 `context.go`，工具实现不得 import `agent` 包。

#### Scenario: 新增工具
- **WHEN** 开发者需要新增一个工具
- **THEN** 在 `tools/` 新建一个文件实现 `Info()` + `InvokableRun()` + 导出构造器，并在父包 `registry.go` 加一行注册，无需改动 `agent.go Run()` 事件循环

### Requirement: 工具上下文与统一事件发射

系统 SHALL 通过 `tools.Context`（含 Emit / Records / Collector / Logger）向工具注入事件能力，并提供 `WrapWithError` 统一包装失败回填、`AddPartial` / `DrainPartials` 处理部分失败提示。

#### Scenario: 部分来源失败
- **WHEN** web_search 成功但部分来源失败
- **THEN** 工具内部调用 `Context.AddPartial` 登记失败来源，父包在发射 `tool_result` 之后调用 `DrainPartials` 发射 `tool_partial`（事件顺序与现实现一致：tool_start → tool_result → tool_partial）

#### Scenario: 工具执行失败
- **WHEN** 工具执行返回 error
- **THEN** 经 `WrapWithError` 包装：记录 `tool_error` 记录并发射事件，错误文本回填模型，ReAct 循环不中断（行为与现实现一致）

### Requirement: 父包统一注册表

系统 SHALL 在 `agent` 包提供 `registry.go` 集中注册全部工具，`Run()` 通过注册表构建工具切片，不再逐工具手写。

## MODIFIED Requirements

### Requirement: Agent 事件循环（agent.go Run）

Run 不再包含任何工具名硬编码特判；工具列表由 `buildTools` 注册表构建；`resultCollector` 的使用改为 `tools.Context.Collector`；`toolCallRecord` 改为 `tools.Record`。

### Requirement: 对外数据契约（types.go）

`Deps` / `Request` / `Result` / `NewAgentService` 保持导出与签名不变；`EmitFn` 以类型别名 `type EmitFn = tools.EmitFn` 保留；`toolCallRecord` / `sourceError` / `resultCollector` 从 agent 包移除（下沉至 tools 包）。

## REMOVED Requirements

### Requirement: 单文件工具实现（tools.go）

**Reason**: 工具增长后单文件膨胀，且注册与事件逻辑耦合在 Run() 中。
**Migration**: 三个工具拆分为 `tools/` 子包独立文件，注册迁移至 `registry.go`；包内引用全部更新，外部调用方（app.go）无感知。

### Requirement: resultCollector.SourceErrors / sourceError 类型

**Reason**: 部分失败提示改由工具本地维护失败列表并经 `Context.AddPartial` 登记，collector 不再需要存储该信息。
**Migration**: 删除相关字段与类型，发射逻辑下沉。
