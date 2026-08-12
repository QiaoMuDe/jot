# Agent 模式最大运行次数可配置 Spec

## Why
AI 助手的 Agent 模式最大运行次数（ReAct 循环最大迭代次数）在 [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L41-L42) 中硬编码为 20，用户无法按需调整。需要将其改为可配置项，默认值仍为 20。

## What Changes
- 后端 `SettingsConfig` 新增 `ai_agent_max_iterations` 字段（默认值 20，合法范围 1-100），随 `GetAllSettings` / `SaveAllSettings` 读写。
- Agent 装配不再使用硬编码常量，改为在 `AgentService.Run` 内从设置库读取 `ai_agent_max_iterations`，未配置或非法时回退默认值 20。
- 设置页「对话与搜索」面板新增「Agent 运行上限」数字输入项，加载/保存/越界校验与现有数字设置项保持一致。
- **BREAKING**（仅内部引用）：`internal/agent.MaxIterations` 常量重命名为 `DefaultMaxIterations`，语义变为"未配置时的默认值"；同步更新 `playground/agent-demo/main.go` 中的注释引用。

## Impact
- Affected specs: 无（新功能，不改动既有能力）
- Affected code:
  - [types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go) — `SettingsConfig` 字段、读写与校验
  - [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go) — 常量改名 + `Run` 内读取配置
  - [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html) — 设置项 UI
  - [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) — 加载/保存/change 校验
  - [main.go](file:///d:/峡谷/Dev/本地项目/jot/playground/agent-demo/main.go) — 仅注释引用更新

## ADDED Requirements

### Requirement: Agent 运行上限设置项
系统 SHALL 在设置页「对话与搜索」面板提供「Agent 运行上限」数字输入项，并随全局设置保存/加载。

#### Scenario: 加载设置
- **WHEN** 用户打开设置页
- **THEN** 输入框显示已保存的 `ai_agent_max_iterations` 值；未保存过时显示默认值 20

#### Scenario: 保存设置
- **WHEN** 用户修改输入值并触发 change（失焦/回车）
- **THEN** 值被保存到设置库并提示成功；值 < 1 时重置为 20 并提示，值 > 100 时重置为 100 并提示

### Requirement: Agent 装配读取配置
系统 SHALL 在 Agent 模式对话时使用配置的最大迭代次数，而非硬编码常量。

#### Scenario: 已配置合法值
- **WHEN** 用户发起 Agent 对话且已保存 `ai_agent_max_iterations` 为合法值（1-100）
- **THEN** ChatModelAgent 的 `MaxIterations` 使用该配置值

#### Scenario: 未配置或非法值
- **WHEN** 用户发起 Agent 对话且未保存该配置（或值为空/非数字/≤0）
- **THEN** ChatModelAgent 的 `MaxIterations` 使用默认值 20
