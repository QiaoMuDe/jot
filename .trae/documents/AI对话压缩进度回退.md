# AI 上下文使用率回退 —— 只保留「历史对话触发压缩进度」

## Summary
把之前的「上下文使用率 / 摘要触发距离」两个进度条整体回退收敛为**一条：历史对话离触发压缩的进度**。
- 后端 `GetAIContextUsage.used` 由「最后一条 user 消息回填的完整 token」（会显示 >100%）**回退为「摘要边界后 tail 的估算 token」**——即触发压缩真正看的那段历史。
- 前端不再展示「上下文总占用 / 超 100%」，指示器语义 = 历史对话压缩进度（tail/预算，达 80% 触发线即 warning）。
- 顺手统一各处文案，避免再出现「123% 却不压」这类口径误导。

只改后端口径 + 文案，**不动** `truncateAIMessages` / 摘要触发判断逻辑（触发行为本身不变）。

## 当前状态（已核查）
- 双进度条改动**已被回退**：`frontend` 的摘要条元素、`.ai-summary-usage` 样式、`summaryUsageEl` 相关 JS、`ContextUsage.Summary*` 字段、`aiSummaryUsageTipDetail` 均已不存在。
- 前端 `updateContextUsage`（[ai-chat.js#L590-L633](file:///d:/资源池/下水道/Dev/本地项目/jot/frontend/src/js/ai-chat.js#L590-L633)）= 单环形，`percent=used/budget`，`is-warning` 阈值 ≥ `trigger`(80%)——渲染逻辑已契合「触发压缩进度」。✅ 无需改渲染。
- 后端 [ContextUsage](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2909-L2943) 仍是单字段（Used/Budget/Percent/Trigger），结构符合；但 `used` 当前取「最后一条 user 消息回填 token」→ 语义是"完整请求占用"，会超 100%，**需回退为 tail**。
- 文案：指示器 `aria-label="上下文使用率"`（index.html L1135）、tooltip「达到上限 80% 自动压缩」（index.html L2728）、`updateContextUsage` 注释「上下文使用率」均为旧口径，需统一。

## Proposed Changes

### 1. `app.go` — `GetAIContextUsage` 回退 used 为 tail 口径（L2917-L2943）
- 把 `used` 从「倒数第一条 user 消息回填 token」**改回**「摘要边界（`Session.SummaryUpToMsgID`）之后、按预算 `SelectTailByTokenBudget(rest, budget)` 选取的 tail 的 `EstimateTokens` 之和」，即第一版逻辑。
  - 关键：与 [truncateAIMessages](file:///d:/资源池/下水道/Dev/本地项目/jot/app.go#L2010-L2029) 同源的 tail 口径，`percent = used/budget` 因此恒反映"当前历史在预算中占多少"，达到 `Trigger`(0.8) 时即将压缩，不再超 100%（歧义消除）。
- 更新结构体与函数注释，明确：
  ```
  used     = 摘要边界后 tail 估算 token（历史对话触发压缩的参考规模）
  percent  = used / budget，达 1 即预算满、达 Trigger(80%) 接近压缩
  Trigger  = 触发比例（ai_context_summary_trigger_ratio，默认 0.8）
  ```
- **不改**其他字段 / 结构，`Summary*` 无需补回。

### 2. `index.html` — 指示器与 tooltip 文案（L1135、L2726-L2728）
- `#aiChatContextUsage` 的 `aria-label` 由 `上下文使用率` 改为 `历史对话压缩进度`。
- tooltip（`data-tip="context-usage"`）文案对齐新语义：
  - 标题：`历史对话压缩进度`
  - 首行：`当前：`（保留 `<em>已用：</em>` 改为 `<em>当前：</em>`）`X / Y tokens`
  - 次行：`达到预算的 <span id="aiContextUsageTipTrigger">80%</span> 时自动压缩更早的历史（生成会话摘要），仅保留最近的预算内容。`

### 3. `ai-chat.js` — 注释与语义对齐（不涉及逻辑）
- `updateContextUsage` 顶部注释「更新上下文使用率指示器」→「更新历史对话压缩进度指示器」，并说明 used 为 tail 口径。
- 模块变量 `contextUsageEl` 注释同步（`#aiChatContextUsage`（历史对话压缩进度指示器容器））。
- tooltip 明细 `aiContextUsageTipDetail` 写入的 `used` 即 tail 估算，无需改代码（配合第1点自动正确）。
- 渲染逻辑 / warning 阈值 / 环形弧长保持不变。

### 4. `ai-chat.css` — 注释（可选，轻微）
- `.ai-context-usage` 相关注释若含「使用率」措辞可顺带改「压缩进度」，无样式变更。

### 5. 文档（可选，若需一致）
- `internal/services/AI_CONTEXT.md` 若描述此指示器为「窗口使用率」可补一句口径说明；**不作为本次必需**，默认跳过。

## Assumptions & Decisions
- 进度条语义 = **历史对话触发压缩进度**（tail / 预算），不再表达「完整请求占用」。这是用户确认的方向，也与触发判断同源。
- user 一次历史估算仍用 `EstimateTokens`（中文系数不变）；触发判断依赖的估算与展示估算同源，故展示的百分比与实际触发点一致，不会出现「显示超 100% 却不压」。
- 触发行为（`truncateAIMessages` 何时压缩、摘要生成、状态条提示）一律**保持不变**，本计划只改展示口径与文案。

## Verification
1. `go build ./...` 与 `wails build` 成功，更新 `build/bin/jot.exe`。
2. 打开已多轮 Agent 会话：指示器 `percent` = 摘要边界后 tail / 预算，**不再出现 >100%**；当 tail 达 80% 显示琥珀 warning，达触发线后自动压缩的旧行为不变。
3. 触发压缩后，tail 重置，指示器百分比回落，确认「压缩 → 回落」闭环展示正确。
4. 悬浮查看 tooltip：标题/「当前 X / Y tokens」/「80%自动压缩」文案符合新语义；`aiContextUsageTipDetail` 数值 = tail 估算。
5. 回归：发送/重发/重新生成后指示器刷新；切换会话数值正确；摘要触发生成的状态条提示（generating/done/failed）不受影响。