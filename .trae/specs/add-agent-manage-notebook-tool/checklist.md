# Checklist
- [x] `internal/agent/tools/manage_notebook.go` 存在，实现 `manage_notebook` 工具，含 `Info()` / `InvokableRun()` / `NewManageNotebook()` 与 `var _ tool.InvokableTool` 断言
- [x] 工具参数校验完备：action 白名单；create/rename 的 name trim 非空校验；rename 的 id 正整数校验；ctx.Err() 取消检查
- [x] 三个动作调用正确的 NotebookService 方法：create→Create、rename→Update、list→GetAll，返回文案符合 spec 格式
- [x] `agent.Deps` 增加 `Notebook *services.NotebookService`，`registry.go` 注册 `manage_notebook`（WrapWithError 包装），`app.go` NewApp 传入 `Notebook: notebookService`
- [x] `tools/doc.go`、`agent/doc.go`、`TOOLS.md` §6 工具清单（及 §3 注册示例）已登记 `manage_notebook` / `NewManageNotebook`（`agent/doc.go` 因工具清单限定"只读工具"，与 manage_todo 处理一致，维持现状）
- [x] 前端 `showToolStatusStart` 有 `manage_notebook` 分支（create/rename/list 动作文案，未匹配 action 走默认"执行"）
- [x] `go build ./...`、`go vet ./internal/agent/...` 通过
