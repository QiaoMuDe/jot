# 子 Agent 运行上限可配置 Spec

## Why

os_agent 内层子 Agent 的 ReAct 循环上限目前硬编码为 `osSubAgentMaxIterations = 20`（[subagent_os.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_os.go#L23-L24)），用户无法按任务复杂度调整。主 Agent 已有同类设置项 `ai_agent_max_iterations`，子 Agent 缺失对应能力。

同时探索时发现主 Agent 该设置存在**三处取值不一致**：HTML `max=500`、前端 change 校验上限 `100`、后端 clamp 上限 `500`，导致「有效上限」实际被前端截到 100，UI 提示与后端口径分叉。本次一并统一。

## What Changes

- 后端新增设置项 `ai_sub_agent_max_iterations`，默认 **50**，合法范围 **1–200**，随 `GetAllSettings` / `SaveAllSettings` 读写。
- 主 Agent 设置项 `ai_agent_max_iterations` **默认值 20 → 100**（范围统一为 1–500），三处口径（HTML / 前端校验 / 后端 clamp）对齐。
- os_agent 装配不再使用硬编码常量，改为在 `buildOSSubAgent` 内读取设置（未配置/非法回退默认 50，越上限取 200）。
- 设置页「对话与搜索」面板「Agent 运行上限」下方新增「子 Agent 运行上限」数字输入项，加载/保存/越界提示与相邻设置项一致。
- **BREAKING**（仅内部引用）：`osSubAgentMaxIterations` 常量重命名为 `osSubAgentDefaultMaxIterations` 且值 20 → 50，语义变为「未配置时的默认值」；`buildOSSubAgent` 增加 `setting *services.SettingService` 参数。

## Impact

- Affected specs: `add-agent-max-iterations-setting`（[spec.md](file:///d:/峡谷/Dev/本地项目/jot/.trae/specs/add-agent-max-iterations-setting/spec.md)）——其「默认 20 / 范围 1-100」的约定被本次修正为「默认 100 / 范围 1-500」。
- Affected code:
  - [db.go](file:///d:/峡谷/Dev/本地项目/jot/internal/database/db.go) — 种子默认值
  - [types.go](file:///d:/峡谷/Dev/本地项目/jot/internal/services/types.go) — `SettingsConfig` 字段、读写映射、clamp
  - [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go) — `DefaultMaxIterations` 常量值
  - [subagent_os.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_os.go) — 常量改名 + 读取设置
  - [registry.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/registry.go) — 装配调用点传参
  - [subagent_test.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/subagent_test.go) — 8 处调用点补参
  - [index.html](file:///d:/峡谷/Dev/本地项目/jot/frontend/index.html) — 两个设置项 UI
  - [main.js](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js) — 主 Agent 默认值/校验修正 + 子 Agent 三处逻辑
  - [SUBAGENTS.md](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/SUBAGENTS.md) — 子 Agent 迭代上限文档同步
  - [AGENTS.md](file:///d:/峡谷/Dev/本地项目/jot/AGENTS.md) — 长期记忆 20 表述 + 临时记忆

## ADDED Requirements

### Requirement: 子 Agent 运行上限设置项

系统 SHALL 在设置页「对话与搜索」面板提供「子 Agent 运行上限」数字输入项，并随全局设置保存/加载。

#### Scenario: 加载设置
- **WHEN** 用户打开设置页
- **THEN** 输入框显示已保存的 `ai_sub_agent_max_iterations` 值；未保存过时显示默认值 50

#### Scenario: 保存合法值
- **WHEN** 用户修改输入值（1–200）并触发 change（失焦/回车）
- **THEN** 值被保存到设置库并提示保存成功

#### Scenario: 越界值回退
- **WHEN** 用户输入 < 1 或 > 200
- **THEN** 输入框重置为 50（< 1）或 200（> 200）并给出警告提示，不保存越界值

### Requirement: 子 Agent 装配读取配置

系统 SHALL 在装配 os_agent 子 Agent 时使用配置的迭代上限，而非硬编码常量。

#### Scenario: 已配置合法值
- **WHEN** 用户发起 Agent 对话且已保存 `ai_sub_agent_max_iterations` 为合法值（1–200）
- **THEN** 内层 `ChatModelAgent` 的 `MaxIterations` 使用该配置值

#### Scenario: 未配置或非法值
- **WHEN** 未保存该配置（或值为空/非数字/≤0，或 SettingService 为 nil）
- **THEN** 内层 `MaxIterations` 使用默认值 50

#### Scenario: 设置即时生效
- **WHEN** 用户在对话进行中修改该设置并保存
- **THEN** 下一条 AI 消息的 os_agent 装配即采用新值（无需重启应用）

## MODIFIED Requirements

### Requirement: Agent 运行上限设置项（主 Agent）

系统 SHALL 在设置页「对话与搜索」面板提供「Agent 运行上限」数字输入项，默认值 **100**，合法范围 **1–500**；HTML `max` 属性、前端 change 校验、后端 clamp 三处口径必须一致。

#### Scenario: 新装用户默认值
- **WHEN** 新用户（全新库）打开设置页
- **THEN** 输入框显示 100

#### Scenario: 存量用户
- **WHEN** 存量用户打开设置页
- **THEN** 输入框显示其原有值（种子仅增量插入缺失键，存量值不被覆盖）

#### Scenario: 越界值回退
- **WHEN** 用户输入 < 1 或 > 500
- **THEN** 输入框重置为 100（< 1）或 500（> 500）并给出警告提示

#### Scenario: 后端 clamp 一致
- **WHEN** `SaveAllSettings` 收到越界值（绕过前端直接调用）
- **THEN** 后端按 `< 1 → 100`、`> 500 → 500` 兜底

## REMOVED Requirements

无。
