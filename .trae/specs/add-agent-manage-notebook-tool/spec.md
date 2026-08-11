# Agent 笔记本管理工具（manage_notebook）Spec

## Why
Agent 模式目前只能管理待办（`manage_todo`），无法操作用户的笔记本。用户在对话中提出"帮我建个笔记本 / 把 XX 笔记本重命名 / 看看我有哪些笔记本"时，模型没有可用工具，只能引导用户手动操作。

## What Changes
- 新增 Agent 工具 `manage_notebook`（一个工具、三个动作）：`create` 创建笔记本、`rename` 重命名、`list` 列出全部笔记本。
- 后端复用现有 `services.NotebookService`（`Create` / `Update` / `GetAll`），**不新增**服务方法、不动数据模型。
- `agent.Deps` 增加 `Notebook *services.NotebookService`；`app.go` 的 `NewApp` 构造 `AgentSvc` 处传入 `Notebook: notebookService`。
- `registry.go` 的 `buildTools` 注册 `manage_notebook`（与既有工具一致用 `tools.WrapWithError` 包装）。
- 文档同步：`internal/agent/tools/doc.go`、`internal/agent/doc.go`、`internal/agent/TOOLS.md`（§6 工具清单表 + §3 注册示例代码块）。
- 前端 `frontend/src/js/ai-chat.js` 的 `showToolStatusStart` 增加 `manage_notebook` 分支动作文案（创建/重命名/列出），未匹配 action 走默认"执行"。

## Impact
- Affected specs: Agent 工具能力（新增写操作工具，与 `manage_todo` 并列）
- Affected code:
  - `internal/agent/tools/manage_notebook.go`（新建）
  - `internal/agent/registry.go`（注册 manage_notebook）
  - `internal/agent/agent.go`（Deps 增加 Notebook 字段）
  - `app.go`（NewApp 构造 AgentSvc 处传入 Notebook）
  - `internal/agent/tools/doc.go`、`internal/agent/doc.go`、`internal/agent/TOOLS.md`（文档同步）
  - `frontend/src/js/ai-chat.js`（动作文案分支）

## ADDED Requirements

### Requirement: manage_notebook 工具
系统 SHALL 提供一个名为 `manage_notebook` 的 Agent 工具，通过 `action` 参数区分三种操作：`create`（创建）、`rename`（重命名）、`list`（列出）。工具内部复用 `services.NotebookService`，不感知父包 agent 的事件循环细节，经 `tools.WrapWithError` 包装（失败回填模型、不中断 ReAct 循环）。

#### Scenario: 创建笔记本成功
- **WHEN** 模型调用 `manage_notebook`，`action="create"` 且提供非空 `name`
- **THEN** 调用 `NotebookService.Create(name)`，成功返回 `已创建笔记本 #<id>：<name>`
- **AND** 若同名笔记本已存在，返回 `NotebookService` 的"笔记本「X」已存在"错误（经 WrapWithError 回填模型）

#### Scenario: 重命名笔记本成功
- **WHEN** 模型调用 `manage_notebook`，`action="rename"` 且提供正整数 `id` 与非空 `name`
- **THEN** 调用 `NotebookService.Update(id, name)`，成功返回 `已重命名笔记本 #<id> 为：<name>`
- **AND** 业务失败（默认笔记本 id=1 不可重命名、id 不存在、新名称与其他笔记本重名）时返回对应错误文本，模型可据此调整策略

#### Scenario: 列出笔记本
- **WHEN** 模型调用 `manage_notebook`，`action="list"`（无额外参数）
- **THEN** 调用 `NotebookService.GetAll()`，返回所有未删除笔记本的列表，每条格式 `[<id>] <名称> · 创建时间 <2006-01-02 15:04>`，编号 `[数字]` 可用于后续 `rename`
- **AND** 无笔记本时返回 `当前没有任何笔记本`

#### Scenario: 参数缺失或非法
- **WHEN** `action` 缺失或不在枚举内；或 `create`/`rename` 缺 `name`（trim 后为空）；或 `rename` 缺 `id` / `id<=0`
- **THEN** 返回错误（`manage_notebook 参数…`），经 `WrapWithError` 回填模型继续推理，不中断循环

#### Scenario: 用户取消
- **WHEN** 执行前 `ctx.Err() != nil`（父包事件循环已终止）
- **THEN** 直接返回 `ctx.Err()`，不执行任何 DB 操作

### Requirement: 工具注册与依赖注入
系统 SHALL 将 `manage_notebook` 注册进 ReAct 循环，并注入笔记本服务依赖。

#### Scenario: 注册与装配
- **WHEN** Agent 会话启动（`buildTools` 执行）
- **THEN** `manage_notebook` 与其他工具并列注册，构造器 `tools.NewManageNotebook(p.deps.Notebook, p.ctx)`
- **AND** `agent.Deps` 含 `Notebook *services.NotebookService`，`app.go` 的 `NewApp` 将 `notebookService` 传入（`rebuildServices` 已重建 `notebookService`，与 `Todo` 注入方式一致）

### Requirement: 文档与前端动作文案同步
系统 SHALL 同步维护工具文档与前端状态条文案。

#### Scenario: 文档同步
- **WHEN** 新增 `manage_notebook` 工具
- **THEN** `tools/doc.go` 工具清单与构造器名、`agent/doc.go` 结构说明、`TOOLS.md` §6 工具清单表（含依赖注入列）均登记 `manage_notebook` / `NewManageNotebook`

#### Scenario: 前端动作文案
- **WHEN** 工具开始调用（`tool_start` 事件，`name="manage_notebook"`）
- **THEN** 状态条按 `args.action` 展示：`create`→"创建笔记本"、`rename`→"重命名笔记本"、`list`→"列出笔记本"，未匹配走默认"执行"；工具展示名沿用英文名直显，不维护中文映射
