# 操作系统子 Agent 封装 Spec

## Why

Agent 内置工具已达 20 个，叠加 MCP 后父 Agent 每轮都要为全部工具描述付 standing cost，工具越多选择准确率越低。文件/命令工具族（11 个）与 `run_command` 职责同域，适合整封装为一个「操作系统子 Agent」：父 Agent 只看到一个 `os_agent` 工具，文件域的内部决策与迭代隔离在子 Agent 上下文内，父层每轮省下 11 份工具描述。

## What Changes

- **后端：新增 `subagent.go` 子 Agent 逻辑文件**——所有子 Agent 逻辑与定义集中于此文件编写：自定义委托工具（非 eino 内置 `NewAgentTool`，以复用现有事件/记录/审批管道）、内层 Agent 构造器、内层系统提示词、事件转发。`os_agent` 委托工具的内层为独立 ChatModelAgent，工具白名单 = 现有 11 个文件/命令工具（read_file/write_file/edit_file/ls_dir/glob/grep_file/copy_file/move_file/delete_file/mkdir_dir/run_command），独立 `MaxIterations`（默认 20），内层指令约束"仅操作 ~/.jot/workspace、不调用非文件能力"。后续新增笔记子 Agent 等同类机制也在此文件扩展定义。
- **后端：父层注册移除 11 个工具**，仅注册 `os_agent`（可被 `ai_agent_tools_disabled` 禁用、非 PlanOnly）。旧禁用名单中的 11 个旧工具名不再匹配任何工具，静默忽略；前端下次保存设置时自然清理。
- **后端：内层事件转发**——子 Agent 内部每一步工具调用以现有 `ai:tool-status` 事件实时下发（复用 `emitToolStart`/`emitToolResult` 语义 + 独立内层 toolByName 映射），并写入同一 `toolRecords`（保证实时与回放一致）；内层流式正文/思考链**不转发**（避免污染主消息气泡），仅把最终摘要作为工具结果返回。审批（`ai:tool-approval`）与反问（`ai:ask-user`）继续复用会话级通道，前端审批面板零改动。
- **元数据与文档**：`tools/meta.go` 移除 11 条目、新增 `os_agent`（Label「操作系统任务（工作区文件读写与命令执行）」）；`tools/doc.go`、`TOOLS.md`、`EVENTS.md` 同步更新。
- **前端：工具列表收敛**——设置页「Agent 工具」管理器与 AI 助手工具栏「Agent 工具」下拉，其数据源 `GetAgentTools()` 自动变为单条 `os_agent`（内置组、可勾选禁用），无需改列表组装逻辑，但需排查硬编码旧工具名残留。
- **前端：子步骤实时渲染**——`os_agent` 调用行显示执行中动画（脉动点/齿轮）；内层步骤（read_file/write_file/run_command…）以缩进子步骤实时渲染（带左侧引导线、弱化色）；`os_agent` 完成时主行收起为摘要并关闭分组。实时路径与历史回放（buildToolRecords）共用同一渲染函数族，保证"所见即所存"。

## Impact

- Affected specs: add-agent-workspace-fs-shell（文件/命令工具域）、add-agent-tool-toggle（工具开关配置）、add-ai-agent-mode（工具装配）
- Affected code:
  - [registry.go](internal/agent/registry.go)（buildTools 移除 11 工具、装配 os_agent）
  - [agent.go](internal/agent/agent.go)（Run 中构造 os_agent：内层 Context/chatModel/迭代上限传入）
  - 新增 [subagent.go](internal/agent/subagent.go)（子 Agent 逻辑与定义文件：委托工具/内层 Agent 构造/事件转发）
  - [tools/meta.go](internal/agent/tools/meta.go)（清单替换）、[tools/doc.go](internal/agent/tools/doc.go)、[TOOLS.md](internal/agent/TOOLS.md)、[EVENTS.md](internal/agent/EVENTS.md)
  - [app.go](app.go)（无需改绑定，核对 GetAgentTools/DisabledTools 透传即可）
  - [main.js](frontend/src/main.js)（子步骤渲染逻辑、排查旧工具名残留）
  - [ai-chat.css](frontend/src/css/components/ai-chat.css)（子步骤/执行中动画样式）

## ADDED Requirements

### Requirement: os_agent 子 Agent 委托工具

后端 SHALL 提供 `os_agent` 工具：内部构建一个独立 ChatModelAgent（复用当前会话 ChatModel 客户端与指纹），工具白名单固定为 11 个文件/命令工具，`MaxIterations` 独立（默认 20），内层 `tools.Context` 复用父层 `Emit`/`Records` 并注入同一会话的 `Approver`/`AskWaiter`。`os_agent` 被禁用（`ai_agent_tools_disabled` 含其名）时不注册。

#### Scenario: 内层白名单与禁用过滤
- **WHEN** 用户启用 `os_agent` 发起 Agent 对话
- **THEN** 父层工具列表只含 `os_agent`（不含 11 个文件/命令工具），内层装配恰好 11 个文件/命令工具，不含 ask_user/plan/MCP 工具
- **AND** 用户禁用 `os_agent` 后，模型不可见任何文件/命令能力

#### Scenario: 内层指令约束
- **WHEN** 子 Agent 运行
- **THEN** 内层系统提示词明确"仅操作 ~/.jot/workspace 内路径；文件与命令之外的诉求（笔记/网络/提问）在最终回复说明，由主 Agent 处理；危险操作会请求用户确认，被拒绝时改用其他方式"

### Requirement: 内层事件实时转发与记录

子 Agent 内层每步工具调用（start/result/error/partial）SHALL 以现有 `ai:tool-status` 事件实时下发，并追加进父层 `toolRecords`（与 `os_agent` 的 start/result 记录同一切片、顺序相邻），保证实时渲染与历史回放一致。内层流式正文与思考链 SHALL 不转发。`os_agent` 工具结果返回内层最终摘要文本。

#### Scenario: 记录顺序与分组
- **WHEN** 一轮中父 Agent 调用一次 `os_agent`，内层依次执行 read_file、write_file
- **THEN** `toolRecords` 顺序为：`os_agent` tool_start → `read_file` start/result → `write_file` start/result → `os_agent` tool_result（内层记录位于两者之间，构成天然分组边界）
- **AND** 前端实时收到的事件顺序与回放渲染完全一致

#### Scenario: 审批继续生效
- **WHEN** 子 Agent 内层调用 `run_command` 命中黑名单且会话模式为 confirm_every/review
- **THEN** 前端照常弹出审批面板（工具名 run_command + 完整命令），批准后继续、拒绝后回填错误继续推理，行为与现状一致

### Requirement: 工具清单收敛

`BuiltinTools()` SHALL 移除 11 个文件/命令工具条目、新增单条 `os_agent`（普通组、可勾选）。`GetAgentTools()` 返回结果随之收敛；设置页与 AI 助手下拉自动只显示 `os_agent`。

#### Scenario: 列表只显示一个子 Agent
- **WHEN** 打开设置页 Agent 工具管理器或 AI 助手 Agent 工具下拉
- **THEN** 内置组显示单条 `os_agent`（名称 + 中文说明 + 勾选状态），不再出现 read_file/write_file 等 11 个条目
- **AND** 旧禁用名单残留（如 `["run_command"]`）不产生任何效果，勾选保存后从设置中清除

### Requirement: 前端子步骤渲染

前端 SHALL 在 `os_agent` 行开启时渲染执行中动画；其后的内层工具记录以缩进子步骤渲染（引导线、弱化色、行高压缩）；匹配的 `os_agent` tool_result 到达后主行收起为摘要并关闭子步骤分组。实时事件渲染与历史回放 SHALL 共用同一渲染函数族（遵守"所见即所存"约束），并按 `os_agent` 的 call_id 配对分组边界（同一轮多次调用亦正确嵌套）。

#### Scenario: 实时子步骤展示
- **WHEN** `os_agent` 调用开始
- **THEN** 主行显示「操作系统任务」+ 执行中动画（脉动点/齿轮），后续 read_file/write_file/run_command 记录以缩进子步骤实时出现
- **AND** 完成后主行变为摘要态，子步骤保持缩进展示或按设计折叠，动画停止

#### Scenario: 历史回放一致
- **WHEN** 打开历史会话（GetMessages 回放 toolRecords）
- **THEN** 子步骤的渲染结构与实时一致（名称/缩进/分组边界完全相同）

## MODIFIED Requirements

### Requirement: buildTools 装配（MODIFIED）
`buildTools` 从全量注册 20+ 工具改为：文件/命令 11 工具不再注册，改为注册 `os_agent`（同样受 `disabled` 过滤、非 PlanOnly）。`agent.Request.DisabledTools` 语义不变。旧禁用名单中的文件工具名（read_file 等）不匹配任何工具，静默忽略。

### Requirement: Agent 工具开关配置（MODIFIED）
设置页与 AI 助手下拉的「Agent 工具」清单由 20+N 收敛为（内置 N-10）+ MCP N + 常驻/Plan 不变。禁用粒度从"单个文件工具"提升为"整个操作系统子 Agent"。历史配置中的旧文件工具名不再影响装配。

## REMOVED Requirements

### Requirement: 父层直接注册文件/命令工具
**Reason**: 全部迁入 `os_agent` 子 Agent，父层不再直接暴露，消除每轮 standing tool 成本并降低选择错误率。
**Migration**: 功能由 `os_agent` 完全承接；设置页旧禁用条目静默失效，下次保存清理。
