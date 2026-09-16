# AI Agent 文件操作与命令执行工具 Spec

## Why
`~/.jot/workspace/` 工作目录、会话审批模式 `ApprovalMode`、前端审批选择器三块基础设施已落地，但 AI 尚无在 workspace 内读写文件、执行命令的工具。本 spec 补齐「文件工具 + 命令执行工具 + 审批暂停/续跑 + 三模式接线」，使 AI 能在沙箱内完成"写脚本 → 执行 → 看结果 → 迭代"的重复性任务。

## What Changes
- **ADDED** 三个文件工具：`read_file` / `write_file` / `list_dir`，锁定 workspace 边界
- **ADDED** 一个命令工具：`run_command`，基于 `os/exec` 裸命令执行（**禁止 shell 语法**），强制 cwd=workspace，未识别命令→`exec.LookPath` 报错回填模型自适应
- **ADDED** 审批暂停/续跑机制：泛化 `ClaimAsk`+`WaitForAnswer` 范式为 `ApprovalWaiter`，新增 `ai:tool-approval` 事件 + `ApproveToolCall` 绑定 + 前端确认面板
- **ADDED** 审批门接入 `approval_mode` 三模式（confirm_every / review / auto）+ 命令黑名单
- **BREAKING** 无既有工具被删；`run_command` 为全新工具，无需迁移 `ai_agent_tools_disabled`

## Impact
- Affected specs: agent 工具注册表、agent 事件协议、审批模式字段
- Affected code:
  - `internal/config/config.go`（workspace 边界校验辅助）
  - `internal/agent/registry.go`（工具注册）
  - `internal/agent/tools/context.go`（`wrappedTool` 前置审批门 + `ApprovalWaiter` 接口）
  - `internal/agent/tools/fs_tools.go`（新增，read_file/write_file/list_dir）
  - `internal/agent/tools/run_command.go`（新增）
  - `internal/agent/agent.go`（`agentSession` 增加 approve waiter 实现 + 事件发射）
  - `internal/services/ai_service.go`（读 `approval_mode`）
  - `app.go`（新增 `ApproveToolCall` 绑定方法）
  - `frontend/src/js/ai-chat.js`、`frontend/index.html`、`frontend/src/css/components/ai-chat.css`（确认面板）
  - `internal/agent/TOOLS.md` / `EVENTS.md`（文档同步）

## ADDED Requirements

### Requirement: workspace 路径边界校验
系统 SHALL 提供统一辅助函数，将任意给定路径 `filepath.Clean` + `Abs` 后校验是否落在 `~/.jot/workspace/` 内，落出直接拒绝并返回用户友好错误。

#### Scenario: 绝对路径越界
- **WHEN** 工具收到 `/etc/passwd` 或 `C:/Windows/...` 这类 workspace 外路径
- **THEN** 工具拒绝执行，返回"超出工作目录"错误，`wrappedTool` 回填模型继续推理

### Requirement: read_file
系统 SHALL 提供 `read_file` 工具，读取 workspace 内指定文件（参数 `path`、可选 `offset`/`length` 分页），超长用 rune 分页截断并附续读提示。

#### Scenario: 分页读取大文件
- **WHEN** 读取内容超出单页上限
- **THEN** 返回当前分页片段 + "未读完，可 offset=N 继续"提示，越界报"已全部读取"

### Requirement: write_file
系统 SHALL 提供 `write_file` 工具，在 workspace 内写入文件（参数 `path`/`content`，可选 `append`），自动 `MkdirAll` 父目录；覆盖已存在文件视为破坏性操作（阶段③ 起触发审批）。

#### Scenario: 覆盖已存在文件
- **WHEN** `write_file` 覆盖 workspace 内既有文件且审批模式需要确认
- **THEN** 工具在执行前暂停，等用户批准/拒绝

### Requirement: list_dir
系统 SHALL 提供 `list_dir` 工具，递归列出 workspace 内目录（深度上限），返回相对路径 + 类型。

### Requirement: run_command
系统 SHALL 提供 `run_command` 工具，基于 `os/exec` 执行**裸命令 + 参数数组**（参数 `command`、`args`、可选 `cwd`）。强制 cwd=workspace；**禁止 shell 语法**；`exec.CommandContext` 携带超时；输出截断回填。

#### Scenario: 执行系统不存在命令
- **WHEN** 模型调用不存在的命令或传入 shell 表达式（如 `ls -al | grep foo`）
- **THEN** `exec.LookPath` 报"环境无此命令"回填模型，模型据此改用已有命令或文件工具（不禁止、报错自适应）

### Requirement: 审批暂停/续跑机制
系统 SHALL 泛化 ask_user 的等待范式为可复用 `ApprovalWaiter`（`BeginApproval` 原子抢占 + 发射 `ai:tool-approval` 事件 + `WaitDecision` 阻塞），供 `wrappedTool` 在执行危险工具前调用。新增绑定方法 `ApproveToolCall(sessionID, callID, approved)`；拒绝=工具返回"用户已拒绝"回填模型继续；审批动作记入 `toolRecords`。

#### Scenario: 工具等待审批
- **WHEN** 危险工具命中审批门且模式需要确认
- **THEN** AI 流暂停，前端弹确认面板，用户允许/拒绝后刷放行/作废该调用

### Requirement: 审批门接入三模式
系统 SHALL 依据会话 `approval_mode` 决定是否暂停审批：`confirm_every` 对所有危险操作等待；`review` 仅黑名单命中才等待；`auto` 不等待（但黑名单命中仍须确认，作最后防线）。命令黑名单含 `rm`/`del`/`rmdir`/`shutdown`/`reboot`/`mkfs`/`format`/`dd` 写盘/联网拉取执行(`wget|curl|pip|npm|go install|git clone`)/`sudo`/`systemctl` 等。

#### Scenario: auto 模式仍拦黑名单
- **WHEN** `approval_mode=auto` 但命令命中黑名单
- **THEN** 仍暂停要求确认，避免完全失控

## MODIFIED Requirements
无既有功能被修改（`ApprovalMode` 字段在上一 spec 已落地，本 spec 仅接线）。

## REMOVED Requirements
无。