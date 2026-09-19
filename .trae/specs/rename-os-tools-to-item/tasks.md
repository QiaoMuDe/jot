# Tasks

- [x] Task 1: 重命名 4 个工具实现文件与符号
  - [x] 删除旧文件 `delete_file.go` / `copy_file.go` / `move_file.go` / `transfer_file.go`，新建 `delete_item.go` / `copy_item.go` / `move_item.go` / `transfer_item.go`
  - [x] struct 改名：`deleteFileTool` → `deleteItemTool`（copy/move/transfer 同理）
  - [x] 构造器改名：`NewDeleteFile` → `NewDeleteItem`（其余同理），`Info().Name` 改 `"delete_item"` 等
  - [x] ActionText 动作文案：「删除文件：」→「删除项：」、「复制文件：」→「复制项：」、「移动文件：」→「移动项：」、「传输文件：」→「传输项：」（transfer 的 download/upload summary 前缀同义调整）
  - [x] 审批 summary 文案与错误文案前缀（`delete_item 参数缺少 path` 等）同步
  - [x] 文件头注释同步（引用旧名/旧 struct 处）
- [x] Task 2: 装配层同步（subagent.go / subagent_os.go）
  - [x] `toolConstructors`：key 改 `copy_item`/`move_item`/`delete_item`/`transfer_item`，值改 `tools.NewCopyItem` 等
  - [x] `osSubAgentToolNames` 白名单 4 项改名（顺序保持）
  - [x] `subagent_os.go` 文件头注释（13 个工具清单）与提示词（transfer_file 引导句、危险操作句 → transfer_item）同步
- [x] Task 3: 测试同步
  - [x] `fs_operate_test.go`：`copy_file`/`move_file`/`delete_file` 工具名断言与构造 helper 改新名
  - [x] `transfer_file_test.go` → `transfer_item_test.go`：文件名、断言、构造器引用改新名
  - [x] `subagent_test.go`：核对注释（白名单数量断言由 `osSubAgentToolNames` 驱动，自动适配，仅查残留旧名）
- [x] Task 4: 文档同步
  - [x] `tools/doc.go`：工具清单与构造器列表（`NewCopyItem` 等）同步
  - [x] `SUBAGENTS.md`：白名单示例（L66-67）同步
  - [x] `AGENTS.md`：长期记忆 20 条工具清单 + 临时记忆 5 中的旧名引用改为新名（保留改名说明）
- [x] Task 5: 验证（gofmt/build/vet/lint/test 通过；全仓 grep 发现注释/文档残留，转入 Task 6）
  - [x] `gofmt -l` 相关文件干净
  - [x] `go build ./...` + `go vet ./internal/agent/...` 通过
  - [x] `golangci-lint.exe run ./...` 0 issues
  - [x] `go test ./internal/agent/...` 全量全绿
  - [x] 全仓 grep `delete_file|copy_file|move_file|transfer_file` 无残留（AGENTS.md 改名说明行除外）
- [x] Task 6: 修复验证发现的注释/文档残留（EVENTS.md / fs_base.go / mkdir_dir.go / ai-chat.js）
  - [x] `EVENTS.md`：L19（审批事件表）、L53（os_agent 工具清单）、L93/L96（tool/critical 字段说明）、L105-107（各工具审批分级）旧工具名改新名
  - [x] `tools/fs_base.go` L25 注释 `transfer_file` → `transfer_item`
  - [x] `tools/mkdir_dir.go` L5/L10 注释 `delete_file/copy_file/move_file` → `delete_item/copy_item/move_item`
  - [x] `frontend/src/js/ai-chat.js` `APPROVAL_TOOL_LABEL` 映射 `delete_file: '删除文件'` → `delete_item: '删除项'`（改名后原键失效，避免审批面板回落显示英文工具名）
  - [x] 修复后全仓 grep 复验无残留（仅 `.trae/` 历史文档与 spec 记录保留）

# Task Dependencies
- Task 2 依赖 Task 1（构造器存在才能注册）
- Task 3 依赖 Task 1
- Task 4 依赖 Task 1/2
- Task 5 依赖 Task 1/2/3/4
