# Agent 模式系统提示词调整：本地知识优先 + 联网兜底

## Summary

调整 Agent 模式的系统提示词（Instruction 组装），新增「本地知识优先」工具选择策略，使 Agent 在执行时：
1. 优先调用 `recall_notes` 检索本地笔记；
2. 本地知识不足时再调用 `web_search` 联网补充；
3. 用户明确要求联网查询时直接 `web_search`，不先检索本地。

仅改动 Agent 模式（`CallAIAgentStream`），不影响问答模式（`CallAIStream`）与共享的 `baseSystemPrompt`。

## Current State Analysis

- Agent 模式系统提示词在 [app.go](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2457-L2552) 的 `CallAIAgentStream` 中组装：`baseSystemPrompt`（无技能时）/ `baseNormsBoundaries`（有技能时）+ 角色扮演 + 笔记引用 + 追问引用 + 上传文件 + 技能提示词 + 「【工具使用规范 - ask_user 反向提问】」。
- `baseSystemPrompt = baseIdentity + baseNormsBoundaries`（[app.go L49-L89](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L49-L89)），其中 `reasoningFramework` 第二步已写「信息优先级：本地笔记 > 联网搜索结果 > 模型自身知识」，但这是**被动整合**已提供参考内容的规则，不指导模型**主动选择工具**的调用顺序。
- `baseSystemPrompt` 同时被问答模式 `CallAIStream` 使用（[app.go L1994](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L1994)），因此**不能**在其中追加工具策略，否则会污染无工具的问答模式。
- Agent 内置工具（[registry.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/registry.go)）：`refine_search_query`、`web_search`、`recall_notes`、`get_current_time`、`manage_*`、`get_stats`、`ask_user`。
  - [recall_notes.go L47](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/recall_notes.go#L44-L56) 工具描述已含「优先于联网搜索」，但其返回空/失败时（如量化连接未配置）模型需要知道应转联网。
  - [web_search.go L68](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/web_search.go#L65-L83) 描述仅提到"新闻/天气/最新动态"类问题，未提"本地知识不足时补充"。
- Agent 是 ReAct 循环（[agent.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L235-L251)），工具调用顺序由模型自主决策；系统提示词是影响其行为的主要手段（纯提示词层调整，无需改循环逻辑）。

## Proposed Changes

### 1. app.go — 新增「本地知识优先」工具选择策略段（核心改动）

在 `CallAIAgentStream` 的 Instruction 组装末尾、现有的「【工具使用规范 - ask_user 反向提问】」段**之前**，追加一段 Agent 模式专用的工具选择策略：

```go
// Agent 模式专用约束：本地知识优先 + 联网兜底的工具选择策略（仅 Agent 模式注入）
instruction.WriteString("\n\n【工具使用规范 - 本地知识优先与联网搜索】\n" +
    "1. 优先检索本地知识：除非问题明确需要实时/外部信息，否则回答前应先调用 recall_notes 工具检索用户本地笔记；本地笔记是用户自己的知识库，可信度最高，应作为首要信息源。\n" +
    "2. 本地知识不足再联网：当 recall_notes 未检索到相关内容、或返回内容不足以支撑完整回答时，再调用 web_search 联网搜索补充信息；不要在未检索本地笔记的情况下直接联网。\n" +
    "3. 明确联网需求直接联网：当用户明确要求“联网搜索/查询网上资料/查最新资讯”等，或问题明显属于需要实时信息（如新闻、天气、股价、最新政策、外部教程）时，直接调用 web_search，无需先检索本地笔记。\n" +
    "4. 信息整合优先级：本地笔记信息优先于联网搜索结果，联网搜索结果优先于模型自身知识；引用时标注来源（如“你的笔记《XXX》中记录了……”或“根据搜索结果显示……”）；不同来源信息矛盾时如实说明差异。\n" +
    "5. 失败降级：recall_notes 因量化连接未配置等原因失败时，直接转 web_search 联网搜索；recall_notes 已返回充足信息时，不再重复联网。\n")
```

位置：紧跟技能提示词注入之后、`ask_user` 段之前（两段同属「工具使用规范」系列，连在一起更易被模型注意到）。

**为什么**：Agent 模式每次会话都组装此段，纯提示词层引导模型工具调用顺序，不动 ReAct 循环与工具装配；问答模式完全不注入，无副作用。

### 2. web_search 工具描述微调（辅助强化）

在 [web_search.go L68](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/tools/web_search.go#L65-L83) 的 `web_search` 工具 Desc 中追加一句，明确其"补充角色"：

- 现有：`搜索互联网获取实时信息。当用户询问新闻、天气、最新动态、人物、事件等需要联网获取实时或外部信息的问题时调用。…`
- 改为：`搜索互联网获取实时信息。当用户询问新闻、天气、最新动态、人物、事件等需要联网获取实时或外部信息的问题时调用；当本地笔记检索（recall_notes）未返回足够内容时，也应调用本工具补充信息。…`（其余不变）

**为什么**：与提示词段呼应，双重强化"本地不足→联网兜底"的策略，且改动仅一行、零风险。

## Assumptions & Decisions

- **纯提示词层调整**：用户要求"把系统提示词往这个方向调整"，故不改动 Agent 循环/工具装配逻辑（不强制预召回、不改变工具注册顺序）。模型仍自主决策工具调用，提示词引导为主。
- **仅影响 Agent 模式**：不改 `baseSystemPrompt` / `baseNormsBoundaries` / `reasoningFramework`，避免污染问答模式（其无工具）。
- **不新增设置项**：不引入"本地优先/联网优先"开关，保持最小改动。若后续需要可配置，另行扩展。
- 现有 `recall_notes` 描述已含"优先于联网搜索"，无需改动。
- 纯后端 Go 改动，无需 `npm run build`；需重新编译 Wails 应用（`wails build`）后生效。

## Verification

1. `go build ./...` 编译通过（或直接 `wails build`）。
2. 运行应用进入 Agent 模式，验证三类场景：
   - 问"我的笔记里关于 XX 的内容" → 应首先调用 `recall_notes` 并基于本地笔记回答，不联网；
   - 问本地无记录的问题（如"XX 最新消息"）→ 先 `recall_notes`（返回空/失败）后转 `web_search`，回答引用搜索结果；
   - 明确说"帮我联网搜索 XX" → 直接 `web_search`，不先检索本地。
3. 确认问答模式（非 Agent）行为无变化（基础提示词未动）。
