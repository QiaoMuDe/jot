# Checklist

- [x] [manage_note.go](internal/agent/tools/manage_note.go) 已创建，文件头注释说明工具职责与实现要点
- [x] `manageNoteTool` 实现 `tool.InvokableTool`，含 `var _ tool.InvokableTool = (*manageNoteTool)(nil)` 编译期断言
- [x] `Info()` 的 action 枚举（create/list/view/pin/move/add_tag/remove_tag）与参数 Schema（Required 标记、枚举、ElemInfo）完整且与 `InvokableRun` case 分发同步
- [x] `Info().Desc` 说明何时调用以及与 `recall_notes` 的边界（知识召回 vs 笔记库操作）
- [x] create：title/content 必填校验；file_ext 缺省 `.md`；notebook_id>0 走 `CreateWithNotebook`；tag_ids 非空逐个打标签；返回 `[数字]` 编号
- [x] list：调用 `Search`（notebook_id>0 时 `SearchByNotebook`），支持 keyword/notebook_id/tag_ids/start_date/end_date/sort_by/page/page_size 过滤分页；分页校验（page<1→1、pageSize<=0→10、>50→50）；返回"共 n 条、第 x/y 页" + `[数字]` 编号 + 标题 + 预览 + 标签 + 置顶标记，页尾提示可翻页
- [x] view：id 必填正整数；`GetNoteContent` 取全文；按 `ai_large_file_preview_threshold`（缺省 10000）Runes 截断并提示"内容过长已截断"
- [x] pin：id 必填正整数，`TogglePin` 切换并返回状态
- [x] move：id、notebook_id 必填正整数，`MoveToNotebook`
- [x] add_tag/remove_tag：id、tag_id 必填正整数，分别调 `AddTagToNote`/`RemoveTagFromNote`
- [x] 动作分发前执行 `ctx.Err()` 取消检查
- [x] [registry.go](internal/agent/registry.go#L19-L28) 已注册 `manage_note`（`WrapWithError` 包装）
- [x] [agent.go](internal/agent/agent.go#L44-L54) `Deps` 新增 `Note *services.NoteService`
- [x] [app.go](app.go) `NewAgentService` 构造与 `rebuildServices` 末尾重建 AgentSvc 均传入 `NoteService`
- [x] 文档更新：[TOOLS.md](internal/agent/TOOLS.md) 工具清单表、[tools/doc.go](internal/agent/tools/doc.go)、[doc.go](internal/agent/doc.go)
- [x] `go build ./...` 与 `go vet ./internal/agent/...` 通过
- [ ] Agent 模式手动触发 create → list → view → pin → add_tag/remove_tag → move 全链路可用（需用户运行 Wails 应用 + 真实 AI 模型验证）
