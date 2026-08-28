# Agent / Plan 模式切换 Spec

## Why

当前 Agent 模式下 `create_plan` / `update_plan` 始终注册并强制使用，简单对话也必须先制定计划，影响效率。需要将"是否走计划流程"提升为显式的模式切换，让日常对话直接执行（Agent 模式），复杂任务走计划流程（Plan 模式）。

## What Changes

- **Session 配置**：`AISessionConfig` / `SessionConfig` 新增 `PlanMode bool` 字段（默认 false = Agent 模式），随会话持久化
- **前端工具栏**：模型选择器左侧新增 Agent/Plan 模式切换按钮（类似主题切换 pill 样式），切换即时生效并保存
- **后端工具过滤**：`buildTools` 新增 `planMode` 参数，Agent 模式下不注册 `create_plan` / `update_plan`
- **后端 GenModelInput 钩子**：Agent 模式下跳过计划注入逻辑（`genPlanHint` 不再强制要求 create_plan）
- **设置页工具列表**：`ToolMeta` 新增 `PlanOnly` 字段标记仅 Plan 模式可用的工具；`GetAgentTools` 返回时标注 `plan_only` 属性；前端渲染时 Plan 模式工具显示禁用样式、点击抖动并通知提示

## Impact

- Affected specs: `add-ai-agent-mode`（模式切换）、`add-agent-plan-tools`（规划工具）、`add-agent-tool-toggle`（工具开关）
- Affected code:
  - `internal/models/ai_session_config.go`（新增字段）
  - `internal/services/ai_service.go`（SessionConfig 结构 + 读写方法）
  - `internal/database/db.go`（AutoMigrate 兼容）
  - `internal/agent/tools/meta.go`（ToolMeta 新增 PlanOnly 字段）
  - `internal/agent/registry.go`（buildTools 按模式过滤）
  - `internal/agent/agent.go`（genPlanHint 按模式跳过 + Request 透传 PlanMode）
  - `internal/agent/types.go`（Request 新增 PlanMode 字段）
  - `app.go`（GetAgentTools 标注 plan_only + CallAIAgentStream 读取配置传入）
  - `frontend/index.html`（工具栏模式切换按钮 DOM）
  - `frontend/src/js/ai-chat.js`（模式切换逻辑 + 发送时传 mode + 设置页工具列表禁用交互）
  - `frontend/src/css/components/ai-chat.css`（切换按钮样式 + 禁用工具样式）

## ADDED Requirements

### Requirement: 会话级 Agent/Plan 模式切换

系统 SHALL 在 AI 会话配置中保存 `plan_mode`（默认 false = Agent 模式），并在 AI 对话工具栏模型选择器左侧提供切换控件，切换即时生效且随会话持久化；切换不丢失当前会话消息。

#### Scenario: 用户切换为 Plan 模式
- **WHEN** 用户在工具栏切换到"Plan"
- **THEN** 会话配置保存 `plan_mode=true`，后续提问走计划流程（create_plan → 逐步执行 → update_plan），`create_plan` / `update_plan` 工具注册到 Agent

#### Scenario: 用户切回 Agent 模式
- **WHEN** 用户在 Plan 模式下切回"Agent"
- **THEN** 会话配置保存 `plan_mode=false`，后续提问直接执行不制定计划，`create_plan` / `update_plan` 工具不注册

#### Scenario: 新会话默认为 Agent 模式
- **WHEN** 创建新 AI 会话
- **THEN** 默认 `plan_mode=false`（Agent 模式）

### Requirement: 后端按模式过滤计划工具

`buildTools` SHALL 接收 `planMode bool` 参数，Agent 模式（`planMode=false`）下跳过 `create_plan` 和 `update_plan` 的注册。`agent.Request` SHALL 新增 `PlanMode bool` 字段，`Run` 读取并传入 `buildTools`。

#### Scenario: Agent 模式下不注册计划工具
- **WHEN** `plan_mode=false` 且用户发起 Agent 对话
- **THEN** 装配的工具列表不含 `create_plan` / `update_plan`，模型不可见、不会调用

#### Scenario: Plan 模式下正常注册计划工具
- **WHEN** `plan_mode=true` 且用户发起 Agent 对话
- **THEN** 装配的工具列表包含 `create_plan` / `update_plan`，行为与现状一致

### Requirement: 后端 GenModelInput 钩子按模式跳过计划注入

`genPlanHint` SHALL 根据 `req.PlanMode` 决定是否注入计划相关提示。Agent 模式下不注入"请先调用 create_plan"等提示，模型自由回答；Plan 模式下保持现有注入逻辑不变。

#### Scenario: Agent 模式下跳过计划提示
- **WHEN** `plan_mode=false` 且 ReAct 循环开始
- **THEN** `genPlanHint` 返回空字符串，不向模型注入计划约束

#### Scenario: Plan 模式下保持计划提示
- **WHEN** `plan_mode=true` 且 ReAct 循环开始
- **THEN** `genPlanHint` 按现有逻辑注入计划状态/进度/强制提醒

### Requirement: 后端结果兜底按模式调整

Agent 模式下，结果汇总处不做计划相关的兜底处理（不补建计划、不补标步骤）。Plan 模式下保持现有兜底逻辑。

#### Scenario: Agent 模式下无计划兜底
- **WHEN** `plan_mode=false` 且 Agent 回答完成
- **THEN** 跳过计划补建/补标逻辑，`result.Plan` 为空

### Requirement: 前端工具栏模式切换控件

AI 对话工具栏 SHALL 在模型选择器左侧提供 Agent/Plan 模式切换 pill 按钮组，包含两个选项（Agent / Plan），点击切换即时生效。

#### Scenario: 切换按钮展示与交互
- **WHEN** 用户打开 AI 对话
- **THEN** 工具栏模型选择器左侧显示 pill 按钮组，当前激活模式高亮（accent 背景）
- **WHEN** 用户点击非激活模式
- **THEN** 切换激活态、保存配置、通知提示"已切换到 Plan/Agent 模式"

#### Scenario: 切换会话时同步按钮状态
- **WHEN** 用户切换到另一个 AI 会话
- **THEN** 按钮组激活态跟随该会话的 `plan_mode` 配置同步更新

### Requirement: 设置页工具列表中 Plan 模式工具禁用展示

`ToolMeta` SHALL 新增 `PlanOnly bool` 字段标记仅 Plan 模式可用的工具（`create_plan` / `update_plan`）。`GetAgentTools` 返回时标注 `plan_only` 属性。设置页工具列表中 Plan 模式工具显示为禁用样式。

#### Scenario: Plan 模式工具列表项禁用样式
- **WHEN** 用户打开设置页「Agent 工具」管理面板
- **THEN** `create_plan` / `update_plan` 条目显示为禁用样式（灰色文字 + checkbox disabled + 底部说明文字"仅 Plan 模式可用"）

#### Scenario: 点击禁用工具条目触发抖动与通知
- **WHEN** 用户点击 `create_plan` / `update_plan` 条目（整行或 checkbox）
- **THEN** 该条目播放短暂左右抖动动画（shake 0.4s），同时弹出 Toast 通知"此工具仅在 Plan 模式下可用，请切换到 Plan 模式"
- **AND** checkbox 状态不变化，禁用列表不更新

### Requirement: 后端 Session 配置读写

`GetSessionConfig` / `SaveSessionConfig` / `CreateDefaultSessionConfig` / `LoadSessionConfig` SHALL 透传新增的 `plan_mode` 字段。`AISessionConfig` 模型新增 `PlanMode` 列（`gorm:"default:false"`）。

#### Scenario: 配置持久化
- **WHEN** 用户切换到 Plan 模式
- **THEN** `ai_session_configs` 表对应行 `plan_mode` 列写入 `true`，重启后仍可读到

## MODIFIED Requirements

### Requirement: AI 会话配置读写
`GetSessionConfig` / `SaveSessionConfig` 等现有接口 SHALL 透传新增的 `plan_mode` 字段，前端切换后刷新配置可读到最新值；不改变既有字段语义。

### Requirement: Agent 工具元数据清单
`ToolMeta` 新增 `PlanOnly bool` 字段，`create_plan` / `update_plan` 的 `PlanOnly` 设为 `true`。`BuiltinTools()` 返回时包含此标记。

## REMOVED Requirements

无。本期不删除任何既有功能。
