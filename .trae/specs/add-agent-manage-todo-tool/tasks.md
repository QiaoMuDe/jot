# Tasks

- [x] Task 1: 实现 manage_todo 工具（internal/agent/tools/manage_todo.go）
  - 新建文件 manage_todo.go，实现 `manage_todo` 工具：
    - 结构体 `manageTodoTool{todo *services.TodoService; ctx *Context}`，编译期断言 `var _ tool.InvokableTool = (*manageTodoTool)(nil)`
    - `Info()`：Name=`manage_todo`；Desc 说明创建/列出/勾选待办及 action 枚举；参数 Schema：
      - `action`（string 必填，Enum create/list/toggle）
      - `text`（string 可选，create 时必填）
      - `status`（string 可选，Enum active/done/all，list 用，缺省 active）
      - `id`（number 可选，toggle 时必填）
    - `InvokableRun`：解析参数 → 校验 action → 分发：
      - create：text 非空校验 → `todo.Create(text)` → 返回"已创建待办 #ID：text（未完成）"
      - list：按 status 对 `todo.List()` 结果过滤（active: Done==false / done: Done==true / all: 全部）→ 格式化列表（每条 `[ID] 文本（未完成/已完成）· 创建时间`）+ 统计"共 N 条（未完成 x / 已完成 y）"；空列表返回"当前没有待办"
      - toggle：id>0 校验 → `todo.Toggle(id)` → 返回"待办 #ID：text 已标记为完成 / 已恢复为未完成"
      - 非法 action / 缺必填 → 返回 error（WrapWithError 回填模型）
    - 构造器 `NewManageTodo(todo *services.TodoService, ctx *Context) tool.InvokableTool`
- [x] Task 2: 依赖注入与注册
  - `internal/agent/agent.go`：`Deps` 增加 `Todo *services.TodoService` 字段
  - `internal/agent/registry.go`：`buildTools` 增加 `tools.WrapWithError("manage_todo", tools.NewManageTodo(p.deps.Todo, p.ctx), p.ctx)`
  - `app.go`：`NewAgentService` 的 Deps 传入 `Todo: todoService`（app.go 已有该变量）
- [x] Task 3: 更新工具清单文档
  - `internal/agent/tools/doc.go`：工具清单补 `manage_todo` 与构造器 `NewManageTodo`
  - `internal/agent/TOOLS.md`：§6 工具清单表补 `manage_todo` 行（依赖注入 `todo`、功能说明、结构化收集"无"）；§4.1 无依赖类型说明需同步（如提及 TodoService 属 services 包）
- [x] Task 4: 前端动作文案分支（可选，保持 §8 风格）
  - `frontend/src/js/ai-chat.js` 的 `showToolStatusStart` 增加 `manage_todo` 分支：解析 `args.action` → 显示 `调用「manage_todo」工具：创建待办 / 列出待办 / 更新待办状态`，未匹配 action 回退默认"执行"
- [x] Task 5: 验证
  - `go build ./...`、`go vet ./internal/agent/...` 通过
  - 如 Task 4 完成，`npm run build` 通过

# Task Dependencies
- Task 2 依赖 Task 1（先有工具实现与构造器，才能注入注册）
- Task 3 依赖 Task 1、Task 2（清单需登记最终的工具与依赖）
- Task 4 依赖 Task 1（动作分支基于工具名/参数）；可与 Task 2、3 并行
- Task 5 依赖全部 Task
