# Agent 工具开关配置 Spec

## Why

Agent 模式目前 10 个内置工具在 [registry.go](internal/agent/registry.go#L19-L31) 硬编码全量注册，模型每轮都会收到全部工具 schema。用户无法控制：
- **数据修改风险**：`manage_note` / `manage_todo` / `manage_notebook` / `manage_tag` 会真实增删改数据，且无二次确认；
- **成本与隐私**：`web_search`（付费/延迟）、`recall_notes`（读取本地笔记库）非人人需要；
- **弱模型误判**：工具多时选错/滥用工具概率上升，且多余 schema 浪费 token。

需要提供配置项控制注册给 Agent 的内置工具，默认全部注册（黑名单语义：不配置即全开，未来新增工具自动默认可用）。

## What Changes

- **存储（无需新增表）**：复用现有 `settings` KV 表，新增键 `ai_agent_tools_disabled`（JSON 数组字符串，如 `["manage_note","web_search"]`，默认空 = 全部注册）。`SettingsConfig` 新增 `AIAgentToolsDisabled string` 字段，`GetAllSettings` / `SaveAllSettings` 各加一行映射。
- **后端工具元数据清单**：`internal/agent/tools` 新增工具元数据（英文名 + 中文说明），作为前端渲染与过滤的单一事实来源。
- **后端注册级过滤**：`buildTools` 按禁用集合过滤工具；`agent.Request` 新增 `DisabledTools []string` 字段，`Run` 读取并传入。
- **后端绑定方法**：`app.go` 新增 `GetAgentTools()` 返回工具清单（名称 + 中文说明 + 当前启用状态）供前端渲染；`CallAIAgentStream` 读取设置解析禁用列表传入 `agent.Run`。
- **前端设置项**：「对话与搜索」面板（dialog-search）尾部新增「Agent 工具」设置项：标签 + 描述 + 右侧按钮（显示「已启用 N/10」），点击展开下拉浮层，浮层内 10 个多选条目（checkbox + 英文工具名 + 中文说明），默认全选，勾选变化即时保存。
- MCP 外部工具不受影响（已有独立 enabled 配置）。

## Impact

- Affected specs: add-ai-agent-mode（Agent 模式工具装配）、add-agent-mcp-server（MCP 工具装配不受影响）
- Affected code:
  - [registry.go](internal/agent/registry.go)（buildTools 过滤）
  - [agent.go](internal/agent/agent.go)（Request 透传 DisabledTools）
  - [types.go](internal/agent/types.go)（Request 新增字段）
  - [internal/agent/tools/](internal/agent/tools/)（新增工具元数据清单）
  - [internal/services/types.go](internal/services/types.go)（SettingsConfig 新增字段 + 映射）
  - [app.go](app.go)（GetAgentTools 绑定 + CallAIAgentStream 读取设置）
  - [frontend/index.html](frontend/index.html)（dialog-search 面板设置项 + 浮层容器）
  - [frontend/src/main.js](frontend/src/main.js)（loadSettings / saveSettings 处理新设置项）
  - [frontend/src/css/components/settings-panel.css](frontend/src/css/components/settings-panel.css)（浮层样式）

## ADDED Requirements

### Requirement: 后端工具元数据清单

`internal/agent/tools` 包 SHALL 提供导出清单 `BuiltinTools() []ToolMeta`（字段：`Name` 英文工具名、`Label` 中文说明、固定分组用途），覆盖全部 10 个内置工具，作为前端渲染与后端过滤的单一事实来源。新增工具时需在此清单追加（与 doc.go 同步约定）。

#### Scenario: 清单返回全部工具
- **WHEN** 调用 `BuiltinTools()`
- **THEN** 返回 10 个工具元数据（refine_search_query / web_search / recall_notes / get_current_time / manage_todo / manage_notebook / manage_tag / manage_note / get_stats / ask_user），每个含中文说明

### Requirement: 后端注册级过滤

`buildTools` SHALL 接收禁用工具名集合，构造工具列表后按集合过滤，被禁用的工具不注册进 eino ChatModelAgent（模型不可见、不会尝试调用）。`agent.Request` SHALL 新增 `DisabledTools []string` 字段；`agent.Run` 将禁用集合解析为 map 传入 `buildTools`。

#### Scenario: 禁用工具后模型不可见
- **WHEN** 配置禁用 `web_search` 且用户发起 Agent 对话
- **THEN** Agent 装配的工具列表不含 `web_search`，模型不会调用该工具，其余工具正常可用
- **AND** 禁用列表为空（默认）时全部 10 个工具正常注册，行为与现状一致

### Requirement: 设置存储与读取

系统 SHALL 通过全局设置键 `ai_agent_tools_disabled` 持久化禁用工具名单（JSON 数组字符串，默认空）。`SettingsConfig` SHALL 新增 `AIAgentToolsDisabled string` 字段（json: `ai_agent_tools_disabled`），`GetAllSettings` / `SaveAllSettings` 读写映射。`CallAIAgentStream` SHALL 读取该设置，解析 JSON 数组后作为 `Request.DisabledTools` 传入 `agent.Run`；解析失败按空列表处理（全部注册）。

#### Scenario: 设置持久化与生效
- **WHEN** 用户在前端勾选禁用 `manage_note` 并保存
- **THEN** `ai_agent_tools_disabled` 键存为 `["manage_note"]`，重启应用后仍生效
- **AND** 后续 Agent 对话中 `manage_note` 不再注册

### Requirement: 前端工具清单绑定

`app.go` SHALL 提供 `GetAgentTools() []agent.ToolMeta` 绑定方法：返回全部内置工具元数据，并标注当前启用状态（未在禁用列表中即为启用）。前端浮层据此渲染条目与默认勾选状态，避免前后端重复维护清单。

#### Scenario: 前端获取工具清单
- **WHEN** 设置页打开且前端调用 `GetAgentTools()`
- **THEN** 返回 10 个工具条目，每个含英文名、中文说明、启用状态（与 `ai_agent_tools_disabled` 一致）

### Requirement: 前端设置项与下拉多选浮层

「对话与搜索」设置面板 SHALL 新增「Agent 工具」设置项（位于面板尾部），包含：
- 标签「Agent 工具」+ 描述「控制 Agent 模式可调用的内置工具」；
- 右侧按钮显示「已启用 N/10」（N 为当前启用数），点击展开下拉浮层，再次点击或点击外部/按 ESC 关闭；
- 浮层内按 `GetAgentTools()` 返回顺序渲染条目，每条 = checkbox 勾选框 + 英文工具名 + 中文说明，勾选状态即启用状态；
- 勾选任一条目即时保存（写入 `ai_agent_tools_disabled` 并调用现有 `saveSettings`），按钮文案同步更新启用数；
- 设置页加载（loadSettings）时回填浮层勾选状态与按钮文案。

#### Scenario: 下拉多选启用/禁用工具
- **WHEN** 用户点击「Agent 工具」设置项按钮展开浮层，取消勾选 `manage_note` 条目
- **THEN** 该条目取消勾选，保存生效，按钮文案更新为「已启用 9/10」
- **AND** 重新打开设置页后 `manage_note` 保持未勾选（持久化）

#### Scenario: 默认全部注册
- **WHEN** 首次使用（`ai_agent_tools_disabled` 为空）
- **THEN** 按钮显示「已启用 10/10」，浮层内 10 个条目全部勾选

## MODIFIED Requirements

无。

## REMOVED Requirements

无。
