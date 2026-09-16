# 为 AI 助手新增工作目录文件系统与命令执行工具 Spec

## Why
当前 AI 助手只能通过 recall/browse/manage 等只读或笔记域工具完成任务，无法读写工作文件、执行脚本，
因此难以自主完成「写脚本→跑脚本→复用产物」这类重复性运维/批处理任务。本需求为其规划一个严格锁定
的 `.jot/workspace/` 工作目录，并接入文件系统操作与命令执行工具，配套三种执行授权模式以控制风险。

技术选型结论：eino 自带文件系统工具（adk/middlewares/filesystem）构造器私有且仅能经 ADK 中间件装配，
与项目 compose+brains/ReAct 架构不兼容，且其工具无任何安全边界。因此**工具自研，但参数/描述/交互契约对齐
eino 的 `adk/filesystem` 独立接口包**（该包不依赖 ADK agent），并补齐 `edit_file`/`glob`。安全层
（工作目录锁定、三种授权模式、命令黑名单、审批续跑）全部自研，eino 不参与该部分。

## What Changes
- 在 `~/.jot/` 下规划并自动创建 `workspace/` 工作目录（沿用 `internal/config` 包体系新增 `DirWorkspace`）。
- 自研 6 个工具并对齐 eino `adk/filesystem` 契约：`ls` / `read_file`（行号 offset/limit）/ `write_file` /
  `edit_file`（find+replace）/ `glob` / `execute`。全部经 `registry.go` `buildTools` 注册、`meta.go` `BuiltinTools` 展示。
- 复用 `github.com/cloudwego/eino/adk/filesystem` 的 `Backend` / `Shell` 接口契约作为我们自己后端的协议标准
  （`grep` 按最简原则暂不引入，后续可按需补）。
- `execute` 支持 shell 与 python 脚本执行（脚本落盘可后续复用），cwd 强制为 workspace，带命令超时 + 输出字节截断。
- 引入文件路径边界校验：所有文件工具与命令执行的相对路径先做归一化 + workspace 前缀检查，越界直接拒绝。
- 新增三种执行授权模式（模块级设置项，默认 `confirm_every`）：每次确认 / 全自动 / 智能审批，
  仅对上述 6 个新工具生效，既有工具行为不变（`confirm_every` 对既有工具不启用确认，避免打扰）。
- 引入异步「审批暂停/续跑」机制：执行前可暂停等用户批准，复用 `AskWaiter` 同款同轮阻塞范式，
  （即 `Context` 新增 `ApprovalWaiter`），新增后端 `ApproveToolCall(id, approved)` 供前端回传审批结果。
- 新增前端工具授权确认弹层 + 执行模式选择器，`Run` 结束前审批动作记入 `Result.ToolCalls` 以便回溯。
- **BREAKING**：无既有接口/数据结构被破坏，均为新增能力；新增依赖 `github.com/cloudwego/eino/adk/filesystem`（已有 eino 版本内）。

## Impact
- 受影响能力：Agent 工具装配（registry）、内置工具清单（meta）、工具共享上下文（context）、
  Agent Run 事件消费、前端工具面板与设置。
- 受影响代码：
  - `internal/config/config.go`（新增 workspace 子目录常量）
  - `internal/agent/registry.go`（注册 6 个工具）
  - `internal/agent/tools/context.go`（新增 ApprovalWaiter）
  - `internal/agent/tools/meta.go`（新增展示文案）
  - `internal/agent/tools/workspace_backend.go`（新：实现 `adk/filesystem.Backend`，绑定 workspace）
  - `internal/agent/tools/workspace_shell.go`（新：实现 `adk/filesystem.Shell`，cwd=workspace 带超时/截断/黑名单）
  - `internal/agent/tools/fs_tools.go`（新：按 eino 契约生成 ls/read_file/write_file/edit_file/glob 工具）
  - `internal/agent/tools/execute_tool.go`（新：execute 工具）
  - `internal/agent/agent.go`（注入 ApprovalWaiter、读取执行策略、终止态汇总审批记录）
  - `app.go`（新增前端可调用的 `ApproveToolCall` 绑定）
  - 前端 AI 助手工具面板与设置区（执行模式选择器、审批弹层）

## ADDED Requirements

### Requirement: 工作目录规划
系统 SHALL 在 `~/.jot/` 下规划并自动创建 `workspace/` 工作目录，作为所有新文件/命令工具的唯一可写根目录。

#### Scenario: 首次进入 AI 助手模块
- **WHEN** 首次进入 AI 助手模块（或首次装配新工具）
- **THEN** 自动创建 `~/.jot/workspace/`（若不存在），作为工具可写根目录

### Requirement: 文件系统与命令执行工具（对齐 eino 契约）
系统 SHALL 提供 `ls` / `read_file` / `write_file` / `edit_file` / `glob` / `execute` 六个内置工具，
其参数与行为契约对齐 eino `adk/filesystem` 的 `Backend` / `Shell` 接口（如 read_file 支持 1-based offset/limit、
edit_file 的 old_string/new_string/replace_all 语义），工具定义自研而非引入 ADK 中间件。

#### Scenario: 写入并复用脚本
- **WHEN** 模型调用 `write_file` 写入 `scripts/clean.py` 并随后调用 `execute` 执行
- **THEN** 脚本落盘到 workspace 可被后续对话 `read_file` / `edit_file` / `execute` 复用
- **AND** 命令 cwd 强制为 workspace，命令超过预设超时上限时被终止，输出超过预设字节上限时被截断

#### Scenario: 路径越界
- **WHEN** 模型传入的文件路径经归一化后落在 workspace 之外
- **THEN** 工具拒绝执行并返回含中文说明的错误回填模型

### Requirement: 三种执行授权模式
系统 SHALL 提供且仅提供三种执行模式，并存为可持久化的设置项，默认 `confirm_every`：
- `confirm_every`：每次执行 6 个新工具前暂停，等待用户批准。
- `auto`：直接执行，仅界面留记录，不触发审批。
- `review`：按「关键操作」规则判定，命中才暂停审批，其余自动执行。

`review` 模式的「关键操作」至少包含：
- 命令串命中破坏性/危险命令黑名单；
- `write_file` 目标文件已存在（覆盖写）或 `edit_file` 修改既有文件；
- 文件/目录删除类操作（若未来引入互删工具）；
- 命令包含「网络下载后直接执行」形态（如 `curl ... | sh/bash`、`wget ... |` 管道到解释器）；
- 命令涉及系统级资源（注册表、服务、防火墙、开机启动、权限提升如 sudo/su）。

#### Scenario: 默认模式
- **WHEN** 首次进入 AI 助手模块且未设置过执行模式
- **THEN** 采用 `confirm_every`，每次新工具执行均需用户批准

#### Scenario: 审批通过/拒绝
- **WHEN** 前端回传 `ApproveToolCall(id, approved=true)`
- **THEN** 暂停的该次工具调用继续执行
- **WHEN** 前端回传 `ApproveToolCall(id, approved=false)`
- **THEN** 该次调用中止，向模型回填「用户拒绝了该操作」，ReAct 循环继续
- **AND** 两种结果都记入 `Result.ToolCalls`，前端可回溯

### Requirement: 破坏性命令黑名单
系统 SHALL 内置一组常见危险命令关键词，供 `review` 模式判定；命中即需审批。黑名单（大小写不敏感、按命令串子串匹配）至少包含：

```
rm del erase format mkfs fdisk dd
shutdown poweroff reboot halt
sudo su runas gksudo gksu pkexec
chmod chown mkpasswd
del /s  rd /s rmdir /s /q diskpart disinfect
taskkill kill -9 pkill
reg delete regedit reg add
curl|sh  curl|bash  wget|bash pipe download&&exec 形态
powershell IEX iex invoke-expression
scrub memtest btrfs zfs (磁盘级)
```

#### Scenario: review 模式下命中黑名单
- **WHEN** `review` 模式下命令串命中黑名单任一关键词
- **THEN** 该命令暂停并请求用户审批

#### Scenario: blacklist 可扩展
- **WHEN** 后续需要新增危险命令
- **THEN** 在一个集中定义的常量表追加即可，不改判断逻辑

### Requirement: 执行模式选择器与审批弹层（前端）
系统 SHALL 在 AI 助手模块设置区提供三模式选择器，并在需要审批时弹出确认层（展示工具名、参数详情、
风险说明与允许/拒绝按钮，符合既有确认交互规范）。

#### Scenario: 调整模式
- **WHEN** 用户在设置区切换三种执行模式
- **THEN** 设置被持久化，后续对话按新模式执行

#### Scenario: 审批弹层
- **WHEN** 需要审批时
- **THEN** 弹层展示该次调用信息，用户可批准或拒绝，弹层可用 ESC 关闭（等价于拒绝）

## MODIFIED Requirements
无（全部为新增能力，未修改既有 Requirement）。