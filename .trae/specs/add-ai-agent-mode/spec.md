# AI 对话新增 Agent 模式 Spec

## Why
现有 AI 对话（`CallAIStream`）是"前端开关 + 确定性流水线"：搜索、卡片召回由开关强制触发后注入 system message，模型没有决策权，也无法执行后续操作。基于已通过的 Eino 技术预研（`agent-demo`），新增一个 **Agent 模式**：由模型自主决定是否调用工具（联网搜索、卡片召回），为后续"创建笔记、保存笔记"等工具化操作打基础，且完全不影响现有问答模式。

## What Changes
- **会话配置**：`AISessionConfig` / `SessionConfig` 新增 `AgentEnabled bool` 字段（默认 false = 问答模式），随会话配置保存/读取
- **后端新增**：`CallAIAgentStream` 绑定方法（Eino 链路），与现有 `CallAIStream` 完全并行，现有问答模式一字不动
- **独立模块（架构约束）**：Agent 全部逻辑置于 `internal/agent` 独立模块，以 `AgentService` 形式对外提供 `Run(ctx, Request, Emit)` 接口；`app.go` 仅负责**依赖注入**（构造 `AgentService` 并持有）与**薄封装调用**（`CallAIAgentStream` 只做参数收集、结果落库），不包含任何 Eino/Agent 实现细节。未来新增功能（创建/保存笔记等）均在 `internal/agent` 内扩展，不改动 app.go 结构
- **前端**：AI 对话操作栏新增"问答 / Agent"模式切换（会话级、即时生效）；Agent 模式下隐藏深度思考、联网搜索、卡片召回三个开关（其能力改为工具注入，由模型自主决定是否调用）
- **工具范围（本期）**：仅读工具——`web_search`（Tavily/知乎多源）、`recall_notes`（向量召回）；写操作（创建/保存笔记）不在本期
- **不新增写操作、不引入反思循环图编排**，本期仅实现 ReAct 循环（`adk.NewChatModelAgent` 内置）

## Impact
- Affected specs: `add-ai-assistant`（对话主链路）、`add-multi-web-search-sources`、`add-card-recall`、`add-eino-agent-demo`（预研）
- Affected code:
  - `internal/agent/`（**新增独立模块**：AgentService、工具注册、事件回调、Eino 组装）
  - `internal/models/ai_session_config.go`（新增字段）
  - `internal/services/ai_service.go`（SessionConfig 结构、配置读写）
  - `app.go`（依赖注入 `AgentSvc` 字段 + 薄封装 `CallAIAgentStream` 绑定）
  - `internal/database/models.go`（若需登记迁移，跟随现有 AutoMigrate 机制）
  - 前端 `frontend/index.html`、`frontend/src/main.js`、相关 CSS

## ADDED Requirements

### Requirement: Agent 模式会话级开关
系统 SHALL 在 AI 会话配置中保存 `agent_enabled`（默认 false），并在前端 AI 对话操作栏提供"问答 / Agent"模式切换控件，切换即时生效且随会话持久化；切换不丢失当前会话消息。

#### Scenario: 用户切换为 Agent 模式
- **WHEN** 用户在 AI 对话操作栏切换到"Agent"
- **THEN** 会话配置保存 `agent_enabled=true`，操作栏隐藏深度思考/联网搜索/卡片召回三个开关（技能、引用笔记、上传文件等保留），后续提问走 Agent 链路

#### Scenario: 用户切回问答模式
- **WHEN** 用户在 Agent 模式下切回"问答"
- **THEN** 三个开关恢复显示，后续提问走现有 `CallAIStream` 链路，行为与改造前完全一致

### Requirement: Agent 模式后端链路（CallAIAgentStream）
系统 SHALL 提供 `CallAIAgentStream` 绑定方法，使用 Eino（`github.com/cloudwego/eino` 的 `adk` 子包）实现 ReAct 循环：模型自主决定是否调用工具、多轮迭代直至给出回答；复用现有 AI 配置（BaseURL/APIKey/Model）与历史消息滑动窗口。

#### Scenario: 普通问题（无需工具）
- **WHEN** 用户在 Agent 模式下提问无需联网/召回即可回答的问题
- **THEN** 模型直接流式回答，不触发任何工具调用

#### Scenario: 需要联网的问题
- **WHEN** 用户提问需要最新信息（如"今天天气"）
- **THEN** 模型自主调用 `web_search` 工具（模型自选搜索词与来源），拿到结果后给出带来源标注的回答，前端展示工具调用过程

#### Scenario: 需要笔记召回的问题
- **WHEN** 用户提问涉及历史笔记内容
- **THEN** 模型自主调用 `recall_notes` 工具（沿用会话配置的召回笔记本过滤），基于召回结果回答

#### Scenario: 流式输出与取消
- **WHEN** Agent 生成回答过程中用户点击停止
- **THEN** 流式输出终止，取消机制与现有问答模式一致（context cancel），不残留卡死的工具调用

### Requirement: 工具调用过程事件展示
系统 SHALL 在 Agent 模式下通过新增事件（如 `ai:tool-status`）向前端推送工具调用名称/参数/结果，前端以可读形式展示（可复用现有搜索状态展示样式）；最终回答仍走现有 `ai:stream-chunk` / `ai:stream-done` 事件链路。

#### Scenario: Agent 调用工具
- **WHEN** 模型发起工具调用且工具返回结果
- **THEN** 前端展示"正在使用 xx 工具…"及结果摘要，回答完成后事件正常结束

### Requirement: 会话消息与工具记录落库
Agent 模式的最终回答 SHALL 以 assistant 消息写入 `ai_messages`（与现有链路一致）；工具调用记录序列化后写入消息的搜索来源字段（`search_sources`），支持回放展示。

#### Scenario: 一轮 Agent 回答完成
- **WHEN** Agent 回答完成并保存消息
- **THEN** `ai_messages` 新增该 assistant 消息，`search_sources` 记录本轮工具调用摘要，会话摘要/历史加载正常

### Requirement: 模型不支持工具调用时的降级
Agent 模式发起前 SHALL 检测模型是否支持 function calling；不支持的模型（如部分 Ollama 小模型）提示用户切换到问答模式，不产生异常中断。

#### Scenario: 模型不支持工具
- **WHEN** 用户使用不支持 function calling 的模型并处于 Agent 模式提问
- **THEN** 前端收到友好中文错误提示，建议切换问答模式，链路安全结束

## MODIFIED Requirements

### Requirement: AI 会话配置读写
`GetSessionConfig` / `SaveSessionConfig` 等现有接口 SHALL 透传新增的 `agent_enabled` 字段，前端切换后刷新配置可读到最新值；不改变既有字段语义。

## REMOVED Requirements
无。本期不删除任何既有功能。
