# Agent 待办管理工具 Spec

## Why
Agent 模式目前只能搜索、召回笔记，无法操作用户的待办清单。用户在对话中提出"帮我记个待办 / 看看我有哪些待办 / 把 XX 标为完成"时，模型无工具可用。新增 `manage_todo` 工具，让 Agent 可以直接创建、查询、勾选待办。

## What Changes
- 新增 Agent 工具 `manage_todo`（一个工具、三个动作）：创建待办、列出待办（未完成/所有/已完成）、勾选待办（完成 ↔ 取消完成）。
- 复用现有 `services.TodoService` 的 `Create` / `List` / `Toggle`，不新增 service API（列表过滤在工具内按 `Done` 字段完成，数据量小、不侵入现有 app.go 调用）。
- Agent 依赖注入 `Deps` 增加 `Todo *services.TodoService`；`registry.go` 注册工具；`app.go` 构造 `AgentService` 时传入已创建的 `todoService`。
- 更新工具清单文档（tools/doc.go）与维护指南（internal/agent/TOOLS.md）。
- 前端状态条按需增加 `manage_todo` 动作文案分支（保持 §8 风格）。

## Impact
- Affected specs: Agent 工具体系（现有工具：web_search / recall_notes / refine_search_query / get_current_time）
- Affected code:
  - `internal/agent/tools/manage_todo.go`（新建）
  - `internal/agent/agent.go`（Deps 增加 Todo 字段）
  - `internal/agent/registry.go`（注册 manage_todo）
  - `internal/agent/tools/doc.go`（工具清单）
  - `internal/agent/TOOLS.md`（维护指南：依赖列 + 工具清单 + 前端动作文案）
  - `app.go`（AgentService 注入 TodoService）
  - `frontend/src/js/ai-chat.js`（可选：showToolStatusStart 增加动作分支）

## ADDED Requirements

### Requirement: manage_todo 工具
系统 SHALL 提供一个名为 `manage_todo` 的 Agent 工具，通过 `action` 参数区分三种操作：`create`（创建）、`list`（列出）、`toggle`（勾选/切换完成状态）。

#### Scenario: 创建待办
- **WHEN** 模型调用 `manage_todo`，`action="create"` 且提供非空 `text`
- **THEN** 调用 `TodoService.Create(text)` 创建待办，返回包含新待办 ID、文本、未完成状态的文本结果；`text` 为空或非法 `action` 时返回错误

#### Scenario: 列出待办
- **WHEN** 模型调用 `manage_todo`，`action="list"`，`status` 为 `active` / `done` / `all`（缺省视为 `active`）
- **THEN** 调用 `TodoService.List()` 并在工具内按 `Done` 过滤，返回格式化列表（每条含 ID、文本、完成状态、时间）与统计（总数/未完成/已完成）；空列表返回明确提示

#### Scenario: 勾选待办
- **WHEN** 模型调用 `manage_todo`，`action="toggle"` 且提供有效 `id`
- **THEN** 调用 `TodoService.Toggle(id)` 切换完成状态，返回切换后的状态文案（"已标记为完成" / "已恢复为未完成"）；`id` 不存在或非法时返回错误，由 WrapWithError 回填模型

#### Scenario: 参数校验
- **WHEN** `action` 不是 create/list/toggle、`create` 缺 `text`、`toggle` 缺 `id` 或 `id<=0`
- **THEN** 工具返回明确错误信息（不中断 ReAct 循环）

## MODIFIED Requirements

### Requirement: Agent 依赖注入（Deps）
`agent.Deps` 增加 `Todo *services.TodoService` 字段；`app.go` 构造 `NewAgentService` 时传入现有 `todoService` 实例。

### Requirement: 工具注册
`registry.go` 的 `buildTools` 增加一行 `tools.WrapWithError("manage_todo", tools.NewManageTodo(p.deps.Todo, p.ctx), p.ctx)`。

### Requirement: 工具清单文档
`tools/doc.go` 与 `internal/agent/TOOLS.md` 的工具清单同步登记 `manage_todo` 及其依赖。

## REMOVED Requirements
无
