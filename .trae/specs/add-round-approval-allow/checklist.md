# Checklist

> 验证方式：静态代码核查（含独立子代理 31 项逐条核查）+ 命令验证（gofmt / go build / go vet / go test / golangci-lint / npm run build / wails build）。纯观感类项已在条目末尾标注「人工终检」。

## 后端门控

- [x] `agentSession` 新增本轮放行状态字段（`allowRoundAll atomic.Bool`），注释明确「本轮 = 一次 Run（一条用户消息触发的完整 ReAct 循环），非会话级、不落库」
- [x] `RequestApproval` 中本轮放行短路分支位于 `claimApproval` 之前，且位于 `needConfirm` 计算之前（不占审批名额、不被审批模式覆盖）
- [x] 本轮已授权时，`critical=false` 与 `critical=true` 的操作均直接返回 nil、不发射 `ai:tool-approval`（短路分支内仅高危时写 `tool_auto_approval`，无审批事件发射）
- [x] 未点击「允许本轮」时，`confirm_every` / `review` / `auto` 三种模式的既有门控行为与文案完全不变（`needConfirm` 公式与 auto 留痕文案逐字一致）
- [x] 放行状态在 `Run` 的 `defer` 清理块中复位（`resetRoundAllow()` 与 `approvePending = false`、`setRunCancel(nil)`、`runCancel()` 同块），正常结束 / 报错 / 停止 / 会话释放四条路径均不残留
- [x] 新一轮对话（新 `Run`）恢复按会话审批模式弹窗（不跨消息继承授权；授权仅内存态，未写入 `services`/`models`/`database` 任何持久化位置）

## 审批决定通道与绑定

- [x] `approveCh` 已升级为携带 `{approved, allowRound}` 的 `approvalDecision`，`drainApproval` 同步适配（无赋值接收，类型无关）
- [x] `ApproveToolCall` 新增 `allowRound` 参数；`allowRound` 仅在 `approved=true` 时生效（决定构造时已规范化 `allowRound && approved`），且在**投递之前**置位放行状态（保证下一工具调用即读到授权）
- [x] 会话不存在 / 无等待中审批 / `approval_id` 不匹配 三条校验失败路径均返回中文错误且**不置位**放行状态（置位语句在三处校验之后）
- [x] 投递失败（通道满 `default` 分支）会回滚已置位的本轮授权，消除「返回错误但已放开」窗口
- [x] `app.go` 绑定已同步 4 参签名并转发第 4 参；`wails generate module` 后 `frontend/wailsjs/go/main/App.js`（`arg1~arg4`）与 `App.d.ts`（`arg4:boolean`）与后端签名一致

## 留痕

- [x] `tool_approval` 记录支持三态文案（批准 / 已被用户拒绝 / 已允许本轮操作（本轮后续操作自动放行）），点击「允许本轮」当次记录为第三态
- [x] 本轮授权后，后续 `critical=true` 操作追加 `tool_auto_approval` 记录，文案为「本轮已授权自动放行（高危操作，用户在审批面板选择了「允许本轮」）」
- [x] 本轮授权后，后续 `critical=false` 操作不额外追加审计记录（仅正常工具记录）
- [x] `auto` 模式留痕文案仍为「完全访问模式自动放行（高危命令，未经人工确认）」，两种来源可区分
- [x] `tool_auto_approval` 的 `action` 字段未变，前端既有高亮渲染分支（`ai-chat.js` 按 action 判断）无需新增分支

## 前端

- [x] 审批面板在「拒绝」「允许」之外渲染第三个按钮「允许本轮」（`ai-approval-btn round`），`title` 明示「本轮内后续所有操作（含高风险）自动执行，不再询问」
- [x] 「拒绝」「允许」调用点补齐第 4 参 `false`；「允许本轮」传 `(true, true)`
- [x] 防重复提交覆盖三个按钮（`disabledBy` 遍历 `querySelectorAll('.ai-approval-btn')`）
- [x] 点击「允许本轮」成功后隐藏面板并给出一次 warning 提示（「本轮已完全放行：后续操作将自动执行，不再询问」）
- [x] 高危面板提示条文案为「高风险操作，请谨慎确认」；`ai-chat.js`、`EVENTS.md` 及全仓库前端代码中不再出现「无法绕过确认」绝对表述（`index.html` 审批模式悬停提示与 `ai-chat.css` 注释亦已同步为「默认不可绕过 + 可选『允许本轮』」/「需谨慎确认」）

## 按钮配色（全主题）

- [x] `variables.css` 中 `--approve-round` 定义 **13 处** = 12 个 `[data-theme]` 主题块（default/light/dark/nord/tokyo-night/eye-protection/catppuccin-latte/gruvbox-light/dracula/quiet-light/ysgrifennwr/mono）+ 1 个 `:root` 兜底，无遗漏（每块均写在主题块内部，遵循「每个主题自包含」约定）
- [x] 逐主题核对：`--approve-round` 与该块 `--accent`、`--error` 色相明显不同（`dracula`/`quiet-light` 因 accent 本身为紫改用青色系 `#22D3CE`/`#0EA5A5`；`mono` accent 与 error 同为近黑灰，故用紫色 `#7C6CF0` 作为该主题唯一彩色；其余主题均为紫色系；独立核查已抽查 default/dracula/quiet-light/mono 四例）
- [x] `.ai-approval-btn.round` 使用 `var(--approve-round)` 实底 + 与 `.allow` 完全相同的 `color-mix` 深色推导文字色；纯观感对比度终检需人工目视（推导法与项目既有 `.allow` 一致）
- [x] 三个按钮排布为「允许本轮 | 拒绝 | 允许」，范围最大的按钮置于行首最左并 `margin-right: 6px` 额外留白，降低误点
- [x] 三按钮颜色在所有面板恒定、仅凭颜色即可区分动作：允许本轮=`--approve-round`（紫/青）、允许=`--accent`、拒绝=`--error`；原 `.ai-approval-panel.is-critical .ai-approval-btn.allow` 的红色覆盖规则已删除，高危警示由面板左侧红边与「高风险操作，请谨慎确认」提示条承担

## 子 Agent 与边界

- [x] os_agent 内层工具复用同一会话 Approver，本轮授权自动生效（`internal/agent/tools/` 下无任何改动，放行逻辑集中在会话层）
- [x] Plan 模式执行阶段与同轮 `ask_user` 续答处于同一 `Run`，授权口径一致（短路由 `agentSession` 单点判定，与调用方无关）
- [x] 并行工具调用场景下，本轮放行分支不占用审批名额（短路位于 `claimApproval` 之前），既有「仅一条进入等待、其余报错」语义未被破坏（`TestRequestApprovalRejectsParallel` 仍绿）

## 测试与静态检查

- [x] `approval_test.go` 新增 4 组：`TestRequestApprovalRoundAllow`（授权后普通与高危均直接放行且不再发射审批事件）、`TestRequestApprovalRoundAllowAudit`（本轮授权高危留痕文案 + 普通不留痕 + 第三态记录）、`TestApproveToolCallRoundScope`（校验失败不置位）、`TestResetRoundAllow`（复位后恢复阻塞）
- [x] 既有 `TestRequestApprovalModeGating` / `TestRequestApprovalRecordsDecision` / `TestRequestApprovalRejectsParallel` / `TestRequestApprovalApprovedDelivers` 等已按新签名调整并全绿
- [x] `gofmt -l internal/ app.go` 无输出；`go build ./...`、`go vet ./...` 通过
- [x] `go test ./internal/... -count=1` 全绿（agent / agent/tools / aierrors / config / mcpserver / services 均 ok）
- [x] `golangci-lint.exe run ./internal/...` 输出 `0 issues.`
- [x] `npm run build` 通过（仅既有 chunk 体积警告）；`wails build` 通过并产出新二进制

## 文档与约定

- [x] `internal/agent/EVENTS.md` §5 已更新：门控表新增「本轮放行（瞬态，优先于三模式）」、`critical` 语义句改写为「默认不可绕过 + 仅『允许本轮』可放行且限本轮、高危逐条留痕」、回调签名补第 4 参 `allowRound` 及时序说明、末尾补「授权由后端复位，前端无需清理」
- [x] `AGENTS.md` 长期记忆第 5 条「审批门控公式」已更新为「先判本轮放行（`allowRoundAll`，一次 `Run` 结束复位），未授权再走 `needConfirm := mode=="confirm_every" || (critical && mode=="review")`」，并注明高危后续写 `tool_auto_approval`
- [x] `internal/agent/TOOLS.md` 已注明「『本轮放行』对工具透明，由 `RequestApproval` 统一短路，5 步审批接入流程与分级规则不变」
- [x] `frontend/src/css/theme-maintenance.md` 已登记新变量 `--approve-round`（出现 1 次，要求新增主题块必须定义）
- [x] `AGENTS.md` 临时记忆已按三步规则追加：`### 临时记忆` 共 5 条且编号连续 1~5，最新第 5 条记录本次「本轮审批放行」变更，全部使用项目相对路径、未记录行数/大小统计
- [x] 无数据库改动：未新增表/字段/设置项，`internal/models/` 与 `internal/database/` 未改动
- [x] 各工具文件未改动（`fs_base.go` / `delete_item.go` / `http_request.go` / `manage_*.go` 中无「本轮」/`allowRound` 相关新增）
