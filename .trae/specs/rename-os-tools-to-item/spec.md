# 重命名 os_agent 文件/目录工具 Spec

## Why

`delete_file` / `copy_file` / `move_file` / `transfer_file` 四个工具实际都支持「文件 + 目录」操作（rm / cp / mv / 工作区↔桌面传输语义，目录均递归处理），名字却以 `_file` 限定，与功能不符，误导模型选择与用户审批判断。

## What Changes

- **BREAKING**：4 个工具重命名（工具名是模型调用契约，新会话按新名调用）：
  - `delete_file` → `delete_item`
  - `copy_file` → `copy_item`
  - `move_file` → `move_item`
  - `transfer_file` → `transfer_item`
- 同步工具实现：文件重命名（一文件一工具规范）、struct 名、构造器名（`NewDeleteItem` 等）、`Info().Name`、ActionText 动作文案（「删除项：」等）、审批 summary 文案、错误文案前缀、文件头注释。
- 同步装配：`toolConstructors` 注册表 key、`os_agent` 内层白名单、`subagent_os.go` 文件头注释与提示词。
- 同步文档与测试：`tools/doc.go`、SUBAGENTS.md、AGENTS.md、`fs_operate_test.go`、`transfer_file_test.go`、`subagent_test.go`。
- 不涉及：`read_file`/`write_file`/`edit_file`/`grep_file`（只操作文件，名实相符）、`ls_dir`/`mkdir_dir`（只操作目录，名实相符）、`glob`/`run_command`/`run_python`（中性名）保持原名；`meta.go`/BuiltinTools 不涉及（内层工具不进）；前端无需改动（工具名动态渲染）。

## Impact

- Affected specs: Agent 工具体系（os_agent 子 Agent 白名单）
- Affected code:
  - `internal/agent/tools/delete_file.go` → `delete_item.go`（及 struct/构造器/文案）
  - `internal/agent/tools/copy_file.go` → `copy_item.go`
  - `internal/agent/tools/move_file.go` → `move_item.go`
  - `internal/agent/tools/transfer_file.go` → `transfer_item.go`（测试同名改）
  - `internal/agent/subagent.go`（toolConstructors）
  - `internal/agent/subagent_os.go`（白名单 + 提示词 + 注释）
  - `internal/agent/subagent_test.go`、`internal/agent/tools/fs_operate_test.go`、`internal/agent/tools/transfer_file_test.go`
  - 文档：`internal/agent/tools/doc.go`、`internal/agent/SUBAGENTS.md`、`AGENTS.md`

## ADDED Requirements

（本次为纯改名，无新增功能需求。）

## MODIFIED Requirements

### Requirement: os_agent 工具命名契约
工具名须与实际能力一致：同时支持「文件 + 目录」的操作，不得以 `_file` 命名（`read_file` 等只操作文件的工具除外）。

#### Scenario: 目录级操作
- **WHEN** 模型调用 `delete_item` / `copy_item` / `move_item` / `transfer_item` 操作目录
- **THEN** 工具名、tool_start 动作文案、审批摘要、错误提示均以「项」语义表述，不再出现 `file` 字样，且目录操作行为（递归/recursive 门控/覆盖审批）与改名前完全一致

### Requirement: 改名后的装配一致性
工具重命名后，os_agent 内层白名单、`toolConstructors` 注册表、文档与测试必须同步，不得残留旧名编译引用。

#### Scenario: 全量构建与测试
- **WHEN** 执行 `golangci-lint run ./...`、`go build ./...`、`go test ./internal/agent/...` 及全仓 grep
- **THEN** 无旧名（`delete_file`/`copy_file`/`move_file`/`transfer_file`）的代码/白名单/测试引用残留（AGENTS.md 历史记忆中的改名说明除外）；白名单与注册表名称一致；测试全绿

## REMOVED Requirements

（无移除。旧工具名 `delete_file`/`copy_file`/`move_file`/`transfer_file` 为 **BREAKING** 重命名，**迁移**：新会话模型按新名调用；旧会话已存储的 toolRecords 为纯文本记录，前端按记录动态渲染不受影响，无需兼容层。）
