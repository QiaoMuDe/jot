# Tasks
- [x] Task 1: 实现 manage_notebook 工具（internal/agent/tools/manage_notebook.go）
  - [x] SubTask 1.1: 新建 manage_notebook.go，文件头注释说明工具职责（创建/重命名/列出笔记本）与实现要点（复用 NotebookService，不感知父包事件循环）
  - [x] SubTask 1.2: 定义 manageNotebookTool 结构体（notebook *services.NotebookService、ctx *Context）与 `var _ tool.InvokableTool` 编译期断言
  - [x] SubTask 1.3: 实现 Info()：Name=manage_notebook；Desc 说明三种动作及参数（action/name/id）；ParamsOneOf 参数 Schema（action 枚举 create/rename/list 必填；name 字符串 create/rename 必填；id 数字 rename 必填）
  - [x] SubTask 1.4: 实现 InvokableRun()：json.Unmarshal 解析参数 → action 白名单校验 → ctx.Err() 取消检查 → 按 action 分发：
    - create：name trim 非空校验 → NotebookService.Create → 返回"已创建笔记本 #<id>：<name>"
    - rename：id>0 且 name trim 非空校验 → NotebookService.Update → 返回"已重命名笔记本 #<id> 为：<name>"
    - list：NotebookService.GetAll → 空返回"当前没有任何笔记本"；否则逐条 `[<id>] <名称> · 创建时间 <2006-01-02 15:04>`
  - [x] SubTask 1.5: 提供导出构造器 `NewManageNotebook(notebook *services.NotebookService, ctx *Context) tool.InvokableTool`（命名与既有构造器一致）
- [x] Task 2: 接线注册（依赖 Task 1 文件存在）
  - [x] SubTask 2.1: internal/agent/agent.go 的 Deps 增加字段 `Notebook *services.NotebookService // Notebook 笔记本服务（manage_notebook 工具使用）`
  - [x] SubTask 2.2: app.go 的 NewApp 构造 NewAgentService 处增加 `Notebook: notebookService`
  - [x] SubTask 2.3: internal/agent/registry.go 的 buildTools 追加 `tools.WrapWithError("manage_notebook", tools.NewManageNotebook(p.deps.Notebook, p.ctx), p.ctx)`
- [x] Task 3: 文档同步（可与 Task 1/2 并行）
  - [x] SubTask 3.1: internal/agent/tools/doc.go 工具清单与构造器名补充 manage_notebook / NewManageNotebook
  - [x] SubTask 3.2: internal/agent/doc.go 结构说明处（如工具清单描述）补充 manage_notebook（注：该文件工具清单限定"只读工具"，与写操作工具 manage_todo 处理一致，维持现状不修改）
  - [x] SubTask 3.3: internal/agent/TOOLS.md §6 工具清单表补 manage_notebook 行（依赖注入 `notebook`、功能说明、结构化收集"无"）；§3 注册示例代码块同步补 manage_todo 与 manage_notebook 两行（修复既有滞后示例）
- [x] Task 4: 前端动作文案（可与 Task 1/2 并行）
  - [x] SubTask 4.1: frontend/src/js/ai-chat.js 的 showToolStatusStart 增加 `manage_notebook` 分支：解析 args.action → create"创建笔记本" / rename"重命名笔记本" / list"列出笔记本"，未匹配走默认"执行"（样式与 manage_todo 分支一致）
- [x] Task 5: 验证
  - [x] SubTask 5.1: `go build ./...` 通过
  - [x] SubTask 5.2: `go vet ./internal/agent/...` 通过

# Task Dependencies
- [Task 2] depends on [Task 1]（注册需工具文件与构造器存在）
- [Task 3]、[Task 4] 可与 [Task 1] [Task 2] 并行（纯文档/前端改动）
- [Task 5] 在所有代码改动完成后执行
