# Tasks

- [x] Task 1: 更新 TOOLS.md §1 架构概览 — 在文件树中新增 `plan.go` 文件说明
  - [x] SubTask 1.1: 在 `tools/` 子包文件树中追加 `plan.go` 行，标注职责为"规划工具实现（create_plan / update_plan）"

- [x] Task 2: 更新 TOOLS.md §4.7 约束与红线 — 补充 ctx.Emit 例外
  - [x] SubTask 2.1: 在"唯一例外：`ask_user`"段落中追加 `create_plan` 和 `update_plan` 为新的 `ctx.Emit` 允许例外，说明这两个工具需要直接发射计划卡片事件（`ai:plan-created` / `ai:plan-updated`）供前端展示

- [x] Task 3: 更新 TOOLS.md §7 事件协议速查 — 新增规划事件
  - [x] SubTask 3.1: 在 §7 主表格后新增 §7.2 小节，说明 `ai:plan-created` 事件（payload 结构：`goal` + `steps`）
  - [x] SubTask 3.2: 在 §7.2 中说明 `ai:plan-updated` 事件（payload 结构：`step_id` + `status` + `result` + `steps` 快照）
  - [x] SubTask 3.3: 说明这两个事件与 `ai:tool-status` 的关系（规划工具同时产生 `tool_start`/`tool_result` 和独立的 plan 事件）

- [x] Task 4: 新增 TOOLS.md §9 规划工具设计说明 — 完整设计文档
  - [x] SubTask 4.1: 定义 `Plan` 和 `PlanStep` 数据结构（含 JSON 标签和字段说明）
  - [x] SubTask 4.2: 编写 `create_plan` 工具规范（职责、参数格式、校验规则、执行逻辑、返回文本格式）
  - [x] SubTask 4.3: 编写 `update_plan` 工具规范（职责、参数格式、执行逻辑、返回文本格式）
  - [x] SubTask 4.4: 编写 GenModelInputFunc 钩子集成说明（如何在每轮 LLM 调用前注入计划状态）
  - [x] SubTask 4.5: 编写前端事件消费说明（`ai:plan-created` 和 `ai:plan-updated` 的前端处理要点）

# Task Dependencies

- Task 1、Task 2、Task 3、Task 4 之间无依赖，可并行执行
- 所有 Task 均为 TOOLS.md 文档变更，不涉及代码实现
