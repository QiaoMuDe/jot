# 笔记写操作审批机制改造 Spec

## Why
笔记/笔记本/标签/待办的写操作目前依赖**模型自觉**（manage_note 靠 `confirm=true` 参数 + ask_user 对话流引导，manage_notebook/manage_tag/manage_todo 完全无确认），模型不遵守提示词约束即可直接落库，且无审批弹窗与审计留痕。命令执行（run_command/write_file 等）已有**代码强制**的三模式审批机制（confirm_every / review / auto），本次将笔记类写操作并入同一机制，依赖代码而非模型自觉。

## What Changes
- **manage_note**（[manage_note.go](internal/agent/tools/manage_note.go)）：移除 `confirm` 参数与"先问用户再带 confirm=true"引导逻辑；写操作（update/edit/pin/move/add_tag/remove_tag）执行前调用 `Context.Approver.RequestApproval`；create 豁免（用户明确要求的创建指令，与现状一致）。
- **manage_notebook / manage_tag / manage_todo**：rename、update、toggle/update 写操作接入同一审批；create 豁免。
- **critical 分级**：结构性/批量操作 critical=true（edit 正文、move 批量移动、add_tag/remove_tag 批量打标签）；轻量操作 critical=false（update/pin/rename/tag 单条/toggle）。
- **manage_memory 豁免**：长期记忆维护规范鼓励 Agent 主动保存跨会话信息，不接入审批。
- 工具描述同步改写：写操作由"先 ask_user 确认"改为"按当前审批模式弹出确认面板"。
- 前端零改动：审批弹窗（ai:tool-approval）、tool_approval/tool_auto_approval 留痕、历史回放全部复用现有命令审批链路。

## Impact
- Affected specs: add-agent-manage-note-tool（确认语义变更）、add-agent-manage-notebook-tool、add-agent-manage-tag-tool、add-agent-manage-todo-tool
- Affected code: `internal/agent/tools/manage_note.go`、`internal/agent/tools/manage_notebook.go`、`internal/agent/tools/manage_tag.go`、`internal/agent/tools/manage_todo.go`、`internal/agent/tools/context.go`（注释）、`internal/agent/TOOLS.md`、测试文件
- 不受影响：registry.go（已传 ctx）、agent.go（门控逻辑）、前端、os_agent 子 Agent（白名单不含笔记工具）、app.go 绑定层

## ADDED Requirements

### Requirement: 管理工具写操作统一审批接入
manage_notebook / manage_tag / manage_todo 的写操作 SHALL 在落库前经 `Context.Approver.RequestApproval` 门控，与 run_command 同路径。

#### Scenario: rename/update/toggle 写操作触发审批
- **WHEN** Agent 调用 manage_notebook(rename)、manage_tag(update)、manage_todo(toggle/update) 且未传入 create
- **THEN** 工具先调用 RequestApproval（summary 含动作与目标详情，critical=false），确认_every 模式弹出审批面板，批准后执行、拒绝返回拒绝错误文本回填模型

#### Scenario: create 动作免确认
- **WHEN** Agent 调用上述工具且 action=create
- **THEN** 直接执行，不触发审批

## MODIFIED Requirements

### Requirement: manage_note 写操作由模型自觉确认改为代码强制审批
manage_note 的 update/edit/pin/move/add_tag/remove_tag 写操作 SHALL 移除 `confirm` 参数门控，改为执行前调用 `Context.Approver.RequestApproval`。

#### Scenario: 写操作审批门控
- **WHEN** Agent 调用 manage_note 且 action 为写操作
- **THEN** 工具按 critical 分级调用 RequestApproval：edit（正文编辑）与批量 move/add_tag/remove_tag（ids>1）为 critical=true；update/pin/单条 move/add_tag/remove_tag 为 critical=false；confirm_every 全确认，review 仅 critical 确认，auto 自动放行（critical 留痕）

#### Scenario: 拒绝后的行为
- **WHEN** 用户在审批面板点击拒绝
- **THEN** 工具返回中文拒绝错误文本（经 wrappedTool 记 tool_error 回填模型），不落库

### Requirement: 工具描述与文档同步
manage_* 工具描述 SHALL 移除"先调用 ask_user 确认"引导，改写为"写操作按当前审批模式弹出确认面板"；TOOLS.md 同步更新。

## REMOVED Requirements

### Requirement: manage_note 的 confirm 参数门控
**Reason**: 模型自觉机制不可靠，与命令审批双轨并存造成语义分裂。
**Migration**: `confirm` 参数从 schema 与解析中移除；历史行为由审批门控替代（confirm_every 模式等效于必须人工确认，review/auto 按分级放行）。
