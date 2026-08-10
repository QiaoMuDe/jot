# Checklist

## 功能验收
- [x] `internal/agent/tools/manage_todo.go` 存在，实现 `manage_todo` 工具，含 `Info()` / `InvokableRun()` / `NewManageTodo()` 与 `var _ tool.InvokableTool` 断言
- [x] `Info()` 定义 action（Enum create/list/toggle）、text、status（Enum active/done/all）、id 参数，Desc 说明三种操作与参数含义
- [x] action=create：text 非空时创建待办并返回含 ID/文本/状态的文本结果；text 为空返回错误
- [x] action=list：status=active 只返回未完成、done 只返回已完成、all 返回全部；返回含 ID 的格式化列表与统计；空列表有明确提示
- [x] action=toggle：有效 id 切换完成状态并返回"已完成/已恢复未完成"文案；非法 id 或 id<=0 返回错误
- [x] 非法 action 返回明确错误（不中断 ReAct 循环，经 WrapWithError 回填模型）
- [x] `agent.Deps` 增加 `Todo *services.TodoService`，`registry.go` 注册 `manage_todo`（WrapWithError 包装），`app.go` 传入 `Todo: todoService`
- [x] `tools/doc.go` 与 `internal/agent/TOOLS.md` 工具清单登记 `manage_todo`
- [x] 前端 `showToolStatusStart` 按 §8 风格展示 manage_todo 动作文案（若 Task 4 实施）

## 代码质量
- [x] 遵循 tools 子包规范：不 import 父包、不直接 emit、依赖构造器注入
- [x] 无新增 service API（列表过滤在工具内完成，未改动 TodoService 现有方法）
- [x] `go build ./...`、`go vet ./internal/agent/...` 通过（如动前端则 `npm run build` 通过）
