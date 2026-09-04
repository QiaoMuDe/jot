# AlwaysOn 工具机制 —— 计划

## Summary

为 Agent 内置工具引入"**AlwaysOn（常驻、不可禁用）**"标记机制，先把 `manage_memory` 设为 AlwaysOn，保证长期记忆的写回能力在任何会话/任何前端配置下都保持开启，不被 `ai_agent_tools_disabled` 关闭。该机制复用现有 `PlanOnly` 的"标记 + 前后端特判"模式，后续其他核心工具可复用。

## Current State Analysis

现有工具可用性只有两种状态，见 [meta.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/tools/meta.go#L7-L11)：

```
tools.ToolMeta{ Name, Label, PlanOnly }        // 单一事实来源 (BuiltinTools)
agent.ToolMeta{ Name, Label, Enabled, PlanOnly } // 前端展示契约 (GetAgentTools 返回)
```

- **禁用链路**：全局 setting `ai_agent_tools_disabled`（JSON 数组）→ [app.go](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2217-L2224) 读为 `disabledTools` → `agent.Request.DisabledTools` → [agent.go](file:///d:/峡谷/Dev/本地项目/jot/internal/agent/agent.go#L526-L530) `buildTools` 命中即**不注册，模型不可见不可调用**。
- **前端设置页**：[GetAgentTools](file:///d:/峡谷/Dev/本地项目/jot/app.go#L2606-L2643) 返回内置 + 已预热 MCP 工具，前端 [createAgentToolRow](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9966-L10034) 渲染，`PlanOnly` 工具 `checkbox.disabled=true` + "仅 Plan 模式可用"提示。
- **结论**：目前无"内置/不可禁用"标记；`manage_memory` 与普通工具一样可被禁用，导致"禁用它后注入仍生效、模型却调不动"的割裂。

## Proposed Changes

### 1. tools/meta.go —— 定义 AlwaysOn 并启用
- `ToolMeta` 增加字段 `AlwaysOn bool // 常驻/不可禁用（注册强不可从禁用集合剔除；前端不可勾选）`
- 把 `manage_memory` 条目改为 `{Name: "manage_memory", Label: "管理长期记忆（保存/更新/删除/列出）", AlwaysOn: true}`

### 2. internal/agent/types.go —— 前端展示契约透传
- `agent.ToolMeta` 增加 `AlwaysOn bool`（json/注释），供前端读取该标记。

### 3. app.go GetAgentTools —— 透传 AlwaysOn
- L2620 内置工具构造处补 `AlwaysOn: m.AlwaysOn`；MCP 工具不设（恒 false）。

### 4. app.go Agent 调用处 —— 后端豁免（双防）
- 在 L2217-L2224 读取 `disabledTools` 之后、组装 `agent.Request` 之前：
  - 遍历 `tools.BuiltinTools()`，收集 `AlwaysOn=true` 的工具名集合 `exempt`；
  - 从 `disabledTools` 中剔除 `exempt` 内名称；
  - 打一行日志（如记录被强制保留的工具名），供排查。
- 原因：即便前端配置（`ai_agent_tools_disabled`）残留 `manage_memory`，后端也强制不生效，确保 AlwaysOn 优先。

> 不改 `agent.go` 的 `buildTools` 禁用过滤逻辑，豁免集中在 app.go 一处，改动最小、语义清晰。

### 5. 前端 createAgentToolRow —— AlwaysOn 置灰不可勾
- 在现有 `PlanOnly` 分支旁新增 `tool.AlwaysOn` 分支：
  - `checkbox.disabled = true`、加入样式 class；
  - 追加 hint span，文案"此工具始终可用，不可禁用"；
  - 点击行触发抖动 + 通知"此工具为常驻能力，不可禁用"。

### 6. 前端全选/全不选/统计 —— 排除不可禁用工具
- 三处用 `!tool.PlanOnly` 收集"可控工具"的地方同步排除 `AlwaysOn`：
  - [L9809](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9809) enabledCount 统计
  - [L9823](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9823) `controllableTools`
  - [L9853](file:///d:/峡谷/Dev/本地项目/jot/frontend/src/main.js#L9853) 全选/全不选 forEach
- 保证"全不选"不会尝试禁用到 AlwaysOn 工具，全选框状态不受其影响。

## Assumptions & Decisions

- **AlwaysOn 仅适用于内置工具**，不适用于 MCP 工具（MCP 由服务器管理与 disabled 过滤）。
- **AlwaysOn 优先级高于用户禁用配置**：前端禁掉入口 + 后端剔除残留，双防。
- 命名统一用 `AlwaysOn`，前后端字段一致，与 `PlanOnly` 平行。
- 本次只把 `manage_memory` 设为 AlwaysOn；是否冷动其他工具由后续决定，不在此计划。
- 记忆注入（【长期记忆】段）本就不经禁用链路，本次不改动。

## Verification

- `go build ./...` 通过
- `go vet ./internal/agent/...` 通过
- 前端 `npm run build` 通过（若项目有此脚本）
- 手动验证：
  - 设置页工具列表中 `manage_memory` 复选框置灰 + 显示"此工具始终可用，不可禁用"，点击有抖动提示；
  - 其他工具（如 `manage_todo`）仍可正常启停；
  - 即使把 `ai_agent_tools_disabled` 手动写入 `["manage_memory"]`，后端日志显示其被强制保留，Agent 仍能调用 `manage_memory`；
  - 全选/全不选不影响 `manage_memory` 的启用态。