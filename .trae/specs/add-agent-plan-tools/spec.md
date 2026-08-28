# Agent 规划工具（create_plan / update_plan）文档补充

## Why

当前 Agent 的 ReAct 循环完全依赖 LLM 自主决策，缺乏显式规划能力。用户无法提前知道 Agent 要做什么，复杂任务容易跑偏且 token 浪费严重。通过新增 `create_plan` 和 `update_plan` 两个工具，让模型在执行前先输出结构化计划，执行中动态调整，实现显式 Planning 能力。

本次变更**仅更新 TOOLS.md 开发指南文档**，补充这两个工具的设计规范与事件协议，为后续实现提供文档基础。实际工具代码、注册、前端展示等实现工作待定。

## What Changes

- 在 `TOOLS.md` 的 §7 事件协议速查中新增 `ai:plan-created` 和 `ai:plan-updated` 两个事件的协议说明
- 在 `TOOLS.md` 的 §4.7 约束与红线中补充 `create_plan` / `update_plan` 作为 `ctx.Emit` 的新例外（与 `ask_user` 类似，需直接发射计划卡片事件）
- 在 `TOOLS.md` 中新增 §9 规划工具设计说明，包含数据结构、工具职责、参数格式、与 GenModelInputFunc 钩子的集成方式

## Impact

- Affected specs: 无（本次仅文档变更）
- Affected code: `internal/agent/TOOLS.md`（开发指南）

## ADDED Requirements

### Requirement: TOOLS.md 补充规划工具事件协议

`TOOLS.md` §7 事件协议速查中须新增以下事件说明：

#### Scenario: create_plan 工具发射计划创建事件

- **WHEN** 模型调用 `create_plan` 工具且参数解析成功
- **THEN** 工具直接 `ctx.Emit("ai:plan-created", payload)` 发射事件，payload 为 JSON 字符串，包含 `goal`（目标）和 `steps`（步骤列表，每项含 `id`、`description`、`status`）

#### Scenario: update_plan 工具发射计划更新事件

- **WHEN** 模型调用 `update_plan` 工具且参数解析成功
- **THEN** 工具直接 `ctx.Emit("ai:plan-updated", payload)` 发射事件，payload 为 JSON 字符串，包含 `step_id`、`status`、`result`、`steps`（完整步骤列表快照）

### Requirement: TOOLS.md 补充规划工具作为 ctx.Emit 例外

`TOOLS.md` §4.7 约束与红线中须将 `create_plan` 和 `update_plan` 列为 `ctx.Emit` 的允许例外（与 `ask_user` 并列），说明这两个工具需要直接发射计划卡片事件供前端展示。

### Requirement: TOOLS.md 新增规划工具设计说明章节

`TOOLS.md` 须新增 §9 章节，包含以下内容：

#### Scenario: 数据结构定义

- **WHEN** 维护者查阅规划工具设计
- **THEN** 文档须定义 `Plan` 和 `PlanStep` 数据结构（含 JSON 标签），说明字段含义

#### Scenario: create_plan 工具规范

- **WHEN** 维护者查阅 create_plan 工具设计
- **THEN** 文档须说明工具职责（模型在开始执行前调用，输出结构化计划）、输入参数格式（goal + steps）、参数校验规则（goal 非空、steps 非空且 ≤10）、执行逻辑（构造 Plan → 存入 ctx.PlanState → 发射事件 → 返回确认文本）

#### Scenario: update_plan 工具规范

- **WHEN** 维护者查阅 update_plan 工具设计
- **THEN** 文档须说明工具职责（模型在执行过程中调整计划）、输入参数格式（step_id/status/result/new_step）、执行逻辑（更新 PlanState → 发射事件 → 返回确认文本）

#### Scenario: GenModelInputFunc 钩子集成

- **WHEN** 维护者查阅规划工具与 ReAct 循环的集成方式
- **THEN** 文档须说明通过 eino 的 `GenModelInputFunc` 钩子在每轮 LLM 调用前注入当前计划状态到系统提示词，引导模型按计划执行

## MODIFIED Requirements

### Requirement: TOOLS.md 架构概览补充

`TOOLS.md` §1 架构概览的文件树中须新增 `plan.go`（规划工具实现文件），说明其职责为实现 `create_plan` 和 `update_plan` 两个工具。

## REMOVED Requirements

无
