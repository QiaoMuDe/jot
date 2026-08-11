# Agent 新增 get_stats 状态统计工具 Spec

## Why

Agent 工具集目前无法回答"我总共有多少笔记/写了多少篇/向量索引建了吗/AI 用了多少 token"这类**存量与状态概览**问题（recall_notes 是内容召回、manage_note list 要翻页猜总数）。新增只读状态工具 `get_stats`，一次调用返回应用数据概览与月度笔记统计，与现有工具零重叠。

## What Changes

- 新增 [internal/agent/tools/get_stats.go](internal/agent/tools/get_stats.go)：`getStatsTool`（依赖 `NoteService` + `VectorService` + `Context`），提供 `Info()` / `InvokableRun()` / `ActionText()`，按 `action` 分发：
  - `overview`（缺省）：`NoteService.GetStats()` 全量数据概览 + `VectorService.GetIndexStatus()` 向量索引状态
  - `month`：`NoteService.GetMonthCounts(year, month)` 某月每日笔记数（year/month 缺省当前年月）
- [internal/agent/registry.go](internal/agent/registry.go)：`buildTools` 追加注册行 `tools.WrapWithError("get_stats", tools.NewGetStats(p.deps.Note, p.deps.Vector, p.ctx), p.ctx)`
- [internal/agent/TOOLS.md](internal/agent/TOOLS.md)：§6 现有工具清单补一行 `get_stats`；§2/§3/§5/§8 无需改动（动作文案机制已就绪）
- 无前端改动（工具名英文直接展示，action_text 由 `ActionTextProvider` 机制自动下发）
- **无依赖新增**：`Deps` 已含 `Note`、`Vector`（app.go 两处构造已传参）

## Impact

- Affected specs: Agent 工具集（新增第 10 个工具）
- Affected code:
  - [internal/agent/tools/get_stats.go](internal/agent/tools/get_stats.go)（新增）
  - [internal/agent/registry.go](internal/agent/registry.go)（注册）
  - [internal/agent/TOOLS.md](internal/agent/TOOLS.md)（工具清单）
- 调用底层：`NoteService.GetStats` / `NoteService.GetMonthCounts` / `VectorService.GetIndexStatus`（均只读）

## ADDED Requirements

### Requirement: get_stats 工具（只读状态概览）

系统 SHALL 提供 `get_stats` 工具，仅执行只读统计，**不修改任何数据**；Desc 明确边界：查看具体笔记/待办/标签列表用对应 `manage_*` 工具，本工具回答"总量/概览/月度趋势"类问题。

#### Scenario: 默认概览（无 action 或 action=overview）

- **WHEN** 用户问"我现在有多少笔记/标签/待办/AI 用量/数据库多大"
- **THEN** 工具返回 `GetStats()` 的格式化概览（笔记总数/回收站/置顶/笔记本/标签/待办完成比/AI 会话与消息/总 token/平均响应与思考时长/DB 大小）+ `GetIndexStatus()` 向量索引状态（已量化笔记数/片段总数/占用字节），模型据此直接作答

#### Scenario: 月度统计（action=month）

- **WHEN** 用户问"这个月/某月写了多少篇笔记"
- **THEN** 工具以 year/month（缺省当前年月，month 范围 1-12 校验）调用 `GetMonthCounts`，返回"YYYY-MM 每日笔记数"列表

#### Scenario: 非法 action 或非法月份

- **WHEN** action 不在枚举（overview/month）内，或 month 不在 1-12
- **THEN** 返回错误（经 WrapWithError 回填模型），不崩溃

### Requirement: 动作文案

`get_stats` SHALL 实现 `ActionText(argumentsInJSON string) string`：overview（或缺省）→ `获取数据统计概览`；month → `获取月度笔记统计`；解析失败返回空串；其他 → `执行`。

#### Scenario: 状态条文案

- **WHEN** 模型调用 get_stats 且未传 action
- **THEN** 前端状态条显示"调用「get_stats」工具：获取数据统计概览"

## MODIFIED Requirements

（无既有需求被修改）

## REMOVED Requirements

（无既有需求被移除）
