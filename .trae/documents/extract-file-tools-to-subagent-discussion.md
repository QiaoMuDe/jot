# 文件工具抽离为子 Agent 方案讨论记录

> 日期：2026-09-17
> 状态：讨论已收敛，尚未动代码，明天继续出具体改造方案

## 背景

- AI 助手模块内置工具已达 27 个，其中文件/工作目录类占 11 个：`read_file` / `write_file` / `edit_file` / `ls_dir` / `glob` / `grep_file` / `copy_file` / `move_file` / `delete_file` / `mkdir_dir` / `run_command`。
- 这 11 个工具每个都带长 Desc 与多参数字段，全部挂在主 Agent 上，提示词与每次 LLM 调用的 schema 载荷都很重，担心太占上下文。

## 候选方案

### 方案 A：抽离为子 Agent（用户倾向，最终选定）

利用框架已支持的 `adk.NewAgentTool` 能力（已有 playground/agent-subagent-demo 用真实 LLM 验证），把 11 个文件工具装进一个子 Agent，主 Agent 只暴露 1 个子 Agent 工具。文件操作作为任务发布给子 Agent 执行。

### 方案 B：合并为多动作工具（曾提议，被否决）

把 11 个文件工具合并成 2~3 个带 action 参数的大工具（类似 manage_note 先例），减少主 Agent 的工具数量。

- **否决理由（用户）**：合并后参数太多。项目已有教训——`manage_note` 约 20 个参数就增加了模型参数选择错误率；合并会让单工具 schema 膨胀到不可控，损害模型调用准确率。

## 选定动机（用户明确的两点）

1. **隔离工具**：11 个工具各自保持小而精准的 schema，内部模型选参准确率不受影响；主 Agent 只见 1 个工具，schema 载荷骤降，避免在 27 个工具里模糊选择。
2. **隔离上下文**：文件原始内容只存在于子 Agent 局部上下文，stateless 每轮重建 = 用完即弃；主上下文只收到蒸馏后的摘要，不被 read_file 等的大段原文污染。

## 结论与权衡

- 方向认可：双 LLM 往返（主 Agent → 子 Agent → 文件工具 → 子 Agent → 主 Agent）是隔离的代价，读大文件时省下的上下文远多于多花的往返，净收益为正。
- 子 Agent 采用 **stateless 每轮重建**（与主 Agent 一致在 `Run()` 内构建）：原始内容不进主上下文，也无需为子 Agent 做压缩/摘要管理。
- 主 Agent 系统提示词里的文件工具使用规范一并移走，指令本身也瘦身。

## 已确认的实现要点（明天改造时落实）

1. **事件转发**：必须开启主 Agent 的 `EmitInternalEvents`，否则子 Agent 内部文件工具事件不转发到顶层，前端工具状态条（`ai:tool-status`）、`Result.ToolCalls` 落库、Plan 模式"工具执行后自动推进度"钩子都会失效。开启后主循环会看到"子 Agent 工具调用 + 内部文件工具调用"两层记录，需决定是否过滤外层。
2. **审批链路保活**：子 Agent 工具复用每轮注入的同一 `tools.Context`（`Approver` = agentSession），文件工具内部的 `RequestApproval → ai:tool-approval → ApproveToolCall` 通道不变，审批 UX 不受影响。
3. **结果契约（待细化）**：主 Agent 派发任务时按需二选一——
   - 摘要模式（默认）：返回"做了什么 + 关键结果"，适合整理/改动类任务；
   - 原样返回模式：任务显式要求"原样返回文件内容"，适合主 Agent 需要基于精确内容继续推理的场景（改代码、定位问题）。
   通过子 Agent 指令动态注入实现，不改造工具本身。
4. **设置页/工具清单迁移**：`ai_agent_tools_disabled` 中 11 个文件工具名将不再单独可禁，退化为只开关"文件操作子 Agent"一个工具；存量禁用配置中的旧名会静默失效（JSON 残留无匹配）。前端 meta.go 工具清单、四分组逻辑（内置/MCP/Plan/常驻）需同步重构。这是用户可见改动面最大的地方。
5. **MaxIterations 透传**：子 Agent 需独立迭代上限；`skill_deep_research` 把主 Agent 提到 200 的预算需传导给子 Agent。

## 下一步

- 基于 playground/agent-subagent-demo 出具体改造方案（子 Agent 怎么建、事件怎么转发、设置迁移怎么收），先审方案再动代码。
