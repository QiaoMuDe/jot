# 注入 AI 当前模式描述信息

## 概述

在 AI 系统提示词中注入当前模式（Chat/Agent/Plan）的详细描述，包括模式特点、适用场景和行为指引，让 AI 具备自我认知能力。

## 当前状态分析

* `buildAIContextInstruction()` 是 Chat/Agent 两模式共用的基础上下文构建函数

* Chat 模式：`CallAIStream` 直接使用 `buildAIContextInstruction()` 的结果作为 system message

* Agent 模式：`CallAIAgentStream` 在 `buildAIContextInstruction()` 之后追加工具使用规范

* Plan 模式：走 Agent 路径，`sessCfg.Mode == "plan"` 时 `PlanMode: true`，但其系统提示词仍然在 `CallAIAgentStream` 中构造

* `sessCfg` 在 `CallAIAgentStream` 中第 2189 行才加载，晚于 `buildAIContextInstruction` 的调用

## 修改方案

### 1. 新增模式描述常量（app.go，靠近 baseSystemPrompt 定义处）

在 `baseSystemPrompt` 附近新增三个常量：`chatModeDescription`、`agentModeDescription`、`planModeDescription`

### 2. 调整 CallAIAgentStream 中的加载顺序

将 `sessCfg := a.aiService.LoadSessionConfig(sessionID)` 从第 2189 行移到 `buildAIContextInstruction` 调用之前（约第 2138 行），以便在构建 instruction 时就能拿到 mode。

### 3. Chat 模式注入

在 `CallAIStream` 中，`buildAIContextInstruction` 返回后，追加 `chatModeDescription` 到 systemMsg。

### 4. Agent/Plan 模式注入

在 `CallAIAgentStream` 中，在所有工具使用规范追加完成后，根据 `sessCfg.Mode` 追加 `agentModeDescription` 或 `planModeDescription`。

### 5. 关键代码变更

#### 文件: d:\峡谷\Dev\本地项目\jot\app.go

**a) 新增常量**（在 `baseSystemPrompt` 之后，约第 63 行）

```go
var chatModeDescription = "\n\n【当前模式 - Chat（对话模式）】\n" +
    "特点：纯文本对话，不调用任何工具。\n" +
    "适用场景：日常问答、头脑风暴、写作创作、翻译、代码审查、知识讨论。\n" +
    "行为指引：直接回答用户问题，保持对话自然流畅。如果用户请求涉及搜索笔记或操作文件，礼貌告知当前模式不支持，建议切换到 Agent 模式。"

var agentModeDescription = "\n\n【当前模式 - Agent（智能体模式）】\n" +
    "特点：可调用工具执行任务（搜索本地笔记、联网搜索、管理笔记/笔记本/标签、反问澄清等）。\n" +
    "适用场景：信息检索、笔记管理、数据查询、需要多步骤操作的任务。\n" +
    "行为指引：优先使用工具完成任务，严格遵循本地知识优先、写操作确认等规范。如果用户只是闲聊，也可以直接对话。"

var planModeDescription = "\n\n【当前模式 - Plan（计划模式）】\n" +
    "特点：先制定计划，再逐步执行，支持复杂任务拆解。\n" +
    "适用场景：需要多步骤执行的复杂任务（如批量修改笔记、跨模块操作）。\n" +
    "行为指引：回答前先分析任务复杂度，生成结构化执行计划，按计划逐步执行。每步完成后检查进度，全部完成后总结。"
```

**b) Chat 模式注入**（`CallAIStream`，第 2434 行附近）

```go
// 原：systemMsg := a.buildAIContextInstruction(...)
// 改：
systemMsg := a.buildAIContextInstruction(skillIds, roleplayNoteIDs, referencedNoteIDs, followUpRefContent, uploadedFiles) + chatModeDescription
```

**c) Agent 模式 - 调整 sessCfg 加载顺序**（第 2138 行附近，移到 instruction 构建之前）

```go
// 新增：在 instruction 构建之前读取会话配置
sessCfg := a.aiService.LoadSessionConfig(sessionID)

// instruction 构建...
var instruction strings.Builder
instruction.WriteString(a.buildAIContextInstruction(...))
// ... 工具使用规范保持不变 ...

// 在工具使用规范之后追加模式描述
if sessCfg.Mode == "plan" {
    instruction.WriteString(planModeDescription)
} else {
    instruction.WriteString(agentModeDescription)
}
```

**d) 删除原位置**的 `sessCfg := a.aiService.LoadSessionConfig(sessionID)`（第 2188-2189 行），因为已移到前面。

## 假设与决策

* Plan 模式的描述注入到 ReAct 循环的 Instruction 中，而非 planGenSystemPrompt，因为计划生成阶段已有独立的角色定义

* 模式描述放在所有工具规范之后（最末尾），确保不干扰更重要的行为约束

* 不用修改 `buildAIContextInstruction` 函数签名，注入全部在调用方完成

## 验证步骤

1. 编译项目确认无语法错误
2. 分别切换到 Chat / Agent / Plan 模式发送消息，观察 AI 回答是否体现出模式认知
3. 检查 Chat 模式下 AI 是否不会再回复"我可以帮你搜索笔记"之类的误导性表述

