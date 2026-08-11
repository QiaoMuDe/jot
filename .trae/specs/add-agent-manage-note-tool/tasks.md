# Tasks

- [x] Task 1: 依赖装配（Deps + app.go 传参）：[agent.go](internal/agent/agent.go#L44-L54) 的 `Deps` 新增 `Note *services.NoteService` 字段；[app.go](app.go) 的 `NewAgentService` 构造处与 `rebuildServices` 末尾重建 AgentSvc 处注入 `NoteService` 实例（与 Todo/Notebook/Tag 同一实例来源）。
- [x] Task 2: 实现 [internal/agent/tools/manage_note.go](internal/agent/tools/manage_note.go)（新建）：结构体 `manageNoteTool`（依赖 `note *services.NoteService`、`tag *services.TagService`、`setting *services.SettingService`、`ctx *Context`）+ `var _ tool.InvokableTool` 断言 + `NewManageNote` 构造器 + `Info()`（action 枚举：create/list/view/pin/move/add_tag/remove_tag，参数 Schema 完整，Desc 说明与 recall_notes 边界）+ `InvokableRun()`（按 action 分发，动作分发前 `ctx.Err()` 取消检查）：
  - `create`：title/content 必填；file_ext 缺省 `.md`；notebook_id > 0 时走 `CreateWithNotebook`，否则 `Create`；tag_ids 非空时逐个 `tag.AddTagToNote`；返回"已创建笔记 [数字]：标题"
  - `list`：调用 `note.Search(keyword, page, pageSize, sortBy, startDate, endDate, tagIDs)`；page<1→1、pageSize<=0→10、>50→50；返回"共 n 条、第 x/y 页" + `[数字]` 编号 + 标题 + 预览（前 200 字符）+ 标签 + 置顶标记，页尾提示可翻页
  - `view`：id 必填正整数；`note.GetNoteContent(id)`；读取设置 `ai_large_file_preview_threshold`（缺省 10000）截断 Runes，超过时末尾提示"内容过长已截断，如需继续阅读可要求分段查看"
  - `pin`：id 必填正整数；`note.TogglePin(id)`；返回切换后的置顶状态
  - `move`：id、notebook_id 必填正整数；`note.MoveToNotebook(id, notebookID)`
  - `add_tag` / `remove_tag`：id、tag_id 必填正整数；分别调 `tag.AddTagToNote` / `tag.RemoveTagFromNote`
- [x] Task 3: 注册与文档：[registry.go](internal/agent/registry.go#L19-L28) 追加 `tools.WrapWithError("manage_note", tools.NewManageNote(p.deps.Note, p.deps.Tag, p.deps.Setting, p.ctx), p.ctx)`；更新 [internal/agent/TOOLS.md](internal/agent/TOOLS.md) 工具清单表、[internal/agent/tools/doc.go](internal/agent/tools/doc.go) 与 [internal/agent/doc.go](internal/agent/doc.go) 清单（含 Deps 说明）。
- [x] Task 4: 验证：`go build ./...`、`go vet ./internal/agent/...` 通过；重启应用后 Agent 模式手动触发一次调用（create → list → view → pin → add_tag/remove_tag → move）验证动作可用、错误回填正常、rebuildServices 后工具不失效。（编译/静态验证已完成；Agent 模式手动触发需用户运行应用验证）

- [x] Task 5（修复项）：list 动作补 notebook_id 过滤——[manage_note.go](internal/agent/tools/manage_note.go) 的 `listNotes` 增加 notebook_id 参数（notebook_id>0 时改调 `note.SearchByNotebook`，否则 `note.Search`）；`InvokableRun` list 分支把 `args.NotebookID` 传入；`Info()` 中 notebook_id 参数 Desc 补充"list 过滤"用途。改完 `go build ./...` 通过。

# Task Dependencies

- [Task 2] depends on [Task 1]（工具构造器需要 Deps 提供 NoteService）
- [Task 3] depends on [Task 2]（注册与文档描述实际实现的工具）
- [Task 4] depends on [Task 3]
