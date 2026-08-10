# Agent 模式耗时/token 统计落库修复

## Summary

Agent 模式的 `elapsedThinking` / `elapsedTotal` 已通过 `ai:stream-done` 回传前端（实时显示正常），但**未写入 `ai_messages` 表**，切换会话后历史消息 `thinking_elapsed` / `total_elapsed` 为 0，耗时徽标消失。本计划将落库逻辑对齐 Chat（问答）模式。

## Current State Analysis

- **Chat 模式**（[CallAIStream L2348-2368](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2348-L2368)）：`assistantMsg` 构造时写入 `ThinkingElapsed`（`thinkingEnabled` 才存，否则 0）与 `TotalElapsed: elapsedTotal`，经 `SaveAIMessage` 落库
- **Agent 模式**（[CallAIAgentStream L2606-2616](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2606-L2616)）：`assistantMsg` 只有 `Content/Tokens/SearchSources/RecallCards/ToolCalls/ReasoningContent`，**缺少 `ThinkingElapsed` / `TotalElapsed`**
- 模型层字段已就绪：`models.AIMessage`（[ai_message.go L14-15](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/ai_message.go#L14-L15)）已有 `ThinkingElapsed` / `TotalElapsed` 列（AutoMigrate 已建）
- 前端历史加载已就绪：`ai-chat.js` L1572/1605/1656/1672 已读取 `thinking_elapsed` / `total_elapsed` 并传给 `addMessage` 渲染，`createMsgActions`（L3629-3637）按 `elapsedTotal > 0` 显示"⏱ N 秒"
- 遗留问题：Agent 模式下 `elapsedThinking` / `elapsedTotal` 的变量定义（L2641-2646）位于 `assistantMsg` 构造（L2606-2616）**之后**，需前移

## Proposed Changes

### 文件：`app.go` 的 `CallAIAgentStream`（仅此一处）

1. **elapsed 计算前移**：把 L2641-2646 的 `elapsedThinking` / `elapsedTotal` 计算移到 `userTokens := estimateUserTokens(messages)`（L2600）之前
2. **assistantMsg 补字段**（对齐 Chat 模式 L2358-2364）：
   - `ThinkingElapsed: elapsedThinking`（思考关闭时 `thinkingStart` 恒为零 → 自动为 0，与 Chat 模式语义一致）
   - `TotalElapsed: elapsedTotal`
3. **stream-done 引用既有变量**：删除 L2641-2646 的重复定义，直接使用已前移的变量

不需要改：模型层、services 层转换、前端（全部已就绪）。

## Assumptions & Decisions

- 思考耗时的起算点沿用现有实现（首个 `ai:stream-thinking` 分片 → 流结束），落库值 = 实时回传值，保证历史与实时显示一致
- `thinkingEnabled=false` 时 `elapsedThinking` 恒为 0（后端未收到思考分片），与 Chat 模式"关闭深度思考不存思考耗时"的行为一致

## Verification

1. `go build ./...`、`go vet ./...` 通过
2. `wails dev` 冒烟：
   - Agent 模式开启深度思考提问 → 回答后消息显示"⏱ N 秒"；**切换会话后仍显示**（历史加载 `total_elapsed` 非 0）
   - Agent 模式关闭深度思考 → 历史消息 `thinking_elapsed` 为 0、`total_elapsed` 正常
   - 重开会话后思考折叠区仍显示"已思考 N 秒"（`thinking_elapsed` 落库生效）
