# 移除执行阶段的 create_plan 工具

## 摘要

Plan-and-Exec 解耦后，计划已在预规划阶段通过 `generatePlan()` 强制生成。执行阶段（ReAct 循环）不再需要 `create_plan` 工具，移除以避免模型重复生成计划。

## 改动

### 1. registry.go — 过滤掉 create_plan

**文件**：`internal/agent/registry.go`

- 从 `planOnlyTools` 中移除 `create_plan`（仅保留 `update_plan`）
- 新增 `planExecExcluded` 集合：`{"create_plan": true}` — Plan 模式执行阶段排除的工具
- 在过滤循环中增加：`planMode && planExecExcluded[n.name]` → skip

### 2. agent.go — 移除兜底补建逻辑

**文件**：`internal/agent/agent.go`

- 移除第 882-895 行的 `else if len(toolRecords) > 0` 分支（模型跳过 create_plan 时自动补建单步计划）。Plan-and-Exec 模式下 PlanState 一定已在预规划阶段设置，这个兜底不再需要。

### 3. agent.go — genPlanHint 移除防御性兜底

**文件**：`internal/agent/agent.go`

- `genPlanHint()` 第 1090 行的 `plan == nil` 分支（引导"必须先用 create_plan"）移除。
- Plan-and-Exec 模式下，`generatePlan()` 失败时 `Run()` 已提前返回 error，不会到达 `genPlanHint`。
- Agent 模式（`PlanMode=false`）不调用 `genPlanHint`，也不会到达。
- 因此 `plan == nil` 不应发生，直接返回空串即可（防御性：不崩溃、不注入垃圾文本）。

## 不改的文件

- `tools/plan.go` — `ParseCreatePlanArgs` 和 `createPlanTool` 保留（`generatePlan()` 仍使用 `create_plan` 的 ToolInfo 进行 function calling）
- `tools/meta.go` — `BuiltinTools()` 中 `create_plan` 的 `PlanOnly: true` 保留（前端工具清单仍显示）
- 前端 — 无改动

## 验证

1. Plan 模式：预规划生成计划 → 执行阶段模型只有 `update_plan`，不会调用 `create_plan`
2. Agent 模式：不注册任何 plan 工具（`planMode=false` 时 `planExecExcluded` 不生效），行为不变
3. 编译通过
