# Tasks

- [x] Task 1: 基础设施改造（[internal/agent/tools/context.go](internal/agent/tools/context.go)）
  - [x] 定义可选接口 `ActionTextProvider`：`ActionText(argumentsInJSON string) string`
  - [x] `Record` 新增字段 `ActionText string`（json tag `action_text,omitempty`）
  - [x] 将 `WrapWithError` 改为自定义 wrapper 结构体：`Info()`/`InvokableRun()` 委托给内层工具，**保持现有失败回填、`tool_error` 事件、用户取消不误报行为逐字不变**（照抄原 handler 逻辑）；wrapper 实现 `ActionTextProvider` 转发给内层工具（未实现返回空串）
  - [x] 验证：`go build ./...` 通过

- [x] Task 2: 父包下发 action_text（[internal/agent/agent.go](internal/agent/agent.go)）
  - [x] `Run()` 中构建 `map[string]tool.BaseTool`（name → 工具，遍历 toolList 取 `Info().Name`）
  - [x] `emitToolStart` 增加工具映射参数，按 `tc.Function.Name` 查映射并断言 `tools.ActionTextProvider`，调用 `ActionText(tc.Function.Arguments)` 填入 `Record.ActionText`；两处调用点（L190、L204）同步传参
  - [x] 验证：`go build ./...`、`go vet ./internal/agent/...` 通过

- [x] Task 3: 8 个工具各自实现 `ActionText`（可并行，文案按 spec.md 映射表）
  - [x] `web_search.go`：解析 query → `搜索 {query}` / `搜索互联网`
  - [x] `recall_notes.go`：解析 notebook_ids 数量 → `检索 n 个笔记本` / `检索本地笔记`
  - [x] `refine_query.go`：返回 `精炼搜索关键词`
  - [x] `current_time.go`：返回 `获取当前日期时间`
  - [x] `manage_todo.go`：create/list/toggle/update → `创建待办`/`列出待办`/`更新待办状态`/`修改待办文本`
  - [x] `manage_notebook.go`：create/rename/list → `创建笔记本`/`重命名笔记本`/`列出笔记本`
  - [x] `manage_tag.go`：create/list/update → `创建标签`/`列出标签`/`更新标签`
  - [x] `manage_note.go`：create/list/view/pin/move/add_tag/remove_tag → `创建笔记`/`列出笔记`/`查看笔记全文`/`置顶或取消置顶笔记`/`移动笔记`/`添加标签`/`移除标签`
  - [x] 验证：`go build ./...`、`go vet ./internal/agent/...` 通过

- [x] Task 4: 前端删除 switch 读 action_text（[frontend/src/js/ai-chat.js](frontend/src/js/ai-chat.js)）
  - [x] `showToolStatusStart` 中 action 改为 `const action = payload.action_text || '执行';`
  - [x] 删除全部工具名 switch 分支（web_search/recall_notes/refine_search_query/get_current_time/manage_todo/manage_notebook/manage_tag）
  - [x] 验证：`npm run build` 通过

- [x] Task 5: 更新工具开发维护规范（[internal/agent/TOOLS.md](internal/agent/TOOLS.md)）
  - [x] §2 第 5 步"按需更新前端动作文案"改为"可选在工具实现内维护动作文案"（实现 `ActionTextProvider`，无需改前端）
  - [x] §3 注册要点补一句：动作文案由工具实现 `ActionTextProvider` 提供，父包自动下发
  - [x] §5.1 维护既有工具、§5.3 自查清单同步更新（新增"可选实现 ActionTextProvider"项）
  - [x] §8 重写：工具动作文案在每个工具实现内维护（前端只读 `payload.action_text`，回退"执行"），删除前端 switch 维护说明
  - [x] §6 现有工具清单保持工具名/文件/依赖列不变

# Task Dependencies

- [Task 2] depends on [Task 1]（父包断言 wrapper 的 ActionTextProvider，需 wrapper 先实现）
- [Task 3] depends on [Task 1]（ActionTextProvider 接口定义）
- [Task 4] 与后端任务无依赖，可并行
- [Task 5] depends on [Task 1]-[Task 4]（文档描述最终行为）
- 验证放最后统一执行：`go build ./...`、`go vet ./internal/agent/...`、`npm run build`
