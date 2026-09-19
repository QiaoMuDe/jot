# Tasks

- [x] Task 1: 后端门控与状态生命周期：为 `agentSession` 增加本轮放行状态，并在 `RequestApproval` 中短路、在 `Run` 结束时复位。
  - [x] SubTask 1.1: `agentSession` 新增字段 `allowRoundAll atomic.Bool`（`sync/atomic`，注释说明「本轮 = 一次 Run/循环」语义，非会话级、不落库）；同步更新结构体上方注释。
  - [x] SubTask 1.2: 新增方法 `resetRoundAllow()`（幂等置 false），并在 `Run` 的 `defer` 清理块中调用（与 `approvePending = false` 同处，保证正常/报错/停止/会话释放四条路径均复位）。
  - [x] SubTask 1.3: `RequestApproval` 开头插入短路分支：`allowRoundAll.Load()` 为 true → `critical` 时写本轮授权留痕（见 Task 3）、随后 `return nil`；该分支必须位于 `claimApproval` 之前（不占用审批名额，不破坏「并行危险操作仅一条等待」语义），且位于 `needConfirm` 计算之前。

- [x] Task 2: 后端审批决定通道升级与绑定签名变更（**BREAKING**）。
  - [x] SubTask 2.1: 新增类型 `approvalDecision{approved bool; allowRound bool}`，`agentSession.approveCh` 由 `chan bool` 改为 `chan approvalDecision`；同步 `drainApproval`。
  - [x] SubTask 2.2: `ApproveToolCall(sessionID uint, approvalID uint64, approved bool, allowRound bool) error`：校验（会话存在 / `approvePending` / `pendingApprovalID` 匹配）通过后，**先** `allowRound && approved` 置位 `allowRoundAll`，**再**投递决定（顺序不可颠倒，保证下一工具调用即读到授权）；校验失败路径不得置位。
  - [x] SubTask 2.3: `app.go` 的 `ApproveToolCall` 绑定同步加参并更新注释；执行 `wails generate module` 重生成 `frontend/wailsjs/go/main/App.js` 与 `App.d.ts`。

- [x] Task 3: 审批留痕三态化。
  - [x] SubTask 3.1: `recordApproval` 改为接收 `approvalDecision`，`Result` 三态：`allowRound`→「已允许本轮操作（本轮后续操作自动放行）」、`approved`→「批准」、否则→「已被用户拒绝」。
  - [x] SubTask 3.2: `recordAutoApproval` 增加来源文案参数（`reason string`），`auto` 模式调用处传「完全访问模式自动放行（高危命令，未经人工确认）」；新增本轮授权专用 helper（或复用同函数）传「本轮已授权自动放行（高危操作，用户在审批面板选择了「允许本轮」）」；`action` 保持 `tool_auto_approval` 不变（前端零改动）。

- [x] Task 4: 前端审批面板交互。
  - [x] SubTask 4.1: `showApprovalPanel` 按钮行新增第三个按钮「允许本轮」（class `ai-approval-btn round`，文案与 `title` 明示范围：本轮内后续所有操作（含高风险）自动执行、不再询问）；调用 `ApproveToolCall(activeSessionId, approvalId, true, true)`，成功后 `hideApprovalPanel()` 并弹一次提示（如「本轮已完全放行：后续操作将自动执行，不再询问」，warning 级）。
  - [x] SubTask 4.2: 既有「拒绝」「允许」调用点补第 4 参 `false`。
  - [x] SubTask 4.3: `disabledBy` 改为遍历 `actions.querySelectorAll('.ai-approval-btn')`，覆盖第三个按钮的防重复提交。
  - [x] SubTask 4.4: 高危面板提示条文案由「高风险操作，无法绕过确认」改为「高风险操作，请谨慎确认」（与「允许本轮」能力自洽）。

- [x] Task 5: 主题变量与按钮样式。（实际主题块为 12 个 + `:root` 兜底，共 13 处定义；`dracula`/`quiet-light` 因 accent 本身为紫改用青色系，`mono` 为纯灰主题故引入紫色为唯一彩色）
  - [x] SubTask 5.1: `frontend/src/css/variables.css` 在全部 11 个主题块（含 `:root` 默认块）各新增一行 `--approve-round`，取值原则：与该主题 `--accent`、`--error` 色相明显区分（优先紫/青色系），明暗主题分别取适配亮度的实底色。
  - [x] SubTask 5.2: `frontend/src/css/components/ai-chat.css` 新增 `.ai-approval-btn.round` 规则（`background: var(--approve-round)`，文字色复用 allow 的 `color-mix` 深色推导法保证各主题对比度）；`is-critical` 面板下保持不变式（不与 allow 的红色规则冲突），必要时加 1px 描边强化。
  - [x] SubTask 5.3: 三个按钮排布与视觉分隔：`拒绝 | 允许 | 允许本轮`（范围最大的按钮置于行末最右，与其余两个留出间隙，降低误点概率）。

- [x] Task 6: 单元测试。
  - [x] SubTask 6.1: `internal/agent/approval_test.go` 新增 `TestRequestApprovalRoundAllow`：`confirm_every` 下普通操作阻塞 → 投递 `{approved:true, allowRound:true}` → 同轮后续 `critical=false` 与 `critical=true` 均直接返回 nil 且不阻塞。
  - [x] SubTask 6.2: 新增 `TestRequestApprovalRoundAllowAudit`：断言授权后 `critical=true` 追加一条 `tool_auto_approval`（文案含「本轮已授权自动放行」），`critical=false` 不追加；对比 `auto` 模式文案含「完全访问模式自动放行」。
  - [x] SubTask 6.3: 新增 `TestApproveToolCallRoundScope`：无等待中审批 / `approval_id` 不匹配时投递 `allowRound=true` 必须报错且**不置位**放行状态（防串审置位）。
  - [x] SubTask 6.4: 新增 `TestResetRoundAllow`：置位后调用 `resetRoundAllow()`，`RequestApproval` 恢复阻塞（覆盖「本轮结束失效」）。
  - [x] SubTask 6.5: 回归既有 `TestRequestApprovalModeGating` / `TestRequestApprovalRecordsDecision` / `TestRequestApprovalRejectsParallel`，按新签名与新记录文案调整断言。

- [x] Task 7: 文档同步（项目硬性约定）。
  - [x] SubTask 7.1: `internal/agent/EVENTS.md` §5：负载说明补「本轮放行」瞬态；门控表新增该状态；`critical` 语义句（原「不应提供忽略直接执行语义」）改为「仅『允许本轮』可放行且范围限本轮、高危后续留痕」；回调语义补第 4 参 `allowRound`；末尾「前端隐藏面板」说明补「本轮授权由后端复位，前端无需清理授权状态」。
  - [x] SubTask 7.2: `AGENTS.md` 长期记忆第 5 条「审批门控公式」更新为「先判本轮放行，再走 `needConfirm := mode=="confirm_every" || (critical && mode=="review")`」，并注明本轮 = 一次 Run、高危后续写 `tool_auto_approval`。
  - [x] SubTask 7.3: `internal/agent/TOOLS.md` 补一句「本轮放行由 `RequestApproval` 统一短路，工具侧无需感知，5 步审批接入流程不变」。
  - [x] SubTask 7.4: `frontend/src/css/theme-maintenance.md` 登记新变量 `--approve-round`（新增主题块必须包含该变量，否则按钮回落 undefined），并把本次变更计入 AGENTS.md 临时记忆（按三步顺移规则追加为临时记忆 5）。

- [x] Task 8: 全量验证。
  - [x] SubTask 8.1: `gofmt -l internal/ app.go` 无输出；`go build ./...`、`go vet ./...` 通过。
  - [x] SubTask 8.2: `go test ./internal/...`（含新增 4 组与既有审批用例）全绿；`golangci-lint run ./internal/...` 0 issues。
  - [x] SubTask 8.3: `npm run build` 通过（仅既有 chunk 警告）；13 处 `--approve-round` 逐一静态核对与该主题 `--accent`/`--error` 色相不同（观感终检需人工目视）。
  - [x] SubTask 8.4: `wails build` 出新二进制（端到端点击验证需人工在应用内执行：手动审批模式发多步任务 → 首次弹窗点「允许本轮」→ 后续普通与高危操作均不弹窗且工具记录出现审计行 → 本轮结束后新一轮恢复弹窗）。

- [x] Task 9: 修复测试审查发现的边界不一致（`ApproveToolCall` 决定自洽化）。
  - [x] SubTask 9.1: 决定构造时规范化为 `allowRound: allowRound && approved`，杜绝「返回拒绝错误但留痕写已允许本轮」的矛盾。
  - [x] SubTask 9.2: 投递失败（通道满 `default` 分支）时回滚已置位的本轮授权。
  - [x] SubTask 9.3: `ApproveToolCall` 函数级注释已同步（生效条件 + 失败回滚）。

- [x] Task 10: 消除残留的「无法绕过」绝对表述（与放行能力保持一致）。
  - [x] SubTask 10.1: `frontend/index.html` 审批模式「自动审批」悬停提示改为「默认不可绕过……如需连续执行，可在审批面板选择『允许本轮』临时放行本轮内后续操作」。
  - [x] SubTask 10.2: `frontend/src/css/components/ai-chat.css` 两处注释由「无法绕过（确认）」改为「需谨慎确认」，与新语义一致。

- [x] Task 11: 按用户反馈调整审批面板按钮布局与高危配色。
  - [x] SubTask 11.1: 「允许本轮」由按钮行末移至**行首最左**（挂载顺序改为 round → deny → allow），留白由 `margin-left` 改为 `margin-right`。
  - [x] SubTask 11.2: 删除 `.ai-approval-panel.is-critical .ai-approval-btn.allow` 红色覆盖规则，使「允许」在高危面板下也保持主题强调色，三按钮颜色在全部面板恒定（紫 / accent / 红），仅凭颜色即可区分动作；高危警示由面板左侧红边与提示条承担，并在 `.allow` 注释中记录该决策防止回退。

# Task Dependencies

- Task 2 依赖 Task 1（`approvalDecision` 与状态字段）
- Task 3 依赖 Task 2（`recordApproval` 入参类型）
- Task 4 依赖 Task 2（`ApproveToolCall` 新签名）
- Task 5 独立（纯样式，可与 Task 3 / 4 并行）
- Task 6 依赖 Task 1~3
- Task 7 依赖 Task 1~4（文案与公式定稿后同步）
- Task 8 依赖全部
