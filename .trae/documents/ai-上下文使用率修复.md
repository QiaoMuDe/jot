# AI 助手上下文使用率修复 —— 直接读取最后一条用户消息的回填 Token

## Summary
当前「上下文使用率」指示器 `used` 远小于真实占用（真实 200+，指示器仅报 130）。根因是 `GetAIContextUsage` 只统计「摘要边界后」的 tail、且用低估的字符启发式 `EstimateTokens`，没复用消息上已经回填的真实 token。

修复方向（经讨论确认、最简）：**不加任何字段，直接读取会话最后一条 user 消息的 `Tokens` 作为 `used`**。该值在对话完成轮已被 `UpdateAIMessageTokens` 回填为「含完整请求」的 token —— 正式复用现有回填数据，不新增持久化字段。只改指示器，**不动摘要/截断逻辑**。

## 已核实的事实
- 每条用户消息落库初值 = `estimateTokens(content)`（只估算该条，[SaveAIMessage](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L3035-L3040)）。
- 对话完成轮后 `UpdateAIMessageTokens` 覆盖为「已回填」值：
  - **Agent 模式** = 真实完整 `result.PromptTokens`（eino 每轮真实 usage 累加，[agent.go#L874-L875](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/agent/agent.go#L874-L875)，[app.go#L2297](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2297)）—— 真实完整请求占用 ✓
  - **Chat 模式** = `estimateUserTokens(messages)+estimateTokens(systemMsg)`（[app.go#L2505](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2505)，user 侧口径；einocli 流式不回传真实 usage）
- 会话模型 [ai_session.go](file:///d:/资源池/下水道/Dev/本地项目/jot/internal/models/ai_session.go#L10-L20)：**无需加字段**。

## Proposed Changes

### 1. `GetAIContextUsage` 直接读最后一条 user 消息 Tokens（app.go）
重构 [GetAIContextUsage](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2920-L2955)：
1. 加载会话消息，找到**最后一条 `Role=="user"`** 的消息。
2. 取命中消息的 `Tokens` 作为 `used`（>0 即用）。
3. `percent = used/budget`（`budget=GetContextTokenBudget()`），`trigger` 不变。

### 2. tiktoken 兜底（仅 used 缺失/为 0 时，services + app.go）
- 引入 `github.com/tiktoken-go/tokenizer`（eino 官方推荐的真实分词库），在 `internal/services/ai_context.go` 新增 `CountTokens(text)`（默认 `o200k_base`，初始化缓存单实例），**不替换** `EstimateTokens`。
- 当会话没有 user 消息或最后一条 `Tokens==0`（极端/旧数据）时，用 tiktoken 按完整请求口径兜底：`CountTokens(system)` + `CountTokens(摘要块)` + Σ`CountTokens(tail)`，避免显示 0。

### 3. 不改摘要/截断、不改前端
- `truncateAIMessages` / 摘要压缩 / 落库逻辑**完全不动** → 对上下文摘要零影响。
- 前端 [updateContextUsage](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L590-L633) 只消费 `used/budget/percent/trigger`，无需改动。

## Assumptions & Decisions
- `used` 以**最后一条 user 消息的 `Tokens`（已回填值）**为准：Agent 完成轮=真实完整 PromptTokens；Chat 完成轮=user 侧口径（含 system+末条 user+本次 system）。
- **已知边界（确认接受）**：Chat 模式下回填值为 user 侧口径，不含中间历史/摘要，指示器在纯 Chat 长会话中会偏保守（偏低）；如需 Chat 也精确到完整窗口，需改走 tiktoken 完整组合——本次按「直接读回填值、最简单」处理，tiktoken 只做 0 值兜底。
- 各条用户消息逐条 `Tokens` 不累加（Agent 的 user Tokens 是整段 prompt，累加会重复计数）；只取最后一条。
- tiktoken 编码默认 `o200k_base`；如需按模型族精确切换，留作后续。

## Verification
1. `go get github.com/tiktoken-go/tokenizer`，`wails build`，更新 `build/bin/jot.exe`。
2. 打开已至少完成一轮的 Agent 会话：`used` = 最后一条 user 消息回填的真实 `PromptTokens`（贴近该轮完整请求），`percent` 合理上升，不再出现「200+ 只报 130」。
3. 构造极端场景（无 user 消息/值为 0）：确认走 tiktoken 兜底不为 0。
4. 回归：发送/重发/重新生成后指示器正确刷新；切换会话数值正确；摘要压缩触发与落库行为与改动前完全一致（确认未受影响）。