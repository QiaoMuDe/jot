# 本轮审批放行 Spec

## Why

当前审批只有三个模式（`confirm_every` / `review` / `auto`），前两种模式下每次工具调用都要单独弹窗确认。Agent 执行一个包含十几步文件/命令操作的任务时，用户必须点十几次「允许」，交互成本高且容易疲劳误点。需要一个「本轮放行」的即时授权入口：用户在审批面板点一次「允许本轮」，本**次循环**内后续操作不再弹窗。

## What Changes

- 后端 `agentSession` 新增本轮放行状态（内存态，不落库、不跨消息、不跨会话）。
- `RequestApproval` 增加门控短路：本轮已授权 → 普通与**高危**操作均直接放行；高危操作额外写 `tool_auto_approval` 审计留痕（文案区分「本轮授权」与「完全访问」）。
- 审批决定通道由「一次性 bool」升级为 `approvalDecision{approved, allowRound}`；**BREAKING**：Wails 绑定 `ApproveToolCall` 增加第 4 个参数 `allowRound bool`，前端调用点与 `frontend/wailsjs` 绑定需同步重生成。
- 前端审批面板新增第三个按钮「允许本轮」，配色为**独立主题变量**，与「拒绝」（error 实底）和「允许」（accent 实底）在 11 个主题下均明显不同。
- 审批留痕文案区分三种结果：批准 / 已被用户拒绝 / 已允许本轮。
- **`review` / `confirm_every` 下高危操作「不可绕过」的既有基线被放宽**（仅当用户显式点击「允许本轮」时，且范围严格限定本轮）。

## Impact

- Affected specs：`ai-workspace-file-command-tools`、`add-agent-os-subagent`（审批语义相关章节）、`add-sub-agent-max-iterations-setting`（无功能影响，仅文档同源）。
- Affected code：
  - `internal/agent/agent.go`（`agentSession`、`RequestApproval`、`recordApproval`、`recordAutoApproval`、`ApproveToolCall`、`Run` 清理）
  - `app.go`（`ApproveToolCall` 绑定签名）
  - `frontend/index.html`（无结构改动，面板为 JS 动态渲染）
  - `frontend/src/js/ai-chat.js`（`showApprovalPanel`、按钮调用）
  - `frontend/src/css/variables.css`（11 个主题块新增 1 个变量）
  - `frontend/src/css/components/ai-chat.css`（审批按钮样式）
  - `internal/agent/EVENTS.md`、`internal/agent/TOOLS.md`、`AGENTS.md`（长期记忆第 5 条门控公式）、`frontend/src/css/theme-maintenance.md`
  - 无数据库改动（不新增表/字段/设置项，`database/models.go` 与 `ResetDatabase` 无需改动）

## ADDED Requirements

### Requirement: 本轮放行入口

审批面板 SHALL 在「拒绝」「允许」之外提供第三个按钮「允许本轮」，仅当后端发射 `ai:tool-approval` 时渲染。

#### Scenario: 普通操作弹窗
- **WHEN** 审批面板因 `critical=false` 的操作弹出
- **THEN** 面板显示三个按钮：拒绝、允许、允许本轮

#### Scenario: 高危操作弹窗
- **WHEN** 审批面板因 `critical=true` 的操作弹出
- **THEN** 同样显示三个按钮；高风险提示条文案不再声称「无法绕过确认」，改为「高风险操作，请谨慎确认」

### Requirement: 本轮放行门控

后端 SHALL 以「一次 `Run`（一条用户消息触发的完整 ReAct 循环）」为「本轮」边界维护放行状态；本轮已授权时，`RequestApproval` 对普通与高危操作一律直接放行、不发射 `ai:tool-approval`。

#### Scenario: 授权后普通操作自动执行
- **WHEN** 用户在审批面板点击「允许本轮」，随后同一轮内产生 `critical=false` 的操作
- **THEN** 不弹窗、不阻塞，工具直接执行

#### Scenario: 授权后高危操作自动执行并留痕
- **WHEN** 用户点击「允许本轮」，随后同一轮内产生 `critical=true` 的操作
- **THEN** 不弹窗、不阻塞，工具直接执行，且工具调用记录中追加一条 `tool_auto_approval` 审计记录，文案为「本轮已授权自动放行（高危操作，用户在审批面板选择了「允许本轮」）」

#### Scenario: 授权范围覆盖子 Agent
- **WHEN** 本轮已授权，os_agent 子 Agent 内层工具发起审批请求
- **THEN** 与主 Agent 一致直接放行（复用同一会话 Approver，无需额外装配）

### Requirement: 本轮状态生命周期

本轮放行状态 SHALL 仅存活于单次 `Run`，在 `Run` 返回（正常/报错/停止/会话释放）时复位。

#### Scenario: 本轮结束后失效
- **WHEN** 一轮对话结束，用户发送下一条消息开启新 `Run`
- **THEN** 审批恢复按会话审批模式弹窗（上一轮的授权不生效）

#### Scenario: 异常路径不残留
- **WHEN** 本轮因停止按钮 / 流式报错 / 会话释放而终止
- **THEN** 放行状态被复位，不残留到下一轮

### Requirement: 本轮放行按钮配色

「允许本轮」按钮 SHALL 使用独立主题变量取值，在全部 11 个主题下与「拒绝」（`--error`）和「允许」（`--accent`，高危面板下为 `--error`）色相明显不同。

#### Scenario: 明暗主题下均可辨识
- **WHEN** 依次切换 11 个系统主题打开审批面板
- **THEN** 三个按钮两两可辨，「允许本轮」不与该主题的 `--accent` 或 `--error` 呈现同色

## MODIFIED Requirements

### Requirement: 审批决定回调

原：`ApproveToolCall(sessionID uint, approvalID uint64, approved bool) error` —— `approved=true` 批准本次，`false` 拒绝本次。

现：`ApproveToolCall(sessionID uint, approvalID uint64, approved bool, allowRound bool) error` —— 在前者语义之上增加 `allowRound`：为 `true` 时（必须 `approved=true`）在投递决定**之前**置位本轮放行状态，使当前工具返回后紧接着的下一个工具调用即读到已授权。

- 校验顺序不变：会话不存在 / 无等待中审批 / `approval_id` 不匹配 → 返回中文错误，且**不得置位**本轮放行状态。
- `approval_id` 不匹配仍报错（防串审），语义不弱化。

### Requirement: 各审批模式门控

原：`needConfirm := mode == "confirm_every" || (critical && mode == "review")`

现：先判本轮放行，再走原公式：

```
若 本轮已授权                 → 放行（critical=true 时写 tool_auto_approval 留痕）
否则 needConfirm := mode == "confirm_every" || (critical && mode == "review")
```

`auto` 模式行为不变（含 `critical=true` 的 `tool_auto_approval` 留痕）；`confirm_every` / `review` 在未授权时行为不变。

### Requirement: 审批留痕文案

`tool_approval` 记录（`recordApproval`）由「批准 / 已被用户拒绝」两态扩展为三态：批准 / 已被用户拒绝 / **已允许本轮操作（本轮后续操作自动放行）**。

`tool_auto_approval` 记录（`recordAutoApproval`）新增来源文案参数，取值：
- `auto` 模式放行 →「完全访问模式自动放行（高危命令，未经人工确认）」（保持原文案）
- 本轮授权放行 →「本轮已授权自动放行（高危操作，用户在审批面板选择了「允许本轮」）」

`action` 字段保持 `tool_auto_approval`，前端既有渲染分支（`ai-chat.js` 按 action 高亮渲染）无需新增分支。

## REMOVED Requirements

### Requirement: 高危操作「无法绕过确认」的绝对语义

**Reason**：用户明确要求高危操作也纳入本轮放行。原基线（`critical=true` 时前端不提供"忽略直接执行"语义，`review` 模式后端强制阻塞）会与本功能直接冲突。

**Migration**：语义收敛为「默认仍不可绕过；仅当用户在本轮内显式点击『允许本轮』时放行，且范围严格限定本轮、高危后续逐条写审计留痕」。未点击「允许本轮」时，`confirm_every` / `review` 的行为与文案完全不变。
