# Agent 反向提问（ask_user 工具）Spec

## Why
Agent 在信息不足或需要用户在多个方案间决策时，目前只能"猜"或给出含糊回答。支持像 Trae 那样的反向提问选择交互（问题卡片 + 选项按钮 + 自定义输入），让 Agent 能结构化地向用户澄清，提升回答质量与交互体验。

## What Changes
- 新增 Agent 工具 `ask_user`：不执行业务，仅发射 `ai:ask-user` 事件（question + options）请求用户澄清，并返回问句文本回填模型。
- `agent.go`：ask_user 调用轮的正文（模型写出的问句）作为最终回答兜底（最终轮无输出时），保证落库内容完整、历史回放可读。
- `app.go`：组装 Agent Instruction 时注入 ask_user 使用约束（仅在信息不足时使用、调用后必须停止生成、用户回答后继续回答）。
- 前端 `ai-chat.js`：监听 `ai:ask-user` 事件，在 assistant 气泡内渲染问题卡片（选项按钮 + 自定义输入）；点击选项/提交自定义输入 → 以新 user 消息发送 → 走既有流式流程续流。
- 前端 `ai-chat.css`：问题卡片样式（主题自适应，复用 `--bg`/`--accent` 变量）。
- 文档更新：`internal/agent/doc.go`、`internal/agent/tools/doc.go` 工具清单补 ask_user；`AGENTS.md` 记忆点追加；`internal/agent/TOOLS.md` 若有工具枚举则补一行。

### 交互时序（新消息发送方案）
1. 用户发消息 → Agent 判断信息不足 → 输出问句（assistant 消息正文）+ 调用 `ask_user` 工具。
2. 工具发射 `ai:ask-user`（{question, options}）→ 前端渲染问题卡片；工具返回问句文本，Instruction 约束模型立即停止生成。
3. 用户点击选项按钮或输入自定义答案 → 以新 user 消息发送 → 新一轮 Agent 流正常执行，模型结合上文继续回答。
4. 历史回放：assistant 消息正文即问句（含兜底逻辑），无按钮，退化为普通文本；工具调用链折叠照常展示。

## Impact
- Affected specs: agent 工具链路（registry.go 装配）、Agent Instruction 组装（app.go CallAIAgentStream）、前端 AI 聊天流事件机制（ai-chat.js startStreaming）
- Affected code:
  - 新增 `internal/agent/tools/ask_user.go`
  - `internal/agent/registry.go`（注册 ask_user）
  - `internal/agent/agent.go`（问句兜底，约 5-8 行）
  - `app.go`（Instruction 追加约束，约 6-10 行）
  - `frontend/src/js/ai-chat.js`（事件监听 + 卡片渲染 + 选择发送）
  - `frontend/src/css/components/ai-chat.css`（卡片样式）
  - `internal/agent/doc.go` / `internal/agent/tools/doc.go` / `internal/agent/TOOLS.md` / `AGENTS.md`（文档）

## ADDED Requirements

### Requirement: ask_user 工具
系统 SHALL 提供 `ask_user` 工具，允许 Agent 向用户发起结构化澄清提问。

- **Schema**：`question`（string，必填）、`options`（array of string，可选，2-6 项）、`reason`（string，可选，说明提问原因）
- **执行行为**：不执行业务操作；将 `{question, options}` JSON 序列化后通过 `ai:ask-user` 事件发射（Agent 模式单参数形态，与 `ai:tool-status` 一致）；返回问句文本给模型
- **ActionText**：`向用户提问：{question}`（question 截断至 30 字符）

#### Scenario: Agent 信息不足时发起提问
- **WHEN** Agent 判断需要用户澄清（如"想用哪个搜索源""选哪个方案"）
- **THEN** Agent 输出问句正文并调用 ask_user；前端收到 `ai:ask-user` 事件渲染问题卡片；Instruction 约束 Agent 调用后停止生成

### Requirement: 问题卡片与选择发送
系统 SHALL 在 AI 聊天界面渲染问题卡片，并支持用户选择或自定义输入后以新消息续流。

- 卡片展示问句标题 + 选项按钮列表 + 自定义输入框与提交按钮
- 点击选项按钮 / 提交自定义输入 → 构造新 user 消息（保存到 DB → 渲染用户气泡 → 触发 `startStreaming` 正常续流）
- 卡片保留在当前 assistant 气泡中，不随流结束移除

#### Scenario: 用户点击选项续流
- **WHEN** 用户点击卡片上的某个选项按钮
- **THEN** 该选项内容作为新 user 消息发送，新一轮 Agent 流开始，模型结合上文（问句 + 用户选择）继续回答

#### Scenario: 用户自定义输入答案
- **WHEN** 用户在卡片输入框输入自定义答案并提交
- **THEN** 输入内容作为新 user 消息发送，流程与点击选项一致

### Requirement: 历史回放退化
系统 SHALL 保证历史消息回放时 ask_user 交互可读且不可再交互。

- assistant 消息落库正文为问句（最终轮无输出时以 ask_user 调用轮的正文兜底）
- 回放渲染普通文本，不渲染按钮/输入框

#### Scenario: 切换会话回看
- **WHEN** 用户切换到包含 ask_user 交互的会话
- **THEN** 看到 assistant 问句文本与工具调用链折叠（"向用户提问：xxx"），无交互控件

### Requirement: 使用边界约束
系统 SHALL 在 Agent Instruction 中约束 ask_user 的合理使用。

- 仅在信息不足或需用户决策时使用；一次仅提一个问题；严禁无关澄清
- 调用 ask_user 后必须停止生成，等待用户回答
- 用户回答后继续正常回答流程

## MODIFIED Requirements
无（新增能力，不改动既有工具行为；仅前端事件监听列表新增 `ai:ask-user` 清理项）

## REMOVED Requirements
无
